package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"kankor-backend/models"
)

// GeminiService handles interactions with Google Gemini AI API
type GeminiService struct {
	apiKey             string
	httpClient         *http.Client
	visionHTTPClient   *http.Client // Longer timeout for vision API (large image payloads)
	baseURL            string
	fallbackURLs       []string // Fallback models if primary is overloaded
	visionFallbackURLs []string // Pro-tier fallback chain for vision/hybrid subjects
}

// Available Gemini models in priority order (most reliable first)
// Gemini 2.0 & 2.5 are deprecated/restricted for new API keys as of July 2026.
var geminiModels = []string{
	"gemini-3.6-flash",      // Primary — newest, released July 2026
	"gemini-3.5-flash",      // Fallback 1 — stable, released May 2026
	"gemini-3.5-flash-lite", // Fallback 2 — lite, fast, released July 2026
}

// geminiVisionModels adds a Pro-tier model for visual reasoning.
// All subjects now use vision/hybrid extraction (see IsVisionSubject).
var geminiVisionModels = []string{
	"gemini-3.1-pro-preview", // Pro-tier — best visual reasoning for math/science
	"gemini-3.6-flash",       // Flash fallback 1
	"gemini-3.5-flash",       // Flash fallback 2
	"gemini-3.5-flash-lite",  // Flash fallback 3
}

// NewGeminiService creates a new Gemini service instance
func NewGeminiService(apiKey string) *GeminiService {
	// Check for proxy configuration
	httpClient := &http.Client{
		Timeout: 60 * time.Second, // 60 second timeout for text AI requests
	}

	// Vision API needs longer timeout (large image payloads take time to upload + process)
	visionHTTPClient := &http.Client{
		Timeout: 180 * time.Second, // 3 minute timeout for vision requests
	}

	// Use HTTP proxy if configured
	proxyURL := os.Getenv("HTTP_PROXY")
	if proxyURL == "" {
		proxyURL = os.Getenv("http_proxy")
	}

	if proxyURL != "" {
		fmt.Printf("[GeminiService] Using HTTP proxy: %s\n", proxyURL)
		proxy, err := url.Parse(proxyURL)
		if err == nil {
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
			httpClient.Transport = transport
			visionHTTPClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
		}
	}

	return &GeminiService{
		apiKey:             apiKey,
		httpClient:         httpClient,
		visionHTTPClient:   visionHTTPClient,
		baseURL:            fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", geminiModels[0]),
		fallbackURLs:       buildFallbackURLs(geminiModels[1:]),
		visionFallbackURLs: buildVisionFallbackURLs(),
	}
}

// buildFallbackURLs creates URLs for backup models
func buildFallbackURLs(models []string) []string {
	urls := make([]string, len(models))
	for i, model := range models {
		urls[i] = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	}
	return urls
}

// buildVisionFallbackURLs creates URLs for the Pro-tier vision model chain.
// The first model (gemini-3.1-pro-preview) replaces the default base URL for vision calls.
func buildVisionFallbackURLs() []string {
	return buildFallbackURLs(geminiVisionModels[1:])
}

// ExtractQuestions sends text to Gemini AI and extracts structured questions
func (gs *GeminiService) ExtractQuestions(text string) ([]models.GeminiExtractedQuestion, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text provided for question extraction")
	}

	// Safety limit: Max 50,000 characters
	if len(text) > 50000 {
		return nil, fmt.Errorf("text exceeds maximum limit of 50,000 characters (got %d)", len(text))
	}

	fmt.Printf("[GeminiService] Processing text of %d characters\n", len(text))

	// DEBUG: Log first 500 chars of text before sending to Gemini
	if len(text) > 0 {
		previewLen := 500
		if len(text) < previewLen {
			previewLen = len(text)
		}
		fmt.Printf("[GeminiService] DEBUG - Text preview (first %d chars):\n% s\n", previewLen, text[:previewLen])
	}

	var allQuestions []models.GeminiExtractedQuestion
	var lastErr error

	// Strategy: Check text length
	if len(text) < 6000 {
		// Small text: Send directly
		fmt.Println("[GeminiService] Text < 6000 chars, sending directly")
		allQuestions, lastErr = gs.callGeminiWithRetry(text)
		if lastErr != nil {
			// If model says no parseable content, return empty (vision pipeline will handle it)
			if strings.Contains(lastErr.Error(), "no parseable MCQ") {
				fmt.Println("[GeminiService] No MCQ content found in text — returning 0 questions")
				return nil, nil
			}
			return nil, lastErr
		}
	} else {
		// Large text: Split into chunks
		fmt.Println("[GeminiService] Text >= 6000 chars, splitting into chunks")
		chunks := gs.splitTextIntoChunks(text)
		fmt.Printf("[GeminiService] Split into %d chunks\n", len(chunks))

		for i, chunk := range chunks {
			fmt.Printf("[GeminiService] Processing chunk %d/%d (%d chars)\n", i+1, len(chunks), len(chunk))

			questions, err := gs.callGeminiWithRetry(chunk)
			if err != nil {
				// If model says no parseable content, skip this chunk gracefully
				if strings.Contains(err.Error(), "no parseable MCQ") {
					fmt.Printf("[GeminiService] Chunk %d has no parseable content, skipping\n", i+1)
					continue
				}
				// Otherwise hard failure — stop immediately
				return nil, fmt.Errorf("extraction failed at chunk %d: %w. Please upload a smaller or cleaner PDF", i+1, err)
			}

			allQuestions = append(allQuestions, questions...)
			fmt.Printf("[GeminiService] Extracted %d questions from chunk %d\n", len(questions), i+1)

			// Add 2 second delay between chunks (except last one)
			if i < len(chunks)-1 {
				fmt.Println("[GeminiService] Waiting 2 seconds before next chunk...")
				time.Sleep(2 * time.Second)
			}
		}
	}

	// Validate extracted questions
	if err := gs.validateQuestions(allQuestions); err != nil {
		return nil, fmt.Errorf("invalid questions extracted: %w", err)
	}

	fmt.Printf("[GeminiService] Successfully extracted %d total questions\n", len(allQuestions))
	return allQuestions, nil
}

// splitTextIntoChunks splits text into chunks of 3000-5000 characters
func (gs *GeminiService) splitTextIntoChunks(text string) []string {
	const (
		minChunkSize = 3000
		maxChunkSize = 5000
	)

	var chunks []string
	textLen := len(text)

	if textLen <= maxChunkSize {
		return []string{text}
	}

	start := 0
	for start < textLen {
		end := start + maxChunkSize
		if end >= textLen {
			// Last chunk
			chunks = append(chunks, text[start:])
			break
		}

		// Try to find a sentence boundary
		chunk := text[start:end]

		// Look for sentence endings (Persian/English)
		boundary := strings.LastIndexAny(chunk, "。.!؟\n")
		if boundary != -1 && (start+boundary) > (start+minChunkSize) {
			// Found a good boundary after minimum chunk size
			end = start + boundary + 1
			chunk = text[start:end]
		}

		chunks = append(chunks, strings.TrimSpace(chunk))
		start = end
	}

	return chunks
}

// callGeminiWithRetry calls Gemini API with model fallback and exponential backoff.
// 429 (rate limit): Exponential backoff on same model, then try next with increasing delay.
// 503 (overloaded): Switch to fallback model immediately.
func (gs *GeminiService) callGeminiWithRetry(text string) ([]models.GeminiExtractedQuestion, error) {
	prompt := gs.buildPrompt(text)

	allURLs := append([]string{gs.baseURL}, gs.fallbackURLs...)

	var lastErr error
	consecutive429s := 0 // Track consecutive rate limits for escalating backoff
	totalAttempts := 0   // Actual number of API calls made

	for _, modelURL := range allURLs {
		if totalAttempts > 0 {
			// Calculate delay based on rate limit history
			var delay time.Duration
			if consecutive429s > 0 {
				// Exponential backoff: 5s, 10s, 20s, 40s (capped at 60s)
				delay = time.Duration(5*(1<<uint(consecutive429s-1))) * time.Second
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
				fmt.Printf("[GeminiService] Rate-limited %d times, waiting %v before trying model '%s'...\n",
					consecutive429s, delay, extractModelName(modelURL))
			} else {
				delay = 2 * time.Second
				fmt.Printf("[GeminiService] Switching to fallback model: %s\n", extractModelName(modelURL))
			}
			time.Sleep(delay)
		}

		questions, err := gs.callGeminiAPIWithURL(prompt, modelURL)
		if err == nil {
			if consecutive429s > 0 {
				fmt.Printf("[GeminiService] Succeeded after %d rate-limit retries\n", consecutive429s)
			}
			return questions, nil
		}

		fmt.Printf("[GeminiService] Attempt %d (model '%s') failed: %v\n",
			totalAttempts+1, extractModelName(modelURL), err)
		lastErr = err
		totalAttempts++

		if strings.Contains(err.Error(), "status 429") {
			// Rate limit: escalate backoff, try next model
			consecutive429s++
		} else if strings.Contains(err.Error(), "status 503") ||
			strings.Contains(err.Error(), "status 404") ||
			strings.Contains(err.Error(), "status 403") ||
			strings.Contains(err.Error(), "API request failed") ||
			strings.Contains(err.Error(), "failed to parse questions") ||
			strings.Contains(err.Error(), "no parseable MCQ") ||
			strings.Contains(err.Error(), "model reported") {
			// Transient or model unavailable, or model produced bad output: try next model, reset 429 counter
			consecutive429s = 0
		} else {
			// Non-retryable error (auth, bad request): stop immediately
			break
		}
	}

	return nil, fmt.Errorf("extraction failed after %d attempts across %d models: %w", totalAttempts, len(allURLs), lastErr)
}

// callGeminiAPIWithURL makes the API call to a specific Gemini model endpoint
func (gs *GeminiService) callGeminiAPIWithURL(prompt string, baseURL string) ([]models.GeminiExtractedQuestion, error) {
	// Build request payload
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 8192,
		},
	}

	jsonPayload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL with API key
	url := fmt.Sprintf("%s?key=%s", baseURL, gs.apiKey)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := gs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("API returned status 429: rate limit exceeded. Please try again in a few minutes")
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response
	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	content := geminiResp.Candidates[0].Content
	if len(content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in response content")
	}

	responseText := content.Parts[0].Text

	// DEBUG: Log Gemini response preview
	if len(responseText) > 0 {
		previewLen := 500
		if len(responseText) < previewLen {
			previewLen = len(responseText)
		}
		fmt.Printf("[GeminiService] DEBUG - Response preview (first %d chars):\n% s\n", previewLen, responseText[:previewLen])
	}

	// Parse JSON from response
	questions, err := gs.parseQuestionsFromText(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse questions: %w", err)
	}

	fmt.Printf("[GeminiService] Successfully parsed %d questions from Gemini response\n", len(questions))
	return questions, nil
}

// callGeminiAPI is kept for backward compatibility, delegates to primary model
func (gs *GeminiService) callGeminiAPI(prompt string) ([]models.GeminiExtractedQuestion, error) {
	return gs.callGeminiAPIWithURL(prompt, gs.baseURL)
}

// extractModelName extracts the model name from a Gemini API URL
func extractModelName(url string) string {
	// URL format: .../models/MODEL_NAME:generateContent
	if idx := strings.Index(url, "models/"); idx != -1 {
		rest := url[idx+7:] // after "models/"
		if colonIdx := strings.Index(rest, ":"); colonIdx != -1 {
			return rest[:colonIdx]
		}
		return rest
	}
	return "unknown"
}

// parseQuestionsFromText extracts JSON array from Gemini's response text
func (gs *GeminiService) parseQuestionsFromText(text string) ([]models.GeminiExtractedQuestion, error) {
	// Try to find JSON object/array in the response
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := text[startIdx : endIdx+1]

	// Try to parse as structured response with "questions" array
	var structuredResp GeminiQuestionsResponse
	if err := json.Unmarshal([]byte(jsonStr), &structuredResp); err == nil {
		// Model explicitly said input is invalid/unparseable → not an error, just no content
		if structuredResp.Invalid {
			fmt.Println("[GeminiService] Model returned 'invalid: true' — no parseable MCQ content in this text")
			return nil, fmt.Errorf("no parseable MCQ content found in provided text")
		}
		if structuredResp.Error != "" {
			fmt.Printf("[GeminiService] Model returned error: %s\n", structuredResp.Error)
			return nil, fmt.Errorf("model reported: %s", structuredResp.Error)
		}
		if len(structuredResp.Questions) > 0 {
			return gs.normalizeQuestions(structuredResp.Questions), nil
		}
		// Valid structured response with empty questions — success, just no content
		fmt.Println("[GeminiService] Structured response parsed with 0 questions")
		return nil, nil
	}

	// If structured parse failed, try to find just the questions array within the text.
	// e.g. response might be [...] without wrapping {"questions": ...} object.
	arrStart := strings.Index(text, "[")
	arrEnd := strings.LastIndex(text, "]")
	if arrStart != -1 && arrEnd != -1 && arrStart < arrEnd {
		arrStr := text[arrStart : arrEnd+1]
		var questions []models.GeminiExtractedQuestion
		if err := json.Unmarshal([]byte(arrStr), &questions); err == nil && len(questions) > 0 {
			return gs.normalizeQuestions(questions), nil
		}
	}

	// Fallback: try to parse entire extracted JSON as array (legacy format)
	var questions []models.GeminiExtractedQuestion
	if err := json.Unmarshal([]byte(jsonStr), &questions); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w. Raw text (first 200 chars): %s", err, truncateForLog(text, 200))
	}

	return gs.normalizeQuestions(questions), nil
}

// normalizeQuestions converts different question formats to standard format
func (gs *GeminiService) normalizeQuestions(questions []models.GeminiExtractedQuestion) []models.GeminiExtractedQuestion {
	normalized := make([]models.GeminiExtractedQuestion, 0, len(questions))

	for _, q := range questions {
		// Skip invalid questions
		if q.Invalid {
			continue
		}

		normalizedQ := models.GeminiExtractedQuestion{}

		// Handle question text (prefer new format, fallback to legacy)
		if q.QuestionText != "" {
			normalizedQ.QuestionText = q.QuestionText
			normalizedQ.Question = q.QuestionText // For backward compatibility
		} else {
			normalizedQ.Question = q.Question
			normalizedQ.QuestionText = q.Question
		}

		// Handle options (prefer new format, fallback to legacy)
		if q.OptionA != "" || q.OptionB != "" || q.OptionC != "" || q.OptionD != "" {
			// New format: option_a, option_b, option_c, option_d
			normalizedQ.OptionA = q.OptionA
			normalizedQ.OptionB = q.OptionB
			normalizedQ.OptionC = q.OptionC
			normalizedQ.OptionD = q.OptionD

			// Also populate legacy Options map for compatibility
			normalizedQ.Options = map[string]string{
				"A": q.OptionA,
				"B": q.OptionB,
				"C": q.OptionC,
				"D": q.OptionD,
			}
		} else if q.Options != nil && len(q.Options) == 4 {
			// Legacy format: Options map
			normalizedQ.Options = q.Options
			normalizedQ.OptionA = q.Options["A"]
			normalizedQ.OptionB = q.Options["B"]
			normalizedQ.OptionC = q.Options["C"]
			normalizedQ.OptionD = q.Options["D"]
		} else {
			// Invalid question - skip
			fmt.Printf("[GeminiService] Skipping question with invalid options: %s\n", q.QuestionText)
			continue
		}

		// Carry through correct_option — now always cleared; admin sets it manually
		normalizedQ.CorrectOption = ""

		normalized = append(normalized, normalizedQ)
	}

	return normalized
}

// validateQuestions ensures all questions have exactly 4 options
func (gs *GeminiService) validateQuestions(questions []models.GeminiExtractedQuestion) error {
	for i, q := range questions {
		if q.Question == "" {
			return fmt.Errorf("question %d has empty text", i+1)
		}

		if q.Options == nil || len(q.Options) != 4 {
			return fmt.Errorf("question %d must have exactly 4 options, got %d", i+1, len(q.Options))
		}

		// Check that all required options exist
		for _, key := range []string{"A", "B", "C", "D"} {
			if _, exists := q.Options[key]; !exists {
				return fmt.Errorf("question %d missing option %s", i+1, key)
			}
		}
		// correct_option is no longer validated — admin sets it manually after review
	}

	return nil
}

// buildPrompt creates the prompt for Gemini API (text-only extraction).
func (gs *GeminiService) buildPrompt(text string) string {
	return fmt.Sprintf(`You are a precise MCQ extraction system for Kankor exam PDFs.

CRITICAL RULES:
1. You are NOT allowed to change, rewrite, translate, summarize, or improve any text
   — with TWO exceptions: fix garbled Persian/Dari RTL text, and convert math to LaTeX.
2. ALL mathematical expressions MUST be converted to LaTeX wrapped in $...$ delimiters.
   Non-math Persian/Dari text must stay outside $...$ and remain untouched.
----------------------------------------
INPUT:
You will receive raw text extracted from a PDF. The text may contain mathematical
content (equations, formulas, symbols, fractions, etc.) alongside regular text.
The PDF text extraction may produce Persian/Dari text with reversed or garbled
character order due to RTL encoding issues in the PDF.

YOUR TASK:
1. Extract Multiple Choice Questions (MCQs)
2. Each question must have:
   - question_text (with ALL math in LaTeX $...$, Persian text readable RTL)
   - option_a, option_b, option_c, option_d (same LaTeX + RTL rules)

STRICT RULES (VERY IMPORTANT):
- DO NOT rewrite or paraphrase anything
- DO NOT translate the content
- DO NOT guess missing parts
- DO NOT merge questions
- DO NOT invent options

LATEX MATH FORMAT (REQUIRED — DO NOT KEEP UNICODE MATH):
- Convert ALL math expressions into valid LaTeX wrapped in $...$ delimiters
- Inline math: single $...$  (e.g. $x^2 + 3x - 4 = 0$)
- Display/block math: $$...$$ on its own line
- Expected LaTeX commands: \frac{a}{b}, \sqrt{x}, \sqrt[n]{x}, \lim_{x \to a},
  \int, \sum_{i=1}^{n}, x^2, a_n, \begin{cases}...\end{cases},
  \begin{pmatrix}...\end{pmatrix}, \rightarrow, \Rightarrow, \infty
- Superscripts: x² → $x^2$  |  aⁿ → $a^n$  |  10³ → $10^3$
- Subscripts: a₁ → $a_1$  |  xₙ → $x_n$  |  H₂O → $H_2O$
- Fractions: ½ → $\frac{1}{2}$  |  a/b → $\frac{a}{b}$
- Roots: √4 → $\sqrt{4}$  |  ∛8 → $\sqrt[3]{8}$
- Greek: α → $\alpha$, β → $\beta$, π → $\pi$, Σ → $\Sigma$, Δ → $\Delta$
- Chemistry: Na⁺ → $Na^+$, SO₄²⁻ → $SO_4^{2-}$

CONCRETE EXAMPLES (image content → expected LaTeX output):
  Image shows:  lim   (x²−1)/(x−1)
               x→1
  Output: $\lim_{x \to 1} \frac{x^2-1}{x-1}$

  Image shows:  f(x) = { x²  if x ≥ 0
                        { −x  if x < 0
  Output: $f(x) = \begin{cases} x^2 & \text{if } x \geq 0 \\ -x & \text{if } x < 0 \end{cases}$

  Image shows:  | 1  2  3 |
               | 4  5  6 |
               | 7  8  9 |
  Output: $\begin{pmatrix} 1 & 2 & 3 \\ 4 & 5 & 6 \\ 7 & 8 & 9 \end{pmatrix}$

PERSIAN/DARI RTL TEXT HANDLING (CRITICAL FOR ACCURACY):
- The raw PDF text may contain Persian/Dari characters in REVERSED or GARBLED order
- You MUST detect and FIX any reversed/garbled Persian/Dari words or sentences
- Ensure all Persian/Dari text reads correctly RIGHT-TO-LEFT
- Example: if you see "؟دوب ر وهشم" fix it to "مشهور بود ؟"
- Persian/Dari words must be in correct logical order for native readers

PARSING RULES:
- Questions are usually numbered (1, 2, 3, ...) or (۱) numbered
- Math expressions starting a line are NOT question numbers
- Options may appear as:
  (1) (2) (3) (4) or A) B) C) D) or A. B. C. D.
  or inline separated by spaces

- You must detect and map:
  first option → option_a
  second → option_b
  third → option_c
  fourth → option_d

IF DATA IS UNCLEAR:
- Return "invalid": true
- Do NOT guess

OUTPUT FORMAT (STRICT JSON ONLY):
{
  "questions": [
    {
      "question_text": "...",
      "option_a": "...",
      "option_b": "...",
      "option_c": "...",
      "option_d": "..."
    }
  ]
}

TEXT TO PARSE:
"""
%s
"""`, text)
}

// ──────────────────────────────────────────────────────────────
// Vision Prompts
// ──────────────────────────────────────────────────────────────

// buildVisionPrompt creates the prompt for vision-only extraction.
// Gemini receives page images and extracts questions visually.
func (gs *GeminiService) buildVisionPrompt(subject string) string {
	return fmt.Sprintf(`You are a precise visual MCQ extractor for Kankor exam PDFs.

SUBJECT: %s

YOUR TASK:
Look at the page image(s) provided and extract Multiple Choice Questions.

CRITICAL RULES:
- Copy text EXACTLY as it appears visually — never invent or guess
- Never translate Persian to English or English to Persian
- Never solve or simplify equations
- Never explain or comment — only output JSON

LATEX MATH FORMAT (REQUIRED — DO NOT KEEP UNICODE MATH):
- Convert ALL math expressions into valid LaTeX wrapped in $...$ delimiters
- Inline math: single $...$ (e.g. $x^2 + 3x - 4 = 0$)
- Display/block math: $$...$$ on its own line
- Expected LaTeX commands: \frac{a}{b}, \sqrt{x}, \sqrt[n]{x}, \lim_{x \to a},
  \int, \sum_{i=1}^{n}, x^2, a_n, \begin{cases}...\end{cases},
  \begin{pmatrix}...\end{pmatrix}, \rightarrow, \Rightarrow, \infty
- Superscripts: x² → $x^2$  |  aⁿ → $a^n$  |  10³ → $10^3$
- Subscripts: a₁ → $a_1$  |  xₙ → $x_n$  |  H₂O → $H_2O$
- Fractions: ½ → $\frac{1}{2}$  |  a/b → $\frac{a}{b}$
- Roots: √4 → $\sqrt{4}$  |  ∛8 → $\sqrt[3]{8}$
- Greek letters: α → $\alpha$, β → $\beta$, π → $\pi$, Σ → $\Sigma$, Δ → $\Delta$
- Chemistry formulas: H₂O → $H_2O$, Na⁺ → $Na^+$, SO₄²⁻ → $SO_4^{2-}$
- Non-math Persian/Dari text must stay OUTSIDE $...$ and remain untouched

CONCRETE EXAMPLES (image content → expected LaTeX output):
  Image shows:  lim   (x²−1)/(x−1)
               x→1
  Output: $\lim_{x \to 1} \frac{x^2-1}{x-1}$

  Image shows:  f(x) = { x²  if x ≥ 0
                        { −x  if x < 0
  Output: $f(x) = \begin{cases} x^2 & \text{if } x \geq 0 \\ -x & \text{if } x < 0 \end{cases}$

  Image shows:  | 1  2  3 |
               | 4  5  6 |
               | 7  8  9 |
  Output: $\begin{pmatrix} 1 & 2 & 3 \\ 4 & 5 & 6 \\ 7 & 8 & 9 \end{pmatrix}$

TEXT RULES:
- Keep Persian text in Persian (Farsi/Dari script), readable right-to-left
- Keep English text in English
- Preserve question numbering (1, 2, 3 or ۱, ۲, ۳)

OPTION DETECTION:
Options may appear as:
- A) B) C) D) or A. B. C. D.
- (1) (2) (3) (4) or 1) 2) 3) 4)
- Inline separated by spaces or tabs
- Each on a separate line

Map them to: option_a, option_b, option_c, option_d

IF UNSURE OR UNCLEAR:
- Set "invalid": true
- Do NOT guess or fabricate content

OUTPUT — STRICT JSON ONLY (no markdown, no explanation):
{
  "questions": [
    {
      "question_text": "EXACT question text from IMAGE with LaTeX math in $...$",
      "option_a": "EXACT option A text from IMAGE",
      "option_b": "EXACT option B text from IMAGE",
      "option_c": "EXACT option C text from IMAGE",
      "option_d": "EXACT option D text from IMAGE"
    }
  ]
}`, subject)
}

// buildHybridPrompt creates the prompt for hybrid extraction.
// Gemini receives BOTH page images AND extracted text.
// Image is primary truth; text is supplemental.
func (gs *GeminiService) buildHybridPrompt(subject string) string {
	return fmt.Sprintf(`You are a precise visual MCQ extractor for Kankor exam PDFs.

SUBJECT: %s

YOU RECEIVE:
1. Page IMAGES (the primary, authoritative source)
2. Extracted TEXT (supplemental only — may contain errors)

ABSOLUTE PRIORITY RULE:
**The image is the PRIMARY source of truth.**
The extracted text is ONLY supplemental.
If there is ANY conflict between the image and the text,
ALWAYS trust the image.

CRITICAL RULES:
- Copy text EXACTLY as it appears in the IMAGE — never invent or guess
- Never translate Persian to English or English to Persian
- Never solve or simplify equations
- Never explain or comment — only output JSON

LATEX MATH FORMAT (REQUIRED — DO NOT KEEP UNICODE MATH):
- Convert ALL math expressions into valid LaTeX wrapped in $...$ delimiters
- Inline math: single $...$ (e.g. $x^2 + 3x - 4 = 0$)
- Display/block math: $$...$$ on its own line
- Expected LaTeX commands: \frac{a}{b}, \sqrt{x}, \sqrt[n]{x}, \lim_{x \to a},
  \int, \sum_{i=1}^{n}, x^2, a_n, \begin{cases}...\end{cases},
  \begin{pmatrix}...\end{pmatrix}, \rightarrow, \Rightarrow, \infty
- Superscripts: x² → $x^2$  |  aⁿ → $a^n$  |  10³ → $10^3$
- Subscripts: a₁ → $a_1$  |  xₙ → $x_n$  |  H₂O → $H_2O$
- Fractions: ½ → $\frac{1}{2}$  |  a/b → $\frac{a}{b}$
- Roots: √4 → $\sqrt{4}$  |  ∛8 → $\sqrt[3]{8}$
- Greek letters: α → $\alpha$, β → $\beta$, π → $\pi$, Σ → $\Sigma$, Δ → $\Delta$
- Chemistry formulas: H₂O → $H_2O$, Na⁺ → $Na^+$, SO₄²⁻ → $SO_4^{2-}$
- Non-math Persian/Dari text must stay OUTSIDE $...$ and remain untouched

CONCRETE EXAMPLES (image content → expected LaTeX output):
  Image shows:  lim   (x²−1)/(x−1)
               x→1
  Output: $\lim_{x \to 1} \frac{x^2-1}{x-1}$

  Image shows:  f(x) = { x²  if x ≥ 0
                        { −x  if x < 0
  Output: $f(x) = \begin{cases} x^2 & \text{if } x \geq 0 \\ -x & \text{if } x < 0 \end{cases}$

  Image shows:  | 1  2  3 |
               | 4  5  6 |
               | 7  8  9 |
  Output: $\begin{pmatrix} 1 & 2 & 3 \\ 4 & 5 & 6 \\ 7 & 8 & 9 \end{pmatrix}$

TEXT RULES:
- Keep Persian text in Persian (Farsi/Dari script), readable right-to-left
- Keep English text in English
- Preserve question numbering as shown in the IMAGE

OPTION DETECTION:
Options may appear as:
- A) B) C) D) or A. B. C. D.
- (1) (2) (3) (4) or 1) 2) 3) 4)
- Inline or each on a separate line

Map them to: option_a, option_b, option_c, option_d

IF UNSURE OR UNCLEAR:
- Set "invalid": true
- Do NOT guess or fabricate content

OUTPUT — STRICT JSON ONLY (no markdown, no explanation):
{
  "questions": [
    {
      "question_text": "EXACT question text from IMAGE with LaTeX math in $...$",
      "option_a": "EXACT option A text from IMAGE",
      "option_b": "EXACT option B text from IMAGE",
      "option_c": "EXACT option C text from IMAGE",
      "option_d": "EXACT option D text from IMAGE"
    }
  ]
}

REMEMBER: Image = Truth. Text = Hint. Always trust the image.`, subject)
}

// ──────────────────────────────────────────────────────────────
// Vision API Core
// ──────────────────────────────────────────────────────────────

// extractQuestionsVisionBatch sends a batch of page images to Gemini Vision.
// When hybrid=true, per-page text is included alongside each image.
func (gs *GeminiService) extractQuestionsVisionBatch(pageImages []string, pageTexts []string, subject string, hybrid bool) ([]models.GeminiExtractedQuestion, error) {
	// Build the appropriate prompt
	var promptText string
	if hybrid {
		promptText = gs.buildHybridPrompt(subject)
	} else {
		promptText = gs.buildVisionPrompt(subject)
	}

	// Build parts array: prompt + per-page text (hybrid) + images
	parts := []GeminiVisionPart{
		{Text: promptText},
	}

	// For hybrid mode, add per-page extracted text before each image
	for i, imgB64 := range pageImages {
		if hybrid && i < len(pageTexts) && pageTexts[i] != "" {
			// Truncate long page texts to avoid overwhelming the model
			pageText := pageTexts[i]
			if len(pageText) > 3000 {
				pageText = pageText[:3000] + "..."
			}
			parts = append(parts, GeminiVisionPart{
				Text: fmt.Sprintf("[SUPPLEMENTAL TEXT for page %d — IMAGE is authoritative]:\n%s", i+1, pageText),
			})
		}

		// Add the page image
		parts = append(parts, GeminiVisionPart{
			InlineData: &GeminiInlineData{
				MimeType: "image/png",
				Data:     imgB64,
			},
		})
	}

	// Build the vision request
	visionReq := GeminiVisionRequest{
		Contents: []GeminiVisionContent{
			{Parts: parts},
		},
		GenerationConfig: GeminiGenerationConfig{
			MaxOutputTokens: 8192,
		},
	}

	// Use Pro-tier vision chain: gemini-3.1-pro-preview first, then Flash fallbacks
	visionPrimaryURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", geminiVisionModels[0])
	allURLs := append([]string{visionPrimaryURL}, gs.visionFallbackURLs...)

	var lastErr error
	consecutive429s := 0

	for attempt, modelURL := range allURLs {
		if attempt > 0 {
			var delay time.Duration
			if consecutive429s > 0 {
				delay = time.Duration(5*(1<<uint(consecutive429s-1))) * time.Second
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
				fmt.Printf("[GeminiService:Vision] Rate-limited %d times, waiting %v before model '%s'...\n",
					consecutive429s, delay, extractModelName(modelURL))
			} else {
				delay = 2 * time.Second
				fmt.Printf("[GeminiService:Vision] Switching to fallback model: %s\n", extractModelName(modelURL))
			}
			time.Sleep(delay)
		}

		questions, err := gs.callGeminiVisionAPI(visionReq, modelURL)
		if err == nil {
			if consecutive429s > 0 {
				fmt.Printf("[GeminiService:Vision] Succeeded after %d rate-limit retries\n", consecutive429s)
			}
			return questions, nil
		}

		fmt.Printf("[GeminiService:Vision] Attempt %d (model '%s') failed: %v\n",
			attempt+1, extractModelName(modelURL), err)
		lastErr = err

		if strings.Contains(err.Error(), "status 429") {
			consecutive429s++
		} else if strings.Contains(err.Error(), "status 503") ||
			strings.Contains(err.Error(), "status 404") ||
			strings.Contains(err.Error(), "status 403") ||
			strings.Contains(err.Error(), "API request failed") ||
			strings.Contains(err.Error(), "failed to parse vision questions") ||
			strings.Contains(err.Error(), "no parseable MCQ") ||
			strings.Contains(err.Error(), "model reported") {
			consecutive429s = 0
		} else {
			// Non-retryable error (auth, bad request): stop immediately
			break
		}
	}

	return nil, fmt.Errorf("vision extraction failed after %d attempts: %w", len(allURLs), lastErr)
}

// callGeminiVisionAPI makes the HTTP call for a vision request.
func (gs *GeminiService) callGeminiVisionAPI(visionReq GeminiVisionRequest, baseURL string) ([]models.GeminiExtractedQuestion, error) {
	jsonPayload, err := json.Marshal(visionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vision request: %w", err)
	}

	fmt.Printf("[GeminiService:Vision] Request payload size: %.1f KB\n", float64(len(jsonPayload))/1024.0)

	url := fmt.Sprintf("%s?key=%s", baseURL, gs.apiKey)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create vision request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := gs.visionHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vision API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read vision response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("API returned status 429: rate limit exceeded")
		}
		// Truncate long error bodies for logging
		errBody := string(body)
		if len(errBody) > 500 {
			errBody = errBody[:500] + "..."
		}
		return nil, fmt.Errorf("vision API returned status %d: %s", resp.StatusCode, errBody)
	}

	// Parse response (same structure as text API)
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse vision response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in vision response")
	}

	content := geminiResp.Candidates[0].Content
	if len(content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in vision response content")
	}

	responseText := content.Parts[0].Text

	// Debug preview
	if len(responseText) > 0 {
		previewLen := 500
		if len(responseText) < previewLen {
			previewLen = len(responseText)
		}
		fmt.Printf("[GeminiService:Vision] Response preview (first %d chars):\n% s\n", previewLen, responseText[:previewLen])
	}

	// Parse JSON from response
	questions, err := gs.parseQuestionsFromText(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vision questions: %w", err)
	}

	fmt.Printf("[GeminiService:Vision] Successfully parsed %d questions\n", len(questions))
	return questions, nil
}

// GeminiResponse represents the response from Gemini API
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// GeminiQuestionsResponse represents the structured questions response
type GeminiQuestionsResponse struct {
	Questions []models.GeminiExtractedQuestion `json:"questions"`
	Invalid   bool                             `json:"invalid,omitempty"`
	Error     string                           `json:"error,omitempty"`
}

// ──────────────────────────────────────────────────────────────
// Gemini Vision API Types (multimodal: image + text)
// ──────────────────────────────────────────────────────────────

// GeminiVisionRequest is the payload for vision/hybrid API calls
type GeminiVisionRequest struct {
	Contents         []GeminiVisionContent  `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
	SafetySettings   []GeminiSafetySetting  `json:"safety_settings,omitempty"`
}

// GeminiVisionContent holds an array of parts (text + images)
type GeminiVisionContent struct {
	Role  string             `json:"role,omitempty"`
	Parts []GeminiVisionPart `json:"parts"`
}

// GeminiVisionPart is either text or an inline image
type GeminiVisionPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inline_data,omitempty"`
}

// GeminiInlineData holds base64-encoded image data
type GeminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

// GeminiGenerationConfig controls model output behavior.
// Note: temperature, topP, topK are deprecated for all Gemini 3.x models.
// MaxOutputTokens is the primary control for output length.
type GeminiGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens"`
}

// GeminiSafetySetting allows tuning safety filters
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// ──────────────────────────────────────────────────────────────
// Strategy Classification
// ──────────────────────────────────────────────────────────────

// textOnlySubjects lists subjects that are explicitly opted OUT of vision extraction
// because their PDFs have clean, reliably-ordered text layers.
//
// RTL-script PDFs (Persian/Dari/Pashto) store text runs in visual order rather than
// logical order at the character level. Text extraction from these PDFs produces
// scrambled, unreadable output that cannot be reconstructed by an LLM. Vision-based
// extraction reads directly from page images and handles RTL correctly.
//
// This set is intentionally empty — all Kankor exam subjects use vision extraction.
// Add a subject here ONLY if you have verified that text extraction produces correct,
// byte-for-byte readable Persian/Dari output for that specific subject.
var textOnlySubjects = map[string]bool{}

// IsVisionSubject returns true for all subjects. PDF text layers for RTL scripts
// (Persian/Dari/Pashto) are stored visually rather than logically, producing
// scrambled output from text extraction. Vision-based extraction reads page images
// and is correct regardless of subject.
func IsVisionSubject(subject string) bool {
	// Default-true: only subjects in textOnlySubjects use the text pipeline.
	// Since that set is empty, all subjects use vision/hybrid extraction.
	return !textOnlySubjects[strings.ToLower(strings.TrimSpace(subject))]
}

// ExtractQuestionsWithVision sends page images to Gemini Vision for visual extraction.
// This preserves mathematical notation, equations, fractions, and symbols exactly as shown.
// Pages are processed in batches of 5 to stay within API size limits.
func (gs *GeminiService) ExtractQuestionsWithVision(pageImages []string, pageTexts []string, subject string) ([]models.GeminiExtractedQuestion, error) {
	if len(pageImages) == 0 {
		return nil, fmt.Errorf("no page images provided for vision extraction")
	}

	fmt.Printf("[GeminiService:Vision] Processing %d pages as images for subject: %s\n", len(pageImages), subject)

	const batchSize = 3 // Smaller batches for faster vision processing
	var allQuestions []models.GeminiExtractedQuestion

	// Process pages in batches
	for start := 0; start < len(pageImages); start += batchSize {
		end := start + batchSize
		if end > len(pageImages) {
			end = len(pageImages)
		}

		batchImages := pageImages[start:end]
		batchTexts := make([]string, 0)
		if start < len(pageTexts) {
			btEnd := end
			if btEnd > len(pageTexts) {
				btEnd = len(pageTexts)
			}
			batchTexts = pageTexts[start:btEnd]
		}

		fmt.Printf("[GeminiService:Vision] Batch %d-%d of %d pages\n", start+1, end, len(pageImages))

		questions, err := gs.extractQuestionsVisionBatch(batchImages, batchTexts, subject, false)
		if err != nil {
			return nil, fmt.Errorf("vision extraction failed on pages %d-%d: %w", start+1, end, err)
		}

		allQuestions = append(allQuestions, questions...)
		fmt.Printf("[GeminiService:Vision] Batch yielded %d questions (total: %d)\n", len(questions), len(allQuestions))
	}

	// Validate
	if err := gs.validateQuestions(allQuestions); err != nil {
		return nil, fmt.Errorf("invalid questions from vision extraction: %w", err)
	}

	fmt.Printf("[GeminiService:Vision] Complete: %d total questions extracted\n", len(allQuestions))
	return allQuestions, nil
}

// ExtractQuestionsHybrid sends BOTH extracted text AND page images to Gemini.
// The image is the primary source of truth; text is supplemental.
// If there is ANY conflict between image and text, the image wins.
func (gs *GeminiService) ExtractQuestionsHybrid(fullText string, pageImages []string, pageTexts []string, subject string) ([]models.GeminiExtractedQuestion, error) {
	if len(pageImages) == 0 {
		// Fall back to text-only if no images
		fmt.Printf("[GeminiService:Hybrid] No images available, falling back to text-only\n")
		return gs.ExtractQuestions(fullText)
	}

	fmt.Printf("[GeminiService:Hybrid] Processing %d pages with text+vision for: %s\n", len(pageImages), subject)

	const batchSize = 3 // Smaller batches for faster vision processing
	var allQuestions []models.GeminiExtractedQuestion

	for start := 0; start < len(pageImages); start += batchSize {
		end := start + batchSize
		if end > len(pageImages) {
			end = len(pageImages)
		}

		batchImages := pageImages[start:end]
		batchTexts := make([]string, 0)
		if start < len(pageTexts) {
			btEnd := end
			if btEnd > len(pageTexts) {
				btEnd = len(pageTexts)
			}
			batchTexts = pageTexts[start:btEnd]
		}

		fmt.Printf("[GeminiService:Hybrid] Batch %d-%d of %d pages\n", start+1, end, len(pageImages))

		questions, err := gs.extractQuestionsVisionBatch(batchImages, batchTexts, subject, true)
		if err != nil {
			return nil, fmt.Errorf("hybrid extraction failed on pages %d-%d: %w", start+1, end, err)
		}

		allQuestions = append(allQuestions, questions...)
		fmt.Printf("[GeminiService:Hybrid] Batch yielded %d questions (total: %d)\n", len(questions), len(allQuestions))
	}

	if err := gs.validateQuestions(allQuestions); err != nil {
		return nil, fmt.Errorf("invalid questions from hybrid extraction: %w", err)
	}

	fmt.Printf("[GeminiService:Hybrid] Complete: %d total questions extracted\n", len(allQuestions))
	return allQuestions, nil
}
