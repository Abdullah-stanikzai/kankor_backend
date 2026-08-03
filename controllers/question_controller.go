package controllers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/services"
)

// QuestionController handles question-related requests
type QuestionController struct {
	config         *config.Config
	questionService *services.QuestionService
}

// NewQuestionController creates a new QuestionController instance
func NewQuestionController(cfg *config.Config) *QuestionController {
	return &QuestionController{
		config:         cfg,
		questionService: services.NewQuestionService(cfg),
	}
}

// GetExamQuestions gets all questions for an exam
// @Summary Get exam questions
// @Description Get all questions for an exam
// @Tags Questions
// @Accept json
// @Produce json
// @Param examId path string true "Exam ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/exams/{examId}/questions [get]
func (qc *QuestionController) GetExamQuestions(c *fiber.Ctx) error {
	examID := c.Params("examId")
	if examID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Exam ID is required", http.StatusBadRequest))
	}

	// Get user info from context
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	questions, err := qc.questionService.GetExamQuestions(examID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get questions", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(questions, "Questions retrieved successfully"))
}

// CreateQuestion creates a new question for an exam
// @Summary Create question
// @Description Create a new question for an exam
// @Tags Questions
// @Accept json
// @Produce json
// @Param examId path string true "Exam ID"
// @Param request body models.CreateQuestionRequest true "Create Question Request"
// @Security BearerAuth
// @Success 201 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/exams/{examId}/questions [post]
func (qc *QuestionController) CreateQuestion(c *fiber.Ctx) error {
	examID := c.Params("examId")
	if examID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Exam ID is required", http.StatusBadRequest))
	}

	var req models.CreateQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate required fields
	if req.QuestionText == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Question text is required", http.StatusBadRequest))
	}

	// Set the exam ID from the path parameter
	req.ExamID = examID

	// Get user info from context
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	question, err := qc.questionService.CreateQuestion(&req, userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to create question", http.StatusInternalServerError))
	}

	return c.Status(http.StatusCreated).JSON(config.SuccessResponse(question, "Question created successfully"))
}

// GetQuestionByID gets a question by ID
// @Summary Get question by ID
// @Description Get a question by its ID
// @Tags Questions
// @Accept json
// @Produce json
// @Param examId path string true "Exam ID"
// @Param id path string true "Question ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Router /center-admin/exams/{examId}/questions/{id} [get]
func (qc *QuestionController) GetQuestionByID(c *fiber.Ctx) error {
	examID := c.Params("examId")
	questionID := c.Params("id")
	if examID == "" || questionID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Exam ID and Question ID are required", http.StatusBadRequest))
	}

	// Get user info from context
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	question, err := qc.questionService.GetQuestionByID(questionID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Question not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(question, "Question retrieved successfully"))
}

// UpdateQuestion updates a question
// @Summary Update question
// @Description Update a question
// @Tags Questions
// @Accept json
// @Produce json
// @Param examId path string true "Exam ID"
// @Param id path string true "Question ID"
// @Param request body models.CreateQuestionRequest true "Update Question Request"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Router /center-admin/exams/{examId}/questions/{id} [put]
func (qc *QuestionController) UpdateQuestion(c *fiber.Ctx) error {
	examID := c.Params("examId")
	questionID := c.Params("id")
	if examID == "" || questionID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Exam ID and Question ID are required", http.StatusBadRequest))
	}

	var req models.CreateQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Get user info from context
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	question, err := qc.questionService.UpdateQuestion(questionID, &req, userID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Question not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(question, "Question updated successfully"))
}

// DeleteQuestion deletes a question
// @Summary Delete question
// @Description Delete a question
// @Tags Questions
// @Accept json
// @Produce json
// @Param examId path string true "Exam ID"
// @Param id path string true "Question ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/exams/{examId}/questions/{id} [delete]
func (qc *QuestionController) DeleteQuestion(c *fiber.Ctx) error {
	examID := c.Params("examId")
	questionID := c.Params("id")
	if examID == "" || questionID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Exam ID and Question ID are required", http.StatusBadRequest))
	}

	// Get user info from context
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	err := qc.questionService.DeleteQuestion(questionID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to delete question", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Question deleted successfully"))
}