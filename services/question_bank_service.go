package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"
)

// QuestionBankService handles question bank operations
type QuestionBankService struct {
	geminiService   *GeminiService
	pdfExtractorURL string
	httpClient      *http.Client
}

// NewQuestionBankService creates a new question bank service
func NewQuestionBankService(geminiService *GeminiService, pdfExtractorURL string) *QuestionBankService {
	return &QuestionBankService{
		geminiService:   geminiService,
		pdfExtractorURL: pdfExtractorURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // 2 minute timeout for PDF processing
		},
	}
}

// pdfFullExtraction holds the result of the full /extract endpoint (text + images)
type pdfFullExtraction struct {
	text       string
	pageImages []string // base64-encoded PNG images
	pageTexts  []string // per-page text
	totalPages int
}

// UploadAndExtractPDF uploads PDF to extractor, chooses the best extraction strategy,
// sends to Gemini, and returns extracted questions.
//
// Strategy selection: all subjects use vision/hybrid extraction because
// RTL-script PDFs (Persian/Dari/Pashto) store text in visual order, not
// logical order. Vision extraction reads page images and is correct.
// The text-only pipeline is reserved for subjects in the textOnlySubjects
// exclusion list (currently empty).
func (qbs *QuestionBankService) UploadAndExtractPDF(file multipart.File, filename string, subject string) ([]models.GeminiExtractedQuestion, error) {
	fmt.Printf("[QuestionBankService] Processing PDF: %s, Subject: %s\n", filename, subject)

	// Determine extraction strategy based on subject
	isVision := IsVisionSubject(subject)

	var questions []models.GeminiExtractedQuestion
	var err error

	if isVision {
		fmt.Printf("[QuestionBankService] Subject '%s' → VISION/HYBRID pipeline\n", subject)
		questions, err = qbs.processWithVisionPipeline(file, filename, subject)
	} else {
		fmt.Printf("[QuestionBankService] Subject '%s' → TEXT pipeline\n", subject)
		questions, err = qbs.processWithTextPipeline(file, filename)
	}

	if err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("no questions could be extracted from the PDF")
	}

	// Clean and normalize
	fmt.Printf("[QuestionBankService] Cleaning and normalizing %d questions...\n", len(questions))
	for i := 0; i < len(questions) && i < 3; i++ {
		fmt.Printf("[QuestionBankService] BEFORE cleaning - Q%d: %s\n", i+1, truncateForLog(questions[i].QuestionText, 100))
	}

	questions = cleanPersianMCQData(questions)

	for i := 0; i < len(questions) && i < 3; i++ {
		fmt.Printf("[QuestionBankService] AFTER cleaning - Q%d: %s\n", i+1, truncateForLog(questions[i].QuestionText, 100))
	}

	fmt.Printf("[QuestionBankService] Extracted %d questions from PDF\n", len(questions))
	return questions, nil
}

// processWithTextPipeline handles text-heavy subjects using the original text-only flow
func (qbs *QuestionBankService) processWithTextPipeline(file multipart.File, filename string) ([]models.GeminiExtractedQuestion, error) {
	// Step 1: Extract text from PDF via Python microservice
	text, err := qbs.extractTextFromPDF(file, filename)
	if err != nil {
		return nil, fmt.Errorf("PDF text extraction failed: %w", err)
	}

	fmt.Printf("[QuestionBankService] Extracted %d characters of text\n", len(text))

	// Step 2: Send text to Gemini text API
	questions, err := qbs.geminiService.ExtractQuestions(text)
	if err != nil {
		return nil, fmt.Errorf("Gemini text extraction failed: %w", err)
	}

	return questions, nil
}

// processWithVisionPipeline handles science/math subjects using hybrid text+vision extraction
func (qbs *QuestionBankService) processWithVisionPipeline(file multipart.File, filename string, subject string) ([]models.GeminiExtractedQuestion, error) {
	// Step 1: Full extraction from PDF (text + rendered page images)
	extraction, err := qbs.extractFullFromPDF(file, filename)
	if err != nil {
		return nil, fmt.Errorf("PDF full extraction failed: %w", err)
	}

	fmt.Printf("[QuestionBankService] Full extraction: %d chars text, %d page images\n",
		len(extraction.text), len(extraction.pageImages))

	// Step 2: Choose hybrid vs vision-only based on text quality
	var questions []models.GeminiExtractedQuestion

	if len(extraction.text) > 100 && len(extraction.pageImages) > 0 {
		// Hybrid mode: both text and images → best accuracy
		fmt.Printf("[QuestionBankService] Using HYBRID mode (text + %d images)\n", len(extraction.pageImages))
		questions, err = qbs.geminiService.ExtractQuestionsHybrid(
			extraction.text, extraction.pageImages, extraction.pageTexts, subject,
		)
	} else if len(extraction.pageImages) > 0 {
		// Vision-only: no usable text, rely purely on images
		fmt.Printf("[QuestionBankService] Using VISION-ONLY mode (%d images)\n", len(extraction.pageImages))
		questions, err = qbs.geminiService.ExtractQuestionsWithVision(
			extraction.pageImages, extraction.pageTexts, subject,
		)
	} else if len(extraction.text) > 0 {
		// Fallback: text-only (no images rendered)
		fmt.Printf("[QuestionBankService] No images rendered, falling back to TEXT mode\n")
		questions, err = qbs.geminiService.ExtractQuestions(extraction.text)
	} else {
		return nil, fmt.Errorf("no extractable content found in PDF")
	}

	if err != nil {
		return nil, fmt.Errorf("Gemini vision extraction failed: %w", err)
	}

	return questions, nil
}

// extractTextFromPDF sends PDF to the Python microservice for text extraction
func (qbs *QuestionBankService) extractTextFromPDF(file multipart.File, filename string) (string, error) {
	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Add file to form
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileContent); err != nil {
		return "", fmt.Errorf("failed to write file to form: %w", err)
	}

	writer.Close()

	// Send to PDF extractor service
	url := fmt.Sprintf("%s/extract-text", qbs.pdfExtractorURL)
	fmt.Printf("[QuestionBankService] Sending PDF to extractor at: %s\n", url)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := qbs.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to PDF extractor: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("[QuestionBankService] PDF Extractor Response - Status: %d, Body: %s\n", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if errMsg, ok := errorResp["error"].(string); ok {
				return "", fmt.Errorf("PDF extractor error: %s", errMsg)
			}
		}
		return "", fmt.Errorf("PDF extractor returned status %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Text           string `json:"text"`
		Success        bool   `json:"success"`
		PagesProcessed int    `json:"pages_processed"`
		Error          string `json:"error"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("extraction failed: %s", result.Error)
	}

	return result.Text, nil
}

// extractFullFromPDF calls the /extract endpoint which returns text + rendered page images.
// Used for vision/hybrid pipelines where images are needed for Gemini Vision API.
func (qbs *QuestionBankService) extractFullFromPDF(file multipart.File, filename string) (*pdfFullExtraction, error) {
	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileContent); err != nil {
		return nil, fmt.Errorf("failed to write file to form: %w", err)
	}

	writer.Close()

	// Send to PDF extractor /extract endpoint
	url := fmt.Sprintf("%s/extract", qbs.pdfExtractorURL)
	fmt.Printf("[QuestionBankService] Sending PDF for full extraction at: %s\n", url)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := qbs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to PDF extractor: %w", err)
	}
	defer resp.Body.Close()

	// Read response (may be large due to base64 images)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if errMsg, ok := errorResp["error"].(string); ok {
				return nil, fmt.Errorf("PDF extractor error: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("PDF extractor returned status %d", resp.StatusCode)
	}

	// Parse the full extraction response
	var result struct {
		Success    bool   `json:"success"`
		Text       string `json:"text"`
		TotalPages int    `json:"total_pages"`
		Pages      []struct {
			PageNumber  int    `json:"page_number"`
			ImageBase64 string `json:"image_base64"`
			Text        string `json:"text"`
		} `json:"pages"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse full extraction response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("full extraction failed: %s", result.Error)
	}

	// Build the result
	extraction := &pdfFullExtraction{
		text:       result.Text,
		totalPages: result.TotalPages,
	}

	for _, p := range result.Pages {
		extraction.pageImages = append(extraction.pageImages, p.ImageBase64)
		extraction.pageTexts = append(extraction.pageTexts, p.Text)
	}

	fmt.Printf("[QuestionBankService] Full extraction: %d chars text, %d pages rendered\n",
		len(extraction.text), len(extraction.pageImages))

	return extraction, nil
}

// truncateForLog safely truncates a string for logging, handling multi-byte characters
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Try to find a safe cut point for multi-byte characters
	runes := []rune(s)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return s[:maxLen] + "..."
}

// cleanPersianMCQData removes numbering prefixes and symbols from questions and options
func cleanPersianMCQData(questions []models.GeminiExtractedQuestion) []models.GeminiExtractedQuestion {
	// Regex patterns for cleaning
	// Question prefixes: "1.", "۱.", ": 1.", ": ۱.", leading colons, etc.
	questionPrefixRegex := regexp.MustCompile(`^\s*[:：]?\s*[0-9۰-۹]+\.\s*`)
	leadingColonRegex := regexp.MustCompile(`^\s*[:：]+\s*`)

	// Option prefixes: "1)", "۱)", "(1)", "(۱)", etc.
	optionPrefixRegex := regexp.MustCompile(`^\s*[\(\[]?[0-9۰-۹]+[\)\]]?\s*`)

	cleaned := make([]models.GeminiExtractedQuestion, 0, len(questions))

	for _, q := range questions {
		cleanQ := q

		// Clean question text - handle BOTH fields
		if cleanQ.QuestionText != "" {
			// Remove number prefix like "1.", "۱."
			cleanQ.QuestionText = questionPrefixRegex.ReplaceAllString(cleanQ.QuestionText, "")
			// Remove leading colons ":" or "："
			cleanQ.QuestionText = leadingColonRegex.ReplaceAllString(cleanQ.QuestionText, "")
			// Remove any remaining leading colons and spaces
			cleanQ.QuestionText = strings.TrimLeft(cleanQ.QuestionText, ":： ")
			cleanQ.QuestionText = strings.TrimSpace(cleanQ.QuestionText)

			// Also clean Question field (legacy) to keep them in sync
			cleanQ.Question = cleanQ.QuestionText
		} else if cleanQ.Question != "" {
			// If only Question field exists, clean it and sync to QuestionText
			cleanQ.Question = questionPrefixRegex.ReplaceAllString(cleanQ.Question, "")
			cleanQ.Question = leadingColonRegex.ReplaceAllString(cleanQ.Question, "")
			cleanQ.Question = strings.TrimLeft(cleanQ.Question, ":： ")
			cleanQ.Question = strings.TrimSpace(cleanQ.Question)
			cleanQ.QuestionText = cleanQ.Question
		}

		// Clean options - handle BOTH new fields AND legacy Options map
		// Clean OptionA
		if cleanQ.OptionA != "" {
			cleanQ.OptionA = optionPrefixRegex.ReplaceAllString(cleanQ.OptionA, "")
			cleanQ.OptionA = strings.TrimSpace(cleanQ.OptionA)
		}

		// Clean OptionB
		if cleanQ.OptionB != "" {
			cleanQ.OptionB = optionPrefixRegex.ReplaceAllString(cleanQ.OptionB, "")
			cleanQ.OptionB = strings.TrimSpace(cleanQ.OptionB)
		}

		// Clean OptionC
		if cleanQ.OptionC != "" {
			cleanQ.OptionC = optionPrefixRegex.ReplaceAllString(cleanQ.OptionC, "")
			cleanQ.OptionC = strings.TrimSpace(cleanQ.OptionC)
		}

		// Clean OptionD
		if cleanQ.OptionD != "" {
			cleanQ.OptionD = optionPrefixRegex.ReplaceAllString(cleanQ.OptionD, "")
			cleanQ.OptionD = strings.TrimSpace(cleanQ.OptionD)
		}

		// Also clean the legacy Options map if it exists
		if cleanQ.Options != nil {
			if val, exists := cleanQ.Options["A"]; exists && val != "" {
				cleanedVal := optionPrefixRegex.ReplaceAllString(val, "")
				cleanQ.Options["A"] = strings.TrimSpace(cleanedVal)
			}
			if val, exists := cleanQ.Options["B"]; exists && val != "" {
				cleanedVal := optionPrefixRegex.ReplaceAllString(val, "")
				cleanQ.Options["B"] = strings.TrimSpace(cleanedVal)
			}
			if val, exists := cleanQ.Options["C"]; exists && val != "" {
				cleanedVal := optionPrefixRegex.ReplaceAllString(val, "")
				cleanQ.Options["C"] = strings.TrimSpace(cleanedVal)
			}
			if val, exists := cleanQ.Options["D"]; exists && val != "" {
				cleanedVal := optionPrefixRegex.ReplaceAllString(val, "")
				cleanQ.Options["D"] = strings.TrimSpace(cleanedVal)
			}
		}

		// Sync cleaned new fields back to Options map (ENSURE Options map is always populated)
		if cleanQ.Options == nil {
			cleanQ.Options = make(map[string]string)
		}
		// Always update Options map with cleaned values (even if empty, to avoid nil)
		cleanQ.Options["A"] = cleanQ.OptionA
		cleanQ.Options["B"] = cleanQ.OptionB
		cleanQ.Options["C"] = cleanQ.OptionC
		cleanQ.Options["D"] = cleanQ.OptionD

		cleaned = append(cleaned, cleanQ)
	}

	return cleaned
}

// SaveQuestions saves extracted questions to the question bank
func (qbs *QuestionBankService) SaveQuestions(req models.SaveQuestionBankRequest, userID string, centerID string) error {
	if len(req.Questions) == 0 {
		return fmt.Errorf("no questions to save")
	}

	fmt.Printf("[QuestionBankService] Saving %d questions for center %s\n", len(req.Questions), centerID)

	query := `INSERT INTO question_bank 
		(center_id, subject, question_text, option_a, option_b, option_c, option_d, correct_option, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	for i, q := range req.Questions {
		// DEBUG: Log the raw question structure
		fmt.Printf("[QuestionBankService] Question %d - Raw data:\n", i+1)
		fmt.Printf("  QuestionText: '%s'\n", q.QuestionText)
		fmt.Printf("  Question: '%s'\n", q.Question)
		fmt.Printf("  OptionA: '%s'\n", q.OptionA)
		fmt.Printf("  OptionB: '%s'\n", q.OptionB)
		fmt.Printf("  OptionC: '%s'\n", q.OptionC)
		fmt.Printf("  OptionD: '%s'\n", q.OptionD)
		if q.Options != nil {
			fmt.Printf("  Options[A]: '%s'\n", q.Options["A"])
			fmt.Printf("  Options[B]: '%s'\n", q.Options["B"])
			fmt.Printf("  Options[C]: '%s'\n", q.Options["C"])
			fmt.Printf("  Options[D]: '%s'\n", q.Options["D"])
		} else {
			fmt.Printf("  Options: nil\n")
		}

		// Safely extract question text - prefer QuestionText, fallback to Question
		questionText := q.QuestionText
		if questionText == "" {
			questionText = q.Question
		}

		// Safely extract options - use helper to ensure non-nil strings
		optionA := getSafeString(q.OptionA, q.Options, "A")
		optionB := getSafeString(q.OptionB, q.Options, "B")
		optionC := getSafeString(q.OptionC, q.Options, "C")
		optionD := getSafeString(q.OptionD, q.Options, "D")

		// Validate that we have the minimum required data
		if questionText == "" {
			fmt.Printf("[QuestionBankService] WARNING: Question %d has empty text, skipping\n", i+1)
			continue
		}

		fmt.Printf("[QuestionBankService] Saving question %d: %s\n", i+1, questionText[:min(50, len(questionText))])
		fmt.Printf("  Options: A='%s', B='%s', C='%s', D='%s'\n", optionA, optionB, optionC, optionD)

		_, err := config.DBConnection.Exec(context.Background(), query,
			centerID,
			req.Subject,
			questionText,
			optionA,
			optionB,
			optionC,
			optionD,
			nil, // correct_option is null by default
			userID,
		)
		if err != nil {
			return fmt.Errorf("failed to save question %d: %w", i+1, err)
		}
	}

	fmt.Printf("[QuestionBankService] Successfully saved %d questions\n", len(req.Questions))
	return nil
}

// getSafeString safely extracts a string value, ensuring it's never nil
func getSafeString(directValue string, optionsMap map[string]string, key string) string {
	// First, try the direct value (new format)
	if directValue != "" {
		return directValue
	}

	// Fallback to Options map (legacy format)
	if optionsMap != nil {
		if val, exists := optionsMap[key]; exists {
			return val
		}
	}

	// Return empty string as fallback (never nil)
	return ""
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetQuestionsByCenter retrieves questions for a specific center with optional subject filter
func (qbs *QuestionBankService) GetQuestionsByCenter(centerID string, subject string, page int, limit int) ([]models.QuestionBankEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	// Build query with optional subject filter
	var countQuery string
	var dataQuery string
	var args []interface{}

	if subject != "" {
		countQuery = `SELECT COUNT(*) FROM question_bank WHERE center_id = $1 AND subject = $2`
		dataQuery = `SELECT id, center_id, subject, question_text, option_a, option_b, option_c, option_d, correct_option, created_by, created_at 
			FROM question_bank WHERE center_id = $1 AND subject = $2 
			ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = append(args, centerID, subject, limit, offset)
	} else {
		countQuery = `SELECT COUNT(*) FROM question_bank WHERE center_id = $1`
		dataQuery = `SELECT id, center_id, subject, question_text, option_a, option_b, option_c, option_d, correct_option, created_by, created_at 
			FROM question_bank WHERE center_id = $1 
			ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = append(args, centerID, limit, offset)
	}

	// Get total count
	var total int64
	var err error
	if subject != "" {
		err = config.DBConnection.QueryRow(context.Background(), countQuery, centerID, subject).Scan(&total)
	} else {
		err = config.DBConnection.QueryRow(context.Background(), countQuery, centerID).Scan(&total)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to count questions: %w", err)
	}

	// Get questions
	var rows interface{}
	if subject != "" {
		rows, err = config.DBConnection.Query(context.Background(), dataQuery, centerID, subject, limit, offset)
	} else {
		rows, err = config.DBConnection.Query(context.Background(),
			`SELECT id, center_id, subject, question_text, option_a, option_b, option_c, option_d, correct_option, created_by, created_at 
			FROM question_bank WHERE center_id = $1 
			ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			centerID, limit, offset)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to query questions: %w", err)
	}

	// Type assert to pgx.Rows
	pgxRows := rows.(interface {
		Close()
		Next() bool
		Scan(dest ...interface{}) error
	})
	defer pgxRows.Close()

	var questions []models.QuestionBankEntry
	for pgxRows.Next() {
		var q models.QuestionBankEntry
		err := pgxRows.Scan(
			&q.ID, &q.CenterID, &q.Subject, &q.QuestionText,
			&q.OptionA, &q.OptionB, &q.OptionC, &q.OptionD,
			&q.CorrectOption, &q.CreatedBy, &q.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan question: %w", err)
		}
		questions = append(questions, q)
	}

	return questions, total, nil
}

// GetQuestionByID retrieves a specific question by ID
func (qbs *QuestionBankService) GetQuestionByID(id string, centerID string) (*models.QuestionBankEntry, error) {
	query := `SELECT id, center_id, subject, question_text, option_a, option_b, option_c, option_d, correct_option, created_by, created_at 
		FROM question_bank WHERE id = $1 AND center_id = $2`

	var q models.QuestionBankEntry
	err := config.DBConnection.QueryRow(context.Background(), query, id, centerID).Scan(
		&q.ID, &q.CenterID, &q.Subject, &q.QuestionText,
		&q.OptionA, &q.OptionB, &q.OptionC, &q.OptionD,
		&q.CorrectOption, &q.CreatedBy, &q.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get question: %w", err)
	}

	return &q, nil
}

// UpdateQuestion updates a question bank entry
func (qbs *QuestionBankService) UpdateQuestion(id string, req models.UpdateQuestionBankRequest, centerID string) error {
	// Validate correct_option if being set
	if req.CorrectOption != nil {
		validOptions := map[string]bool{"A": true, "B": true, "C": true, "D": true}
		if !validOptions[*req.CorrectOption] {
			return fmt.Errorf("correct_option must be one of A, B, C, D, got '%s'", *req.CorrectOption)
		}
	}

	// Build dynamic update query
	query := `UPDATE question_bank SET`
	var args []interface{}
	argIndex := 1
	var setClauses []string

	if req.Subject != nil {
		setClauses = append(setClauses, fmt.Sprintf("subject = $%d", argIndex))
		args = append(args, *req.Subject)
		argIndex++
	}
	if req.QuestionText != nil {
		setClauses = append(setClauses, fmt.Sprintf("question_text = $%d", argIndex))
		args = append(args, *req.QuestionText)
		argIndex++
	}
	if req.OptionA != nil {
		setClauses = append(setClauses, fmt.Sprintf("option_a = $%d", argIndex))
		args = append(args, *req.OptionA)
		argIndex++
	}
	if req.OptionB != nil {
		setClauses = append(setClauses, fmt.Sprintf("option_b = $%d", argIndex))
		args = append(args, *req.OptionB)
		argIndex++
	}
	if req.OptionC != nil {
		setClauses = append(setClauses, fmt.Sprintf("option_c = $%d", argIndex))
		args = append(args, *req.OptionC)
		argIndex++
	}
	if req.OptionD != nil {
		setClauses = append(setClauses, fmt.Sprintf("option_d = $%d", argIndex))
		args = append(args, *req.OptionD)
		argIndex++
	}
	if req.CorrectOption != nil {
		setClauses = append(setClauses, fmt.Sprintf("correct_option = $%d", argIndex))
		args = append(args, *req.CorrectOption)
		argIndex++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query += " " + joinStrings(setClauses, ", ")
	query += fmt.Sprintf(" WHERE id = $%d AND center_id = $%d", argIndex, argIndex+1)
	args = append(args, id, centerID)

	_, err := config.DBConnection.Exec(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to update question: %w", err)
	}

	return nil
}

// DeleteQuestion deletes a question bank entry
func (qbs *QuestionBankService) DeleteQuestion(id string, centerID string) error {
	query := `DELETE FROM question_bank WHERE id = $1 AND center_id = $2`

	result, err := config.DBConnection.Exec(context.Background(), query, id, centerID)
	if err != nil {
		return fmt.Errorf("failed to delete question: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("question not found or access denied")
	}

	return nil
}

// GetSubjectsByCenter returns all unique subjects for a center
func (qbs *QuestionBankService) GetSubjectsByCenter(centerID string) ([]string, error) {
	query := `SELECT DISTINCT subject FROM question_bank WHERE center_id = $1 ORDER BY subject`

	rows, err := config.DBConnection.Query(context.Background(), query, centerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subjects: %w", err)
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, fmt.Errorf("failed to scan subject: %w", err)
		}
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

// Helper function to join strings (since we can't import strings in some contexts)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
