package controllers

import (
	"errors"
	"fmt"
	"kankor-backend/middleware"
	"kankor-backend/models"
	"kankor-backend/services"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ExamEngineController handles exam taking operations
type ExamEngineController struct {
	engineService *services.ExamEngineService
}

// NewExamEngineController creates a new ExamEngineController instance
func NewExamEngineController(engineService *services.ExamEngineService) *ExamEngineController {
	return &ExamEngineController{
		engineService: engineService,
	}
}

// GetQuestionsForExam GET /api/student/exams/:exam_id/questions
// Fetches all questions for a specific exam
func (ctrl *ExamEngineController) GetQuestionsForExam(c *fiber.Ctx) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ExamEngine] PANIC recovered: %v\n", r)
			c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"message": "Internal server error occurred while fetching questions",
				},
			})
		}
	}()

	// Get student info from JWT token
	claims, exists := c.Locals("userClaims").(*middleware.JWTClaims)
	if !exists || claims == nil {
		fmt.Println("[ExamEngine] Invalid or missing JWT claims")
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized: Invalid token")
	}

	if claims.Role != "student" {
		fmt.Printf("[ExamEngine] Unauthorized role: %s\n", claims.Role)
		return fiber.NewError(fiber.StatusForbidden, "Only students can access exams")
	}

	examID := strings.TrimSpace(c.Params("exam_id"))
	if examID == "" {
		fmt.Println("[ExamEngine] Empty exam ID provided")
		return fiber.NewError(fiber.StatusBadRequest, "Exam ID is required")
	}

	fmt.Printf("[ExamEngine] Fetching questions for exam %s by student %s (center: %s)\n",
		examID, claims.UserID, claims.CenterID)

	questions, err := ctrl.engineService.GetQuestionsForExam(examID, claims.UserID, claims.CenterID)
	if err != nil {
		fmt.Printf("[ExamEngine] Error fetching questions: %v\n", err)

		// Check specific error types
		errMsg := err.Error()
		if errMsg == "exam not found or inaccessible" {
			fmt.Printf("[ExamEngine] Exam %s not found or inaccessible\n", examID)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"message": "Exam not found or you don't have access to it",
				},
			})
		}

		if errMsg == "student does not have access to this exam" {
			fmt.Printf("[ExamEngine] Student %s doesn't have access to exam %s\n", claims.UserID, examID)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"message": "You don't have access to this exam",
				},
			})
		}

		// For any other error, return 500 with details
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"message": "Failed to fetch questions: " + errMsg,
			},
		})
	}

	// Handle empty questions gracefully - DO NOT throw error
	if len(questions) == 0 {
		fmt.Printf("[ExamEngine] No questions found for exam %s - returning empty array\n", examID)
		return c.JSON(fiber.Map{
			"success": true,
			"data":    []interface{}{}, // Return empty array, not null
			"message": "No questions found for this exam",
		})
	}

	fmt.Printf("[ExamEngine] Successfully fetched %d questions for exam %s\n", len(questions), examID)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    questions,
		"message": "Questions retrieved successfully",
	})
}

// SaveAnswer POST /api/student/answers
// Saves or updates a student's answer (auto-save functionality)
func (ctrl *ExamEngineController) SaveAnswer(c *fiber.Ctx) error {
	// Get student info from JWT token
	claims, exists := c.Locals("userClaims").(*middleware.JWTClaims)
	if !exists || claims == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized: Invalid token")
	}

	if claims.Role != "student" {
		return fiber.NewError(fiber.StatusForbidden, "Only students can save answers")
	}

	var req models.SaveAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate request
	if req.ExamID == "" || req.QuestionID == "" || req.SelectedOption == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Exam ID, Question ID, and Selected Option are required")
	}

	if req.SelectedOption != "A" && req.SelectedOption != "B" && req.SelectedOption != "C" && req.SelectedOption != "D" {
		return fiber.NewError(fiber.StatusBadRequest, "Selected option must be A, B, C, or D")
	}

	fmt.Printf("[ExamEngine] Saving answer - Student: %s, Exam: %s, Question: %s, Option: %s\n",
		claims.UserID, req.ExamID, req.QuestionID, req.SelectedOption)

	answer, err := ctrl.engineService.SaveAnswer(claims.UserID, req.ExamID, req.QuestionID, req.SelectedOption)
	if err != nil {
		fmt.Printf("[ExamEngine] Error saving answer: %v\n", err)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to save answer: %v", err))
	}

	fmt.Printf("[ExamEngine] Answer saved successfully - Correct: %v\n", answer.IsCorrect)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    answer,
		"message": "Answer saved successfully",
	})
}

// SubmitExam POST /api/student/exams/:exam_id/submit
// Submits the exam and calculates results
func (ctrl *ExamEngineController) SubmitExam(c *fiber.Ctx) error {
	// Get student info from JWT token
	claims, exists := c.Locals("userClaims").(*middleware.JWTClaims)
	if !exists || claims == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized: Invalid token")
	}

	if claims.Role != "student" {
		return fiber.NewError(fiber.StatusForbidden, "Only students can submit exams")
	}

	examID := strings.TrimSpace(c.Params("exam_id"))
	if examID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Exam ID is required")
	}

	// Parse time spent (optional)
	var req models.SubmitExamRequest
	if err := c.BodyParser(&req); err != nil {
		req.TimeSpent = 0 // Default to 0 if not provided
	}

	fmt.Printf("[ExamEngine] Submitting exam - Student: %s, Exam: %s, Time Spent: %ds\n",
		claims.UserID, examID, req.TimeSpent)

	result, err := ctrl.engineService.SubmitExam(claims.UserID, examID, req.TimeSpent)
	if err != nil {
		fmt.Printf("[ExamEngine] Error submitting exam: %v\n", err)

		// Check if it's a validation error
		if errors.Is(err, fmt.Errorf("exam has no questions")) {
			return fiber.NewError(fiber.StatusBadRequest, "Cannot submit exam with no questions")
		}

		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to submit exam: %v", err))
	}

	fmt.Printf("[ExamEngine] Exam submitted successfully - Score: %.2f%%, Correct: %d/%d\n",
		result.Percentage, result.CorrectAnswers, result.TotalQuestions)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
		"message": "Exam submitted successfully",
	})
}

// GetExamResult GET /api/student/exams/:exam_id/result
// Fetches the result for a completed exam
func (ctrl *ExamEngineController) GetExamResult(c *fiber.Ctx) error {
	// Get student info from JWT token
	claims, exists := c.Locals("userClaims").(*middleware.JWTClaims)
	if !exists || claims == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized: Invalid token")
	}

	if claims.Role != "student" {
		return fiber.NewError(fiber.StatusForbidden, "Only students can view their results")
	}

	examID := strings.TrimSpace(c.Params("exam_id"))
	if examID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Exam ID is required")
	}

	fmt.Printf("[ExamEngine] Fetching result for student %s, exam %s\n", claims.UserID, examID)

	result, err := ctrl.engineService.GetStudentResult(claims.UserID, examID)
	if err != nil {
		fmt.Printf("[ExamEngine] Error fetching result: %v\n", err)
		return fiber.NewError(fiber.StatusNotFound, "No result found for this exam. Please submit the exam first.")
	}

	fmt.Printf("[ExamEngine] Result fetched successfully - Score: %.2f%%\n", result.Score)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
		"message": "Exam result retrieved successfully",
	})
}

// GetAllResults GET /api/student/results
// Fetches all exam results for a student
func (ctrl *ExamEngineController) GetAllResults(c *fiber.Ctx) error {
	// Get student info from JWT token
	claims, exists := c.Locals("userClaims").(*middleware.JWTClaims)
	if !exists || claims == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized: Invalid token")
	}

	if claims.Role != "student" {
		return fiber.NewError(fiber.StatusForbidden, "Only students can view their results")
	}

	fmt.Printf("[ExamEngine] Fetching all results for student %s, center %s\n", claims.UserID, claims.CenterID)

	results, err := ctrl.engineService.GetAllStudentResults(claims.UserID, claims.CenterID)
	if err != nil {
		fmt.Printf("[ExamEngine] Error fetching results: %v\n", err)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to fetch results: %v", err))
	}

	fmt.Printf("[ExamEngine] Found %d results for student %s\n", len(results), claims.UserID)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    results,
		"message": "Exam results retrieved successfully",
	})
}
