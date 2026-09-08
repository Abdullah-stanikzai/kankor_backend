package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/services"

	"github.com/gofiber/fiber/v2"
)

// ExamController handles exam-related requests
type ExamController struct {
	config          *config.Config
	examService     *services.ExamService
	questionService *services.QuestionService
}

// NewExamController creates a new ExamController instance
func NewExamController(cfg *config.Config) *ExamController {
	return &ExamController{
		config:          cfg,
		examService:     services.NewExamService(cfg),
		questionService: services.NewQuestionService(cfg),
	}
}

// GetAllExams gets all exams for a center admin
func (ec *ExamController) GetAllExams(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)
	fmt.Printf("[GetAllExams] userID: %s, centerID: %s\n", userID, centerID)

	exams, err := ec.examService.GetAllExams(userID, centerID)
	if err != nil {
		fmt.Printf("[GetAllExams] Service error: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get exams", http.StatusInternalServerError))
	}

	fmt.Printf("[GetAllExams] Found %d exams\n", len(exams))
	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exams, "Exams retrieved successfully"))
}

// GetExamByID gets an exam by ID
func (ec *ExamController) GetExamByID(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	exam, err := ec.examService.GetExamByID(examID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Exam not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exam, "Exam retrieved successfully"))
}

// CreateExam creates a new exam
func (ec *ExamController) CreateExam(c *fiber.Ctx) error {
	var req models.CreateExamRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	userID := c.Locals("user_id").(string)

	exam, err := ec.examService.CreateExam(&req, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusCreated).JSON(config.SuccessResponse(exam, "Exam created successfully"))
}

// UpdateExam updates an existing exam
func (ec *ExamController) UpdateExam(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)

	var req models.CreateExamRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	exam, err := ec.examService.UpdateExam(examID, &req, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exam, "Exam updated successfully"))
}

// DeleteExam deletes an exam
func (ec *ExamController) DeleteExam(c *fiber.Ctx) error {
	examID := c.Params("id")
	centerID := c.Locals("user_center_id").(string)

	err := ec.examService.DeleteExam(examID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Exam deleted successfully"))
}

// PublishExam publishes an exam
func (ec *ExamController) PublishExam(c *fiber.Ctx) error {
	examID := c.Params("id")
	centerID := c.Locals("user_center_id").(string)

	exam, err := ec.examService.PublishExam(examID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exam, "Exam published successfully"))
}

// UnpublishExam unpublishes an exam
func (ec *ExamController) UnpublishExam(c *fiber.Ctx) error {
	examID := c.Params("id")
	centerID := c.Locals("user_center_id").(string)

	exam, err := ec.examService.UnpublishExam(examID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exam, "Exam unpublished successfully"))
}

// GetExamResults gets all results for an exam
func (ec *ExamController) GetExamResults(c *fiber.Ctx) error {
	examID := c.Params("id")
	centerID := c.Locals("user_center_id").(string)

	results, err := ec.examService.GetExamResults(examID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get exam results", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(results, "Exam results retrieved successfully"))
}

// GetExamResult gets a specific exam result
func (ec *ExamController) GetExamResult(c *fiber.Ctx) error {
	examID := c.Params("id")
	attemptID := c.Params("attemptId")
	centerID := c.Locals("user_center_id").(string)

	result, err := ec.examService.GetExamResult(examID, attemptID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Exam result not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(result, "Exam result retrieved successfully"))
}

// GetExamAnalytics gets analytics for an exam
func (ec *ExamController) GetExamAnalytics(c *fiber.Ctx) error {
	examID := c.Params("id")
	centerID := c.Locals("user_center_id").(string)

	analytics, err := ec.examService.GetExamAnalytics(examID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get exam analytics", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(analytics, "Exam analytics retrieved successfully"))
}

// Student endpoints

// GetUpcomingExams gets upcoming exams for a student
func (ec *ExamController) GetUpcomingExams(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	exams, err := ec.examService.GetUpcomingExams(userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get upcoming exams", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exams, "Upcoming exams retrieved successfully"))
}

// GetAvailableExams gets available exams for a student
func (ec *ExamController) GetAvailableExams(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	exams, err := ec.examService.GetAvailableExams(userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get available exams", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exams, "Available exams retrieved successfully"))
}

// GetSyncData returns a consolidated payload for the student app:
// server time, upcoming/available exams and submitted exam ids.
// GET /api/v1/student/exams/sync
func (ec *ExamController) GetSyncData(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	data, err := ec.examService.GetSyncData(userID, centerID)
	if err != nil {
		fmt.Printf("[GetSyncData] Service error: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to sync data", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(data, "Sync data retrieved successfully"))
}

// GetExamHistory gets a student's exam history
func (ec *ExamController) GetExamHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	history, err := ec.examService.GetExamHistory(userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get exam history", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(history, "Exam history retrieved successfully"))
}

// GetExamDetails gets exam details for a student
func (ec *ExamController) GetExamDetails(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	exam, err := ec.examService.GetExamDetails(examID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Exam not found or not available", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(exam, "Exam details retrieved successfully"))
}

// CheckExamSubmissionStatus checks if a student has already submitted an exam
func (ec *ExamController) CheckExamSubmissionStatus(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)

	hasSubmitted, err := ec.examService.HasStudentSubmittedExam(examID, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to check submission status", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(fiber.Map{
		"has_submitted": hasSubmitted,
		"exam_id":       examID,
	}, "Submission status retrieved successfully"))
}

// StartExam starts an exam for a student
func (ec *ExamController) StartExam(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)

	attempt, err := ec.examService.StartExam(examID, userID)
	if err != nil {
		status, message := mapStartExamError(err)
		return c.Status(status).JSON(config.ErrorResponse(message, status))
	}

	return c.Status(http.StatusCreated).JSON(config.SuccessResponse(attempt, "Exam started successfully"))
}

// mapStartExamError translates StartExam service errors into proper HTTP
// statuses so the app can distinguish "not started yet" / "has ended" from
// internal failures.
func mapStartExamError(err error) (int, string) {
	message := err.Error()
	switch {
	case strings.Contains(message, "has not started"):
		return http.StatusBadRequest, message
	case strings.Contains(message, "has ended"):
		return http.StatusConflict, message
	case strings.Contains(message, "not found") || strings.Contains(message, "not available"):
		return http.StatusNotFound, message
	default:
		return http.StatusInternalServerError, message
	}
}

// GetExamQuestions gets questions for an exam
func (ec *ExamController) GetExamQuestions(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)
	role := c.Locals("user_role").(string)

	fmt.Printf("[ExamController] GetExamQuestions - Exam: %s, User: %s, Role: %s, Center: %s\n",
		examID, userID, role, centerID)

	// Only students can get exam questions
	if role != "student" {
		fmt.Printf("[ExamController] Unauthorized role: %s\n", role)
		return c.Status(http.StatusForbidden).JSON(config.ErrorResponse("Only students can access exam questions", http.StatusForbidden))
	}

	questions, err := ec.examService.GetExamQuestions(examID, userID)
	if err != nil {
		fmt.Printf("[ExamController] Error: %v\n", err)

		// Check if exam not found
		if err.Error() == "exam not found" {
			return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Exam not found", http.StatusNotFound))
		}

		// For any other error, return 500 with details
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get exam questions: "+err.Error(), http.StatusInternalServerError))
	}

	// Handle empty questions gracefully
	if len(questions) == 0 {
		fmt.Printf("[ExamController] No questions found for exam %s - returning empty array\n", examID)
		return c.Status(http.StatusOK).JSON(config.SuccessResponse([]interface{}{}, "No questions found for this exam"))
	}

	fmt.Printf("[ExamController] Successfully fetched %d questions for exam %s\n", len(questions), examID)

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(questions, "Exam questions retrieved successfully"))
}

// SubmitExam submits an exam
func (ec *ExamController) SubmitExam(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)

	var req struct {
		AttemptID string            `json:"attempt_id"`
		Answers   map[string]string `json:"answers"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	result, err := ec.examService.SubmitExam(examID, req.AttemptID, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(result, "Exam submitted successfully"))
}

// GetStudentExamResults gets exam results for a student
func (ec *ExamController) GetStudentExamResults(c *fiber.Ctx) error {
	examID := c.Params("id")
	userID := c.Locals("user_id").(string)

	results, err := ec.examService.GetExamResults(examID, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get exam results", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(results, "Exam results retrieved successfully"))
}

// Question endpoints for center admin

// GetExamQuestionsAdmin gets all questions for an exam with their options
func (ec *ExamController) GetExamQuestionsAdmin(c *fiber.Ctx) error {
	examID := c.Params("examId")
	// userID and centerID available for future validation if needed

	// Fetch questions using NEW schema (with option_a, option_b, etc.)
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT id, exam_id, text, option_a, option_b, option_c, option_d, correct_option, created_at, updated_at 
		 FROM questions WHERE exam_id = $1 ORDER BY created_at ASC`,
		examID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get questions", http.StatusInternalServerError))
	}
	defer rows.Close()

	type QuestionWithOptionsAdmin struct {
		ID            string            `json:"id"`
		ExamID        string            `json:"exam_id"`
		Text          string            `json:"text"`
		Options       map[string]string `json:"options"`        // {"A": "Option A", "B": "Option B", ...}
		CorrectOption string            `json:"correct_option"` // "A", "B", "C", or "D"
		CreatedAt     time.Time         `json:"created_at"`
	}

	var questionsWithOptions []QuestionWithOptionsAdmin
	for rows.Next() {
		var q models.QuestionWithFlatOptions
		err := rows.Scan(
			&q.ID, &q.ExamID, &q.Text, &q.OptionA, &q.OptionB, &q.OptionC, &q.OptionD,
			&q.CorrectOption, &q.CreatedAt, &q.UpdatedAt,
		)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to scan question: %v", err), http.StatusInternalServerError))
		}

		questionsWithOptions = append(questionsWithOptions, QuestionWithOptionsAdmin{
			ID:     q.ID,
			ExamID: q.ExamID,
			Text:   q.Text,
			Options: map[string]string{
				"A": q.OptionA,
				"B": q.OptionB,
				"C": q.OptionC,
				"D": q.OptionD,
			},
			CorrectOption: q.CorrectOption,
			CreatedAt:     q.CreatedAt,
		})
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(questionsWithOptions, "Questions retrieved successfully"))
}

// CreateQuestion creates a new question for an exam (using NEW schema)
func (ec *ExamController) CreateQuestion(c *fiber.Ctx) error {
	examID := c.Params("examId")

	// Log request for debugging
	fmt.Printf("[CreateQuestion] Received request for exam ID: %s\n", examID)
	fmt.Printf("[CreateQuestion] Request body: %s\n", string(c.Body()))

	var req struct {
		Text          string            `json:"text" validate:"required"`
		Options       map[string]string `json:"options" validate:"required"` // {"A": "Option A", "B": "Option B", ...}
		CorrectOption string            `json:"correct_option" validate:"required,oneof=A B C D"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate options
	if len(req.Options) != 4 {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Must have exactly 4 options", http.StatusBadRequest))
	}
	for _, key := range []string{"A", "B", "C", "D"} {
		if _, exists := req.Options[key]; !exists {
			return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse(fmt.Sprintf("Missing option %s", key), http.StatusBadRequest))
		}
	}

	// Insert question with NEW schema
	var question models.QuestionWithFlatOptions
	query := `INSERT INTO questions (exam_id, text, option_a, option_b, option_c, option_d, correct_option, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, exam_id, text, option_a, option_b, option_c, option_d, correct_option, created_at, updated_at`

	fmt.Printf("[CreateQuestion] Executing SQL query with examID: %s\n", examID)
	fmt.Printf("[CreateQuestion] Text: %s, Options: A=%s, B=%s, C=%s, D=%s, Correct: %s\n",
		req.Text, req.Options["A"], req.Options["B"], req.Options["C"], req.Options["D"], req.CorrectOption)

	err := config.DBConnection.QueryRow(context.Background(), query,
		examID, req.Text, req.Options["A"], req.Options["B"], req.Options["C"], req.Options["D"], req.CorrectOption).Scan(
		&question.ID, &question.ExamID, &question.Text, &question.OptionA, &question.OptionB, &question.OptionC, &question.OptionD,
		&question.CorrectOption, &question.CreatedAt, &question.UpdatedAt,
	)

	if err != nil {
		fmt.Printf("[CreateQuestion] Database error: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to create question: %v", err), http.StatusInternalServerError))
	}

	fmt.Printf("[CreateQuestion] Question created successfully with ID: %s\n", question.ID)

	// Return the question with its options
	type QuestionWithOptions struct {
		ID        string            `json:"id"`
		ExamID    string            `json:"exam_id"`
		Text      string            `json:"text"`
		Options   map[string]string `json:"options"`
		CreatedAt time.Time         `json:"created_at"`
	}

	questionWithOptions := QuestionWithOptions{
		ID:        question.ID,
		ExamID:    question.ExamID,
		Text:      question.Text,
		Options:   req.Options,
		CreatedAt: question.CreatedAt,
	}

	return c.Status(http.StatusCreated).JSON(config.SuccessResponse(questionWithOptions, "Question created successfully"))
}

// GetQuestionByIDAdmin gets a question by ID
func (ec *ExamController) GetQuestionByIDAdmin(c *fiber.Ctx) error {
	examID := c.Params("examId")
	questionID := c.Params("questionId")
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	question, err := ec.questionService.GetQuestionByID(questionID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Question not found", http.StatusNotFound))
	}

	// Verify the question belongs to the specified exam
	if question.ExamID != examID {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Question not found in this exam", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(question, "Question retrieved successfully"))
}

// UpdateQuestionAdmin updates a question
func (ec *ExamController) UpdateQuestionAdmin(c *fiber.Ctx) error {
	examID := c.Params("examId")
	questionID := c.Params("questionId")

	// Log request for debugging
	fmt.Printf("[UpdateQuestionAdmin] Received request for exam ID: %s, question ID: %s\n", examID, questionID)
	fmt.Printf("[UpdateQuestionAdmin] Request body: %s\n", string(c.Body()))

	var req struct {
		Text          string            `json:"text" validate:"required"`
		Options       map[string]string `json:"options" validate:"required"` // {"A": "Option A", "B": "Option B", ...}
		CorrectOption string            `json:"correct_option" validate:"required,oneof=A B C D"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate options
	if len(req.Options) != 4 {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Must have exactly 4 options", http.StatusBadRequest))
	}
	for _, key := range []string{"A", "B", "C", "D"} {
		if _, exists := req.Options[key]; !exists {
			return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse(fmt.Sprintf("Missing option %s", key), http.StatusBadRequest))
		}
	}

	// Update the question in database
	var question models.QuestionWithFlatOptions
	query := `UPDATE questions 
		SET text = $1, option_a = $2, option_b = $3, option_c = $4, option_d = $5, correct_option = $6, updated_at = NOW()
		WHERE id = $7 AND exam_id = $8
		RETURNING id, exam_id, text, option_a, option_b, option_c, option_d, correct_option, created_at, updated_at`

	fmt.Printf("[UpdateQuestionAdmin] Executing SQL query with questionID: %s\n", questionID)
	fmt.Printf("[UpdateQuestionAdmin] Text: %s, Options: A=%s, B=%s, C=%s, D=%s, Correct: %s\n",
		req.Text, req.Options["A"], req.Options["B"], req.Options["C"], req.Options["D"], req.CorrectOption)

	err := config.DBConnection.QueryRow(context.Background(), query,
		req.Text, req.Options["A"], req.Options["B"], req.Options["C"], req.Options["D"], req.CorrectOption, questionID, examID).Scan(
		&question.ID, &question.ExamID, &question.Text, &question.OptionA, &question.OptionB, &question.OptionC, &question.OptionD,
		&question.CorrectOption, &question.CreatedAt, &question.UpdatedAt,
	)

	if err != nil {
		fmt.Printf("[UpdateQuestionAdmin] Database error: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(fmt.Sprintf("Failed to update question: %v", err), http.StatusInternalServerError))
	}

	fmt.Printf("[UpdateQuestionAdmin] Question updated successfully with ID: %s\n", question.ID)

	// Return the question with its options
	type QuestionWithOptions struct {
		ID        string            `json:"id"`
		ExamID    string            `json:"exam_id"`
		Text      string            `json:"text"`
		Options   map[string]string `json:"options"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}

	questionWithOptions := QuestionWithOptions{
		ID:        question.ID,
		ExamID:    question.ExamID,
		Text:      question.Text,
		Options:   req.Options,
		CreatedAt: question.CreatedAt,
		UpdatedAt: question.UpdatedAt,
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(questionWithOptions, "Question updated successfully"))
}

// DeleteQuestionAdmin deletes a question
func (ec *ExamController) DeleteQuestionAdmin(c *fiber.Ctx) error {
	_ = c.Params("examId") // examId is not used but required for route consistency
	questionID := c.Params("questionId")
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	err := ec.questionService.DeleteQuestion(questionID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Question deleted successfully"))
}

// DeleteExamQuestion deletes a question from an exam (convenience endpoint)
func (ec *ExamController) DeleteExamQuestion(c *fiber.Ctx) error {
	questionID := c.Params("questionId")
	userID := c.Locals("user_id").(string)
	centerID := c.Locals("user_center_id").(string)

	err := ec.questionService.DeleteQuestion(questionID, userID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Question deleted successfully"))
}

// UpdateExamStatus updates the status of an exam (pending/published)
func (ec *ExamController) UpdateExamStatus(c *fiber.Ctx) error {
	examID := c.Params("id")

	var req struct {
		Status string `json:"status" validate:"required,oneof=pending published"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate status
	if req.Status != "pending" && req.Status != "published" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid status. Must be 'pending' or 'published'", http.StatusBadRequest))
	}

	err := ec.examService.UpdateExamStatus(examID, req.Status)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse(err.Error(), http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Exam status updated successfully"))
}
