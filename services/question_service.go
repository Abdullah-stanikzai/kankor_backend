package services

import (
	"context"
	"errors"
	"fmt"

	"kankor-backend/config"
	"kankor-backend/models"
)

// QuestionService handles question-related business logic
type QuestionService struct {
	config *config.Config
}

// NewQuestionService creates a new QuestionService instance
func NewQuestionService(cfg *config.Config) *QuestionService {
	return &QuestionService{
		config: cfg,
	}
}

// GetExamQuestions gets all questions for an exam with their options
func (qs *QuestionService) GetExamQuestions(examID, userID, centerID string) ([]*models.Question, error) {
	// First verify that the exam belongs to the user's center
	var examCenterID string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT center_id FROM exams WHERE id = $1 AND center_id = $2`, examID, centerID).Scan(&examCenterID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("exam not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify exam access: %w", err)
	}

	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT id, exam_id, question_text, question_order, score, created_at FROM questions WHERE exam_id = $1 ORDER BY question_order ASC`,
		examID)
	if err != nil {
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer rows.Close()

	var questions []*models.Question
	for rows.Next() {
		var question models.Question
		err := rows.Scan(
			&question.ID, &question.ExamID, &question.QuestionText, &question.QuestionOrder, &question.Score, &question.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}
		questions = append(questions, &question)
	}

	// Fetch options for each question
	for _, question := range questions {
		optionRows, err := config.DBConnection.Query(context.Background(),
			`SELECT id, question_id, option_text, option_letter, is_correct, option_order, created_at 
			FROM question_options WHERE question_id = $1 ORDER BY option_order ASC`,
			question.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to query options for question %s: %w", question.ID, err)
		}
		defer optionRows.Close()

		var options []models.QuestionOption
		for optionRows.Next() {
			var option models.QuestionOption
			err := optionRows.Scan(
				&option.ID, &option.QuestionID, &option.OptionText, &option.OptionLetter,
				&option.IsCorrect, &option.OptionOrder, &option.CreatedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan option: %w", err)
			}
			options = append(options, option)
		}
		// Add options to question (this would require modifying the Question struct to include options)
		// For now, we'll return the questions as is and let the controller handle the options
	}

	return questions, nil
}

// GetQuestionByID gets a question by ID
func (qs *QuestionService) GetQuestionByID(questionID, userID, centerID string) (*models.Question, error) {
	var question models.Question
	query := `SELECT q.id, q.exam_id, q.question_text, q.question_order, q.score, q.created_at 
			FROM questions q 
			JOIN exams e ON q.exam_id = e.id 
			WHERE q.id = $1 AND e.center_id = $2`

	err := config.DBConnection.QueryRow(context.Background(), query, questionID, centerID).Scan(
		&question.ID, &question.ExamID, &question.QuestionText,
		&question.QuestionOrder, &question.Score, &question.CreatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("question not found or not accessible")
		}
		return nil, fmt.Errorf("question not found: %w", err)
	}

	return &question, nil
}

// CreateQuestion creates a new question with its options
func (qs *QuestionService) CreateQuestion(req *models.CreateQuestionRequest, userID, centerID string) (*models.Question, error) {
	// First verify that the exam belongs to the user's center
	var examCenterID string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT center_id FROM exams WHERE id = $1 AND center_id = $2`, req.ExamID, centerID).Scan(&examCenterID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("exam not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify exam access: %w", err)
	}

	// Create the question first
	var newQuestion models.Question
	query := `INSERT INTO questions (exam_id, question_text, question_order, score, created_at) 
	VALUES ($1, $2, $3, $4, NOW()) 
	RETURNING id, exam_id, question_text, question_order, score, created_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		req.ExamID, req.QuestionText, req.QuestionOrder, req.Score,
	).Scan(
		&newQuestion.ID, &newQuestion.ExamID, &newQuestion.QuestionText,
		&newQuestion.QuestionOrder, &newQuestion.Score, &newQuestion.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create question: %w", err)
	}

	// Create the options for this question
	if len(req.Options) > 0 {
		for i, optionReq := range req.Options {
			optionQuery := `INSERT INTO question_options (question_id, option_text, option_letter, is_correct, option_order, created_at) 
			VALUES ($1, $2, $3, $4, $5, NOW())`

			_, err := config.DBConnection.Exec(context.Background(), optionQuery,
				newQuestion.ID, optionReq.OptionText, optionReq.OptionLetter, optionReq.IsCorrect, i)
			if err != nil {
				// If option creation fails, we should delete the question and return error
				config.DBConnection.Exec(context.Background(), "DELETE FROM questions WHERE id = $1", newQuestion.ID)
				return nil, fmt.Errorf("failed to create question option: %w", err)
			}
		}
	}

	return &newQuestion, nil
}

// UpdateQuestion updates a question
func (qs *QuestionService) UpdateQuestion(questionID string, req *models.CreateQuestionRequest, userID, centerID string) (*models.Question, error) {
	// First verify that the question belongs to an exam in the user's center
	var examCenterID string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT e.center_id FROM questions q JOIN exams e ON q.exam_id = e.id WHERE q.id = $1 AND e.center_id = $2`,
		questionID, centerID).Scan(&examCenterID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("question not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify question access: %w", err)
	}

	var question models.Question
	query := `UPDATE questions SET question_text=$1, question_order=$2, score=$3 WHERE id=$4 RETURNING id, exam_id, question_text, question_order, score, created_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		req.QuestionText, req.QuestionOrder, req.Score, questionID,
	).Scan(
		&question.ID, &question.ExamID, &question.QuestionText,
		&question.QuestionOrder, &question.Score, &question.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update question: %w", err)
	}

	// Update the options for this question
	if len(req.Options) > 0 {
		// First delete existing options
		_, err := config.DBConnection.Exec(context.Background(), "DELETE FROM question_options WHERE question_id = $1", questionID)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing question options: %w", err)
		}

		// Then create new options
		for i, optionReq := range req.Options {
			optionQuery := `INSERT INTO question_options (question_id, option_text, option_letter, is_correct, option_order, created_at) 
			VALUES ($1, $2, $3, $4, $5, NOW())`

			_, err := config.DBConnection.Exec(context.Background(), optionQuery,
				questionID, optionReq.OptionText, optionReq.OptionLetter, optionReq.IsCorrect, i)
			if err != nil {
				return nil, fmt.Errorf("failed to update question option: %w", err)
			}
		}
	}

	return &question, nil
}

// DeleteQuestion deletes a question
func (qs *QuestionService) DeleteQuestion(questionID, userID, centerID string) error {
	// First verify that the question belongs to an exam in the user's center
	var examCenterID string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT e.center_id FROM questions q JOIN exams e ON q.exam_id = e.id WHERE q.id = $1 AND e.center_id = $2`,
		questionID, centerID).Scan(&examCenterID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return errors.New("question not found or not accessible")
		}
		return fmt.Errorf("failed to verify question access: %w", err)
	}

	// First delete the question options
	_, err = config.DBConnection.Exec(context.Background(), "DELETE FROM question_options WHERE question_id = $1", questionID)
	if err != nil {
		return fmt.Errorf("failed to delete question options: %w", err)
	}

	query := `DELETE FROM questions WHERE id=$1`
	result, err := config.DBConnection.Exec(context.Background(), query, questionID)
	if err != nil {
		return fmt.Errorf("failed to delete question: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("question not found")
	}
	return nil
}
