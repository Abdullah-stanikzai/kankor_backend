package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/services"

	"github.com/gofiber/fiber/v2"
)

// QuestionBankController handles question bank HTTP requests
type QuestionBankController struct {
	questionBankService *services.QuestionBankService
}

// NewQuestionBankController creates a new question bank controller
func NewQuestionBankController(cfg *config.Config, questionBankService *services.QuestionBankService) *QuestionBankController {
	return &QuestionBankController{
		questionBankService: questionBankService,
	}
}

// UploadPDF handles PDF upload and question extraction
func (qbc *QuestionBankController) UploadPDF(c *fiber.Ctx) error {
	// Get user info from JWT
	userID := c.Locals("user_id").(string)

	// Get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("No file uploaded", http.StatusBadRequest))
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Failed to open file", http.StatusBadRequest))
	}
	defer file.Close()

	// Get subject from form
	subject := c.FormValue("subject")
	if subject == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Subject is required", http.StatusBadRequest))
	}

	fmt.Printf("[QuestionBankController] User %s uploading PDF: %s, Subject: %s\n", userID, fileHeader.Filename, subject)

	// Process PDF and extract questions
	questions, err := qbc.questionBankService.UploadAndExtractPDF(file, fileHeader.Filename, subject)
	if err != nil {
		fmt.Printf("[QuestionBankController] Error extracting questions: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to extract questions: %v", err), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(fiber.Map{
		"questions": questions,
		"count":     len(questions),
		"subject":   subject,
		"filename":  fileHeader.Filename,
	}, fmt.Sprintf("Successfully extracted %d questions", len(questions))))
}

// SaveQuestions saves extracted questions to the question bank
func (qbc *QuestionBankController) SaveQuestions(c *fiber.Ctx) error {
	fmt.Println("\n[QuestionBankController] ========== SAVE QUESTIONS ENDPOINT HIT ==========")
	fmt.Printf("[QuestionBankController] Method: %s\n", c.Method())
	fmt.Printf("[QuestionBankController] Path: %s\n", c.Path())

	// Get user info from JWT
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		fmt.Printf("[QuestionBankController] ERROR: user_id not found in locals\n")
		return c.Status(http.StatusUnauthorized).JSON(config.ErrorResponse("User ID not found", http.StatusUnauthorized))
	}

	// FIX: Middleware stores it as "user_center_id", not "center_id"
	centerID, ok := c.Locals("user_center_id").(string)
	if !ok {
		fmt.Printf("[QuestionBankController] ERROR: user_center_id not found in locals\n")
		return c.Status(http.StatusUnauthorized).JSON(config.ErrorResponse("Center ID not found", http.StatusUnauthorized))
	}

	fmt.Printf("[QuestionBankController] User ID: %s\n", userID)
	fmt.Printf("[QuestionBankController] Center ID: %s\n", centerID)

	// Parse request body
	var req models.SaveQuestionBankRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Printf("[QuestionBankController] ERROR: Failed to parse body: %v\n", err)
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse(fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest))
	}

	fmt.Printf("[QuestionBankController] Parsed request - Subject: '%s', Questions count: %d\n", req.Subject, len(req.Questions))

	// Log first question structure
	if len(req.Questions) > 0 {
		q := req.Questions[0]
		fmt.Printf("[QuestionBankController] First question structure:\n")
		fmt.Printf("  QuestionText: '%s'\n", q.QuestionText)
		fmt.Printf("  Question: '%s'\n", q.Question)
		fmt.Printf("  OptionA: '%s'\n", q.OptionA)
		fmt.Printf("  OptionB: '%s'\n", q.OptionB)
		if q.Options != nil {
			fmt.Printf("  Options map exists with %d entries\n", len(q.Options))
		} else {
			fmt.Printf("  Options map: nil\n")
		}
	}

	// Validate request
	if req.Subject == "" {
		fmt.Printf("[QuestionBankController] ERROR: Subject is empty\n")
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Subject is required", http.StatusBadRequest))
	}
	if len(req.Questions) == 0 {
		fmt.Printf("[QuestionBankController] ERROR: No questions in request\n")
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("No questions to save", http.StatusBadRequest))
	}

	fmt.Printf("[QuestionBankController] Validation passed, calling service...\n")

	// Save questions
	if err := qbc.questionBankService.SaveQuestions(req, userID, centerID); err != nil {
		fmt.Printf("[QuestionBankController] ERROR from service: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to save questions: %v", err), http.StatusInternalServerError))
	}

	fmt.Printf("[QuestionBankController] Successfully saved %d questions\n", len(req.Questions))
	fmt.Println("[QuestionBankController] ========== END SAVE QUESTIONS ==========\n")

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(fiber.Map{
		"saved_count": len(req.Questions),
		"subject":     req.Subject,
	}, fmt.Sprintf("Successfully saved %d questions to question bank", len(req.Questions))))
}

// GetQuestions retrieves questions for the center
func (qbc *QuestionBankController) GetQuestions(c *fiber.Ctx) error {
	centerID := c.Locals("user_center_id").(string)

	// Get query parameters
	subject := c.Query("subject", "")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "50")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	questions, total, err := qbc.questionBankService.GetQuestionsByCenter(centerID, subject, page, limit)
	if err != nil {
		fmt.Printf("[QuestionBankController] Error getting questions: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to get questions: %v", err), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(fiber.Map{
		"questions": questions,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"has_more":  (page * limit) < int(total),
	}, "Questions retrieved successfully"))
}

// GetQuestion retrieves a specific question
func (qbc *QuestionBankController) GetQuestion(c *fiber.Ctx) error {
	centerID := c.Locals("user_center_id").(string)
	questionID := c.Params("id")

	question, err := qbc.questionBankService.GetQuestionByID(questionID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Question not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(question, "Question retrieved successfully"))
}

// UpdateQuestion updates a question
func (qbc *QuestionBankController) UpdateQuestion(c *fiber.Ctx) error {
	centerID := c.Locals("user_center_id").(string)
	questionID := c.Params("id")

	var req models.UpdateQuestionBankRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	if err := qbc.questionBankService.UpdateQuestion(questionID, req, centerID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to update question: %v", err), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Question updated successfully"))
}

// DeleteQuestion deletes a question
func (qbc *QuestionBankController) DeleteQuestion(c *fiber.Ctx) error {
	centerID := c.Locals("user_center_id").(string)
	questionID := c.Params("id")

	if err := qbc.questionBankService.DeleteQuestion(questionID, centerID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to delete question: %v", err), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Question deleted successfully"))
}

// GetSubjects retrieves all unique subjects for the center
func (qbc *QuestionBankController) GetSubjects(c *fiber.Ctx) error {
	centerID := c.Locals("user_center_id").(string)

	subjects, err := qbc.questionBankService.GetSubjectsByCenter(centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to get subjects: %v", err), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(fiber.Map{
		"subjects": subjects,
	}, "Subjects retrieved successfully"))
}
