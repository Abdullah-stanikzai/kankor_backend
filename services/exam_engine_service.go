package services

import (
	"context"
	"fmt"
	"kankor-backend/config"
	"kankor-backend/models"
	"time"
)

// ExamEngineService handles exam taking and answer management
type ExamEngineService struct {
	cfg *config.Config
}

// NewExamEngineService creates a new ExamEngineService instance
func NewExamEngineService(cfg *config.Config) *ExamEngineService {
	return &ExamEngineService{cfg: cfg}
}

// GetQuestionsForExam fetches all questions for a specific exam
func (s *ExamEngineService) GetQuestionsForExam(examID, studentID, centerID string) ([]*models.QuestionWithOptions, error) {
	// Verify student has access to this exam through their center
	var examCenterID string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT center_id FROM exams WHERE id = $1`, examID).Scan(&examCenterID)
	if err != nil {
		fmt.Printf("[DB] Exam %s not found: %v\n", examID, err)
		return nil, fmt.Errorf("exam not found or inaccessible: %w", err)
	}

	fmt.Printf("[DB] Exam %s belongs to center %s\n", examID, examCenterID)

	// Verify the exam belongs to student's center
	if examCenterID != centerID {
		fmt.Printf("[DB] Center mismatch: exam center=%s, student center=%s\n", examCenterID, centerID)
		return nil, fmt.Errorf("student does not have access to this exam")
	}

	fmt.Printf("[DB] Fetching questions for exam %s\n", examID)

	// Fetch questions for the exam (including score)
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT id, exam_id, text, option_a, option_b, option_c, option_d, correct_option, score, created_at, updated_at 
		 FROM questions 
		 WHERE exam_id = $1 
		 ORDER BY created_at ASC`,
		examID)
	if err != nil {
		fmt.Printf("[DB] Query failed: %v\n", err)
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer rows.Close()

	var questions []*models.QuestionWithOptions
	questionCount := 0
	for rows.Next() {
		var q models.QuestionWithFlatOptions
		err := rows.Scan(
			&q.ID, &q.ExamID, &q.Text, &q.OptionA, &q.OptionB, &q.OptionC, &q.OptionD,
			&q.CorrectOption, &q.Score, &q.CreatedAt, &q.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("[DB] Scan failed: %v\n", err)
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}
		questionCount++

		// Format options as map for frontend
		questionWithOptions := &models.QuestionWithOptions{
			ID:   q.ID,
			Text: q.Text,
			Options: map[string]string{
				"A": q.OptionA,
				"B": q.OptionB,
				"C": q.OptionC,
				"D": q.OptionD,
			},
		}
		questions = append(questions, questionWithOptions)
	}

	fmt.Printf("[DB] Found %d questions for exam %s\n", questionCount, examID)

	return questions, nil
}

// GetAllStudentResults fetches all exam results for a student
func (s *ExamEngineService) GetAllStudentResults(studentID, centerID string) ([]map[string]interface{}, error) {
	fmt.Printf("[DB] Fetching all exam results for student %s\n", studentID)

	// Query exam_results joined with exams to get exam titles
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT 
			er.id,
			er.exam_id,
			er.score,
			er.total_questions,
			er.correct_answers,
			er.wrong_answers,
			er.unanswered_questions,
			er.time_spent_seconds,
			er.created_at,
			e.title as exam_title,
			e.duration_minutes
		 FROM exam_results er
		 INNER JOIN exams e ON er.exam_id = e.id
		 WHERE er.student_id = $1 AND e.center_id = $2
		 ORDER BY er.created_at DESC`,
		studentID, centerID)

	if err != nil {
		fmt.Printf("[DB] Query failed: %v\n", err)
		return nil, fmt.Errorf("failed to query exam results: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	resultCount := 0

	for rows.Next() {
		var (
			id, examID, examTitle      string
			score                      float64
			totalQuestions, correctAns int
			wrongAns, unansweredAns    int
			timeSpent                  int
			createdAt                  time.Time
			durationMinutes            int
		)

		err := rows.Scan(
			&id, &examID, &score, &totalQuestions, &correctAns,
			&wrongAns, &unansweredAns, &timeSpent, &createdAt,
			&examTitle, &durationMinutes,
		)
		if err != nil {
			fmt.Printf("[DB] Scan failed: %v\n", err)
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		result := map[string]interface{}{
			"id":                   id,
			"exam_id":              examID,
			"exam_title":           examTitle,
			"score":                score,
			"total_questions":      totalQuestions,
			"correct_answers":      correctAns,
			"wrong_answers":        wrongAns,
			"unanswered_questions": unansweredAns,
			"time_spent_seconds":   timeSpent,
			"created_at":           createdAt,
			"duration_minutes":     durationMinutes,
		}

		results = append(results, result)
		resultCount++
	}

	fmt.Printf("[DB] Found %d exam results for student %s\n", resultCount, studentID)

	return results, nil
}

// SaveAnswer saves or updates a student's answer (upsert operation)
func (s *ExamEngineService) SaveAnswer(studentID, examID, questionID, selectedOption string) (*models.StudentAnswerSimple, error) {
	// Check if answer already exists
	var existingAnswer models.StudentAnswerSimple
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT id, student_id, exam_id, question_id, selected_option, is_correct, created_at, updated_at 
		 FROM answers 
		 WHERE student_id = $1 AND question_id = $2`,
		studentID, questionID).Scan(
		&existingAnswer.ID, &existingAnswer.StudentID, &existingAnswer.ExamID,
		&existingAnswer.QuestionID, &existingAnswer.SelectedOption, &existingAnswer.IsCorrect,
		&existingAnswer.CreatedAt, &existingAnswer.UpdatedAt,
	)

	if err == nil {
		// Answer exists, update it
		return s.updateAnswer(existingAnswer.ID, selectedOption)
	} else {
		// Answer doesn't exist, create new one
		return s.createAnswer(studentID, examID, questionID, selectedOption)
	}
}

// createAnswer creates a new answer record
func (s *ExamEngineService) createAnswer(studentID, examID, questionID, selectedOption string) (*models.StudentAnswerSimple, error) {
	// Get the correct option for this question
	var correctOption string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT correct_option FROM questions WHERE id = $1`, questionID).Scan(&correctOption)
	if err != nil {
		return nil, fmt.Errorf("question not found: %w", err)
	}

	// Determine if answer is correct
	isCorrect := selectedOption == correctOption

	// Insert the answer
	var answer models.StudentAnswerSimple
	query := `INSERT INTO answers 
		(student_id, exam_id, question_id, selected_option, is_correct, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING id, student_id, exam_id, question_id, selected_option, is_correct, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		studentID, examID, questionID, selectedOption, isCorrect).Scan(
		&answer.ID, &answer.StudentID, &answer.ExamID, &answer.QuestionID,
		&answer.SelectedOption, &answer.IsCorrect, &answer.CreatedAt, &answer.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to save answer: %w", err)
	}

	return &answer, nil
}

// updateAnswer updates an existing answer
func (s *ExamEngineService) updateAnswer(answerID, selectedOption string) (*models.StudentAnswerSimple, error) {
	// Get question ID and correct option
	var questionID, correctOption string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT q.id, q.correct_option FROM questions q 
		 JOIN answers a ON q.id = a.question_id 
		 WHERE a.id = $1`, answerID).Scan(&questionID, &correctOption)
	if err != nil {
		return nil, fmt.Errorf("question not found: %w", err)
	}

	// Determine if answer is correct
	isCorrect := selectedOption == correctOption

	// Update the answer
	var answer models.StudentAnswerSimple
	query := `UPDATE answers 
		SET selected_option = $1, is_correct = $2, updated_at = NOW() 
		WHERE id = $3 
		RETURNING id, student_id, exam_id, question_id, selected_option, is_correct, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		selectedOption, isCorrect, answerID).Scan(
		&answer.ID, &answer.StudentID, &answer.ExamID, &answer.QuestionID,
		&answer.SelectedOption, &answer.IsCorrect, &answer.CreatedAt, &answer.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update answer: %w", err)
	}

	return &answer, nil
}

// SubmitExam submits the exam and calculates the result
func (s *ExamEngineService) SubmitExam(studentID, examID string, timeSpentSeconds int) (*models.ExamResultResponse, error) {
	// Start a transaction
	tx, err := config.DBConnection.Begin(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Get total number of questions in the exam
	var totalQuestions int
	err = tx.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM questions WHERE exam_id = $1`, examID).Scan(&totalQuestions)
	if err != nil {
		return nil, fmt.Errorf("failed to count questions: %w", err)
	}

	if totalQuestions == 0 {
		return nil, fmt.Errorf("exam has no questions")
	}

	// Get all student's answers for this exam
	rows, err := tx.Query(context.Background(),
		`SELECT selected_option FROM answers WHERE student_id = $1 AND exam_id = $2`,
		studentID, examID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch answers: %w", err)
	}
	defer rows.Close()

	answeredCount := 0
	correctCount := 0

	for rows.Next() {
		var selectedOption string
		err := rows.Scan(&selectedOption)
		if err != nil {
			return nil, fmt.Errorf("failed to scan answer: %w", err)
		}
		answeredCount++
	}

	// Get detailed question review data
	var questionReview []models.QuestionReview

	// Calculate correct answers by comparing with question table
	rows2, err := tx.Query(context.Background(),
		`SELECT q.id, q.text, q.correct_option, a.selected_option 
		 FROM questions q 
		 LEFT JOIN answers a ON q.id = a.question_id AND a.student_id = $1 AND a.exam_id = $2
		 WHERE q.exam_id = $3
		 ORDER BY q.created_at`,
		studentID, examID, examID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch correct answers: %w", err)
	}
	defer rows2.Close()

	correctCount = 0
	wrongCount := 0
	unansweredCount := 0

	for rows2.Next() {
		var qID, qText, correctOption string
		var selectedOption *string // Can be null if no answer

		err := rows2.Scan(&qID, &qText, &correctOption, &selectedOption)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comparison: %w", err)
		}

		isCorrect := selectedOption != nil && *selectedOption == correctOption

		if selectedOption != nil {
			if isCorrect {
				correctCount++
			} else {
				wrongCount++
			}
		} else {
			unansweredCount++
		}

		questionReview = append(questionReview, models.QuestionReview{
			QuestionID:     qID,
			Question:       qText,
			SelectedOption: selectedOption,
			CorrectOption:  correctOption,
			IsCorrect:      isCorrect,
		})
	}

	// Calculate metrics
	score := float64(correctCount) / float64(totalQuestions) * 100.0

	// Save exam result (upsert) using the transaction
	var result models.ExamResult
	resultQuery := `INSERT INTO exam_results 
		(student_id, exam_id, score, total_questions, correct_answers, wrong_answers, unanswered_questions, time_spent_seconds, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW()) 
		ON CONFLICT (student_id, exam_id) DO UPDATE SET
			score = EXCLUDED.score,
			total_questions = EXCLUDED.total_questions,
			correct_answers = EXCLUDED.correct_answers,
			wrong_answers = EXCLUDED.wrong_answers,
			unanswered_questions = EXCLUDED.unanswered_questions,
			time_spent_seconds = EXCLUDED.time_spent_seconds,
			created_at = NOW()
		RETURNING id, student_id, exam_id, score, total_questions, correct_answers, wrong_answers, unanswered_questions, time_spent_seconds, created_at`

	err = tx.QueryRow(context.Background(), resultQuery,
		studentID, examID, score, totalQuestions, correctCount, wrongCount, unansweredCount, timeSpentSeconds).Scan(
		&result.ID, &result.StudentID, &result.ExamID, &result.Score, &result.TotalQuestions,
		&result.CorrectAnswers, &result.WrongAnswers, &result.UnansweredQuestions,
		&result.TimeSpentSeconds, &result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to save exam result: %w", err)
	}

	// Commit transaction
	err = tx.Commit(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Build response with question review
	response := &models.ExamResultResponse{
		Score:               result.Score,
		CorrectAnswers:      result.CorrectAnswers,
		TotalQuestions:      result.TotalQuestions,
		WrongAnswers:        result.WrongAnswers,
		UnansweredQuestions: result.UnansweredQuestions,
		Percentage:          score,
		Passed:              score >= 50.0, // Passing threshold: 50%
		TimeSpent:           timeSpentSeconds,
		QuestionsReview:     questionReview,
	}

	fmt.Printf("[SubmitExam] Exam submitted - Score: %.2f%%, Correct: %d, Wrong: %d, Unanswered: %d\n",
		score, correctCount, wrongCount, unansweredCount)

	return response, nil
}

// GetStudentResult fetches the result for a specific exam attempt
func (s *ExamEngineService) GetStudentResult(studentID, examID string) (*models.ExamResult, error) {
	var result models.ExamResult
	query := `SELECT id, student_id, exam_id, score, total_questions, correct_answers, wrong_answers, unanswered_questions, time_spent_seconds, created_at 
		FROM exam_results 
		WHERE student_id = $1 AND exam_id = $2 
		ORDER BY created_at DESC 
		LIMIT 1`

	err := config.DBConnection.QueryRow(context.Background(), query, studentID, examID).Scan(
		&result.ID, &result.StudentID, &result.ExamID, &result.Score, &result.TotalQuestions,
		&result.CorrectAnswers, &result.WrongAnswers, &result.UnansweredQuestions,
		&result.TimeSpentSeconds, &result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("result not found: %w", err)
	}

	return &result, nil
}
