package services

import (
	"context"
	"fmt"
	"time"

	"kankor-backend/config"
)

// CenterScore is one submitted exam result enriched with student and exam data.
type CenterScore struct {
	ID                  string    `json:"id"`
	StudentID           string    `json:"student_id"`
	StudentName         string    `json:"student_name"`
	StudentEmail        string    `json:"student_email"`
	ExamID              string    `json:"exam_id"`
	ExamTitle           string    `json:"exam_title"`
	Subject             string    `json:"subject"`
	Class               string    `json:"class"`
	Score               float64   `json:"score"`
	TotalQuestions      int       `json:"total_questions"`
	CorrectAnswers      int       `json:"correct_answers"`
	WrongAnswers        int       `json:"wrong_answers"`
	UnansweredQuestions int       `json:"unanswered_questions"`
	TimeSpentSeconds    int       `json:"time_spent_seconds"`
	CompletedAt         time.Time `json:"completed_at"`
	Passed              bool      `json:"passed"`
}

// CenterScoreSummary contains page-level totals calculated from the same
// center-scoped rows returned to the admin.
type CenterScoreSummary struct {
	Students     int     `json:"students"`
	Results      int     `json:"results"`
	AverageScore float64 `json:"average_score"`
	PassRate     float64 `json:"pass_rate"`
}

// CenterScoresResponse is the payload consumed by the center-admin scores page.
type CenterScoresResponse struct {
	Scores  []*CenterScore     `json:"scores"`
	Summary CenterScoreSummary `json:"summary"`
}

// ScoreService provides center-admin access to submitted student results.
type ScoreService struct {
	config *config.Config
}

func NewScoreService(cfg *config.Config) *ScoreService {
	return &ScoreService{config: cfg}
}

// GetCenterScores returns every submitted result belonging to the authenticated
// admin's center. When studentID is non-nil, results are restricted to that
// student. New exam-engine results take precedence over duplicate legacy rows.
func (s *ScoreService) GetCenterScores(centerID string, studentID *string) (*CenterScoresResponse, error) {
	var studentFilter interface{}
	if studentID != nil {
		studentFilter = *studentID
	}

	rows, err := config.DBConnection.Query(context.Background(), `
		WITH unified_results AS (
			SELECT
				er.id,
				u.id AS student_id,
				u.full_name AS student_name,
				u.email AS student_email,
				e.id AS exam_id,
				e.title AS exam_title,
				COALESCE(e.description, '') AS subject,
				COALESCE(e.class, '') AS class,
				er.score,
				er.total_questions,
				er.correct_answers,
				er.wrong_answers,
				er.unanswered_questions,
				COALESCE(er.time_spent_seconds, 0) AS time_spent_seconds,
				er.created_at AS completed_at
			FROM exam_results er
			JOIN users u ON u.id = er.student_id
			JOIN exams e ON e.id = er.exam_id
			WHERE e.center_id = $1
			  AND u.center_id = $1
			  AND u.role = 'student'
			  AND ($2::uuid IS NULL OR u.id = $2::uuid)

			UNION ALL

			SELECT
				ea.id,
				u.id AS student_id,
				u.full_name AS student_name,
				u.email AS student_email,
				e.id AS exam_id,
				e.title AS exam_title,
				COALESCE(e.description, '') AS subject,
				COALESCE(e.class, '') AS class,
				COALESCE(ea.score, 0) AS score,
				COALESCE(ea.total_questions, 0) AS total_questions,
				COALESCE(ea.correct_answers, 0) AS correct_answers,
				GREATEST(COALESCE(ea.total_questions, 0) - COALESCE(ea.correct_answers, 0), 0) AS wrong_answers,
				0 AS unanswered_questions,
				COALESCE(ea.time_taken_seconds, 0) AS time_spent_seconds,
				COALESCE(ea.submit_time, ea.created_at) AS completed_at
			FROM exam_attempts ea
			JOIN users u ON u.id = ea.student_id
			JOIN exams e ON e.id = ea.exam_id
			WHERE ea.is_submitted = true
			  AND e.center_id = $1
			  AND u.center_id = $1
			  AND u.role = 'student'
			  AND ($2::uuid IS NULL OR u.id = $2::uuid)
			  AND NOT EXISTS (
				SELECT 1 FROM exam_results er
				WHERE er.student_id = ea.student_id AND er.exam_id = ea.exam_id
			  )
		)
		SELECT id, student_id, student_name, student_email, exam_id, exam_title,
			subject, class, score, total_questions, correct_answers, wrong_answers,
			unanswered_questions, time_spent_seconds, completed_at
		FROM unified_results
		ORDER BY completed_at DESC`, centerID, studentFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to query center scores: %w", err)
	}
	defer rows.Close()

	scores := make([]*CenterScore, 0)
	studentIDs := make(map[string]struct{})
	totalScore := 0.0
	passedCount := 0

	for rows.Next() {
		var score CenterScore
		if err := rows.Scan(
			&score.ID,
			&score.StudentID,
			&score.StudentName,
			&score.StudentEmail,
			&score.ExamID,
			&score.ExamTitle,
			&score.Subject,
			&score.Class,
			&score.Score,
			&score.TotalQuestions,
			&score.CorrectAnswers,
			&score.WrongAnswers,
			&score.UnansweredQuestions,
			&score.TimeSpentSeconds,
			&score.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan center score: %w", err)
		}

		score.Passed = score.Score >= 50
		if score.Passed {
			passedCount++
		}
		totalScore += score.Score
		studentIDs[score.StudentID] = struct{}{}
		scores = append(scores, &score)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading center scores: %w", err)
	}

	summary := CenterScoreSummary{
		Students: len(studentIDs),
		Results:  len(scores),
	}
	if len(scores) > 0 {
		summary.AverageScore = totalScore / float64(len(scores))
		summary.PassRate = float64(passedCount) * 100 / float64(len(scores))
	}

	return &CenterScoresResponse{Scores: scores, Summary: summary}, nil
}
