package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"
)

// ExamService handles exam-related business logic
type ExamService struct {
	config *config.Config
}

// NewExamService creates a new ExamService instance
func NewExamService(cfg *config.Config) *ExamService {
	return &ExamService{
		config: cfg,
	}
}

// GetAllExams gets all exams for a center admin
func (es *ExamService) GetAllExams(userID, centerID string) ([]*models.Exam, error) {
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT id, title, description, center_id, creator_id, start_time, end_time, duration_minutes, is_published, is_active, created_at, updated_at FROM exams WHERE center_id = $1 ORDER BY created_at DESC`,
		centerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query exams: %w", err)
	}
	defer rows.Close()

	var exams []*models.Exam = make([]*models.Exam, 0) // Initialize as empty slice instead of nil
	for rows.Next() {
		var exam models.Exam
		err := rows.Scan(
			&exam.ID, &exam.Title, &exam.Description, &exam.CenterID,
			&exam.CreatorID, &exam.StartTime, &exam.EndTime,
			&exam.DurationMinutes, &exam.IsPublished, &exam.IsActive,
			&exam.CreatedAt, &exam.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exam: %w", err)
		}
		exams = append(exams, &exam)
	}

	return exams, nil
}

// GetExamByID gets an exam by ID
func (es *ExamService) GetExamByID(examID, userID, centerID string) (*models.Exam, error) {
	var exam models.Exam
	query := `SELECT id, title, description, center_id, creator_id, start_time, end_time, duration_minutes, is_published, is_active, created_at, updated_at FROM exams WHERE id = $1 AND center_id = $2`

	err := config.DBConnection.QueryRow(context.Background(), query, examID, centerID).Scan(
		&exam.ID, &exam.Title, &exam.Description, &exam.CenterID,
		&exam.CreatorID, &exam.StartTime, &exam.EndTime,
		&exam.DurationMinutes, &exam.IsPublished, &exam.IsActive,
		&exam.CreatedAt, &exam.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("exam not found: %w", err)
	}

	return &exam, nil
}

// CreateExam creates a new exam
func (es *ExamService) CreateExam(req *models.CreateExamRequest, userID string) (*models.Exam, error) {
	// Parse date and time
	dateLayout := "2006-01-02"
	timeLayout := "15:04"

	examDate, err := time.Parse(dateLayout, req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
	}

	startTimeParsed, err := time.Parse(timeLayout, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid time format (expected HH:MM): %w", err)
	}

	// Combine date and time in Afghanistan timezone (UTC+4:30)
	// Create a fixed offset timezone for Afghanistan (UTC+4:30)
	afghanistanOffset := 4*3600 + 30*60 // 4 hours 30 minutes in seconds
	afghanistanTZ := time.FixedZone("AFT", afghanistanOffset)

	startDateTime := time.Date(
		examDate.Year(), examDate.Month(), examDate.Day(),
		startTimeParsed.Hour(), startTimeParsed.Minute(), 0, 0,
		afghanistanTZ,
	)

	// Calculate end time
	endDateTime := startDateTime.Add(time.Duration(req.DurationMinutes) * time.Minute)

	// Default status to "pending" if not provided
	status := req.Status
	if status == "" {
		status = "pending"
	}

	// Log the request data for debugging
	fmt.Printf("[CreateExam] Creating exam with: Title=%s, CenterID=%s, UserID=%s, Date=%s, StartTime=%s, Duration=%d, Status=%s\n",
		req.Title, req.CenterID, userID, req.Date, req.StartTime, req.DurationMinutes, status)
	fmt.Printf("[CreateExam] startDateTime: %v\n", startDateTime)
	fmt.Printf("[CreateExam] startDateTime.UTC(): %v\n", startDateTime.UTC())

	var newExam models.Exam
	query := `INSERT INTO exams (title, description, center_id, creator_id, start_time, end_time, duration_minutes, is_published, is_active, status, created_at, updated_at) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()) 
	RETURNING id, title, description, center_id, creator_id, start_time, end_time, duration_minutes, is_published, is_active, status, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		req.Title, req.Description, req.CenterID, userID,
		startDateTime, endDateTime, req.DurationMinutes,
		status == "published", true, // is_published based on status, is_active = true
		status,
	).Scan(
		&newExam.ID, &newExam.Title, &newExam.Description, &newExam.CenterID,
		&newExam.CreatorID, &newExam.StartTime, &newExam.EndTime,
		&newExam.DurationMinutes, &newExam.IsPublished, &newExam.IsActive,
		&newExam.Status, &newExam.CreatedAt, &newExam.UpdatedAt,
	)
	if err != nil {
		// Log the specific error for debugging
		fmt.Printf("Database error in CreateExam: %v\n", err)
		return nil, fmt.Errorf("failed to create exam: %w", err)
	}

	fmt.Printf("Successfully created exam with ID: %s, Status: %s\n", newExam.ID, newExam.Status)
	return &newExam, nil
}

// UpdateExam updates an existing exam
func (es *ExamService) UpdateExam(examID string, req *models.CreateExamRequest, userID string) (*models.Exam, error) {
	// Parse date and time
	dateLayout := "2006-01-02"
	timeLayout := "15:04"

	examDate, err := time.Parse(dateLayout, req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
	}

	startTimeParsed, err := time.Parse(timeLayout, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid time format (expected HH:MM): %w", err)
	}

	// Combine date and time in Afghanistan timezone (UTC+4:30)
	// Create a fixed offset timezone for Afghanistan (UTC+4:30)
	afghanistanOffset := 4*3600 + 30*60 // 4 hours 30 minutes in seconds
	afghanistanTZ := time.FixedZone("AFT", afghanistanOffset)

	startDateTime := time.Date(
		examDate.Year(), examDate.Month(), examDate.Day(),
		startTimeParsed.Hour(), startTimeParsed.Minute(), 0, 0,
		afghanistanTZ,
	)

	// Calculate end time
	endDateTime := startDateTime.Add(time.Duration(req.DurationMinutes) * time.Minute)

	// Default status to "pending" if not provided
	status := req.Status
	if status == "" {
		status = "pending"
	}

	var updatedExam models.Exam

	// First get the current exam to get the center_id
	var currentCenterID string
	getQuery := `SELECT center_id FROM exams WHERE id = $1`
	err = config.DBConnection.QueryRow(context.Background(), getQuery, examID).Scan(&currentCenterID)
	if err != nil {
		return nil, fmt.Errorf("failed to find exam: %w", err)
	}

	// Log for debugging
	fmt.Printf("Updating exam %s for center %s\n", examID, currentCenterID)
	fmt.Printf("Update request: Title=%s, Date=%s, StartTime=%s, Duration=%d, Status=%s\n",
		req.Title, req.Date, req.StartTime, req.DurationMinutes, status)

	query := `UPDATE exams 
	SET title = $1, description = $2, start_time = $3, end_time = $4, duration_minutes = $5, is_published = $6, status = $7, updated_at = NOW() 
	WHERE id = $8 AND center_id = $9 
	RETURNING id, title, description, center_id, creator_id, start_time, end_time, duration_minutes, is_published, is_active, status, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		req.Title, req.Description, startDateTime, endDateTime,
		req.DurationMinutes, status == "published", status, examID, currentCenterID,
	).Scan(
		&updatedExam.ID, &updatedExam.Title, &updatedExam.Description, &updatedExam.CenterID,
		&updatedExam.CreatorID, &updatedExam.StartTime, &updatedExam.EndTime,
		&updatedExam.DurationMinutes, &updatedExam.IsPublished, &updatedExam.IsActive,
		&updatedExam.Status, &updatedExam.CreatedAt, &updatedExam.UpdatedAt,
	)

	if err != nil {
		fmt.Printf("Database error in UpdateExam: %v\n", err)
		return nil, fmt.Errorf("failed to update exam: %w", err)
	}

	fmt.Printf("Successfully updated exam with ID: %s, Status: %s\n", updatedExam.ID, updatedExam.Status)
	return &updatedExam, nil
}

// DeleteExam deletes an exam
func (es *ExamService) DeleteExam(examID, centerID string) error {
	query := "DELETE FROM exams WHERE id = $1 AND center_id = $2"

	result, err := config.DBConnection.Exec(context.Background(), query, examID, centerID)
	if err != nil {
		return fmt.Errorf("failed to delete exam: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("exam not found")
	}

	return nil
}

// UpdateExamStatus updates the status of an exam (pending/published)
func (es *ExamService) UpdateExamStatus(examID, status string) error {
	// Validate status
	if status != "pending" && status != "published" {
		return errors.New("invalid status: must be 'pending' or 'published'")
	}

	query := `UPDATE exams SET status = $1, is_published = $2, updated_at = NOW() WHERE id = $3`

	result, err := config.DBConnection.Exec(context.Background(), query, status, status == "published", examID)
	if err != nil {
		return fmt.Errorf("failed to update exam status: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("exam not found")
	}

	fmt.Printf("[UpdateExamStatus] Exam %s status updated to: %s\n", examID, status)
	return nil
}

// PublishExam publishes an exam
func (es *ExamService) PublishExam(examID, centerID string) (*models.Exam, error) {
	return es.updateExamPublishStatus(examID, centerID, true)
}

// UnpublishExam unpublishes an exam
func (es *ExamService) UnpublishExam(examID, centerID string) (*models.Exam, error) {
	return es.updateExamPublishStatus(examID, centerID, false)
}

// Helper function to update exam publish status
func (es *ExamService) updateExamPublishStatus(examID, centerID string, isPublished bool) (*models.Exam, error) {
	var exam models.Exam
	query := `UPDATE exams SET is_published = $1, updated_at = NOW() WHERE id = $2 AND center_id = $3 RETURNING id, title, description, center_id, creator_id, start_time, end_time, duration_minutes, is_published, is_active, created_at, updated_at`

	err := config.DBConnection.QueryRow(context.Background(), query, isPublished, examID, centerID).Scan(
		&exam.ID, &exam.Title, &exam.Description, &exam.CenterID,
		&exam.CreatorID, &exam.StartTime, &exam.EndTime,
		&exam.DurationMinutes, &exam.IsPublished, &exam.IsActive,
		&exam.CreatedAt, &exam.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update exam publish status: %w", err)
	}

	return &exam, nil
}

// GetExamResults gets all results for an exam
func (es *ExamService) GetExamResults(examID, centerID string) ([]*models.ExamAttempt, error) {
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT id, exam_id, student_id, start_time, submit_time, time_taken_seconds, score, total_questions, correct_answers, is_submitted, is_timed_out, created_at 
		FROM exam_attempts 
		WHERE exam_id = $1 
		ORDER BY created_at DESC`,
		examID)
	if err != nil {
		return nil, fmt.Errorf("failed to query exam results: %w", err)
	}
	defer rows.Close()

	var results []*models.ExamAttempt
	for rows.Next() {
		var result models.ExamAttempt
		err := rows.Scan(
			&result.ID, &result.ExamID, &result.StudentID,
			&result.StartTime, &result.SubmitTime, &result.TimeTakenSeconds,
			&result.Score, &result.TotalQuestions, &result.CorrectAnswers,
			&result.IsSubmitted, &result.IsTimedOut, &result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exam result: %w", err)
		}
		results = append(results, &result)
	}

	return results, nil
}

// GetExamResult gets a specific exam attempt result
func (es *ExamService) GetExamResult(examID, attemptID, centerID string) (*models.ExamAttempt, error) {
	var result models.ExamAttempt
	query := `SELECT id, exam_id, student_id, start_time, submit_time, time_taken_seconds, score, total_questions, correct_answers, is_submitted, is_timed_out, created_at 
	FROM exam_attempts WHERE id = $1 AND exam_id = $2`

	err := config.DBConnection.QueryRow(context.Background(), query, attemptID, examID).Scan(
		&result.ID, &result.ExamID, &result.StudentID,
		&result.StartTime, &result.SubmitTime, &result.TimeTakenSeconds,
		&result.Score, &result.TotalQuestions, &result.CorrectAnswers,
		&result.IsSubmitted, &result.IsTimedOut, &result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("exam result not found: %w", err)
	}

	return &result, nil
}

// GetUpcomingExams gets upcoming exams for a student's center
func (es *ExamService) GetUpcomingExams(studentID, centerID string) ([]*models.Exam, error) {
	// TEMPORARY FIX: Remove strict filtering to allow debugging
	// Return all exams for the student's center, even if not published/active or has 0 questions
	// Also handle case where student_center_enrollment might not exist
	fmt.Printf("[GetUpcomingExams] Querying for studentID=%s, centerID=%s\n", studentID, centerID)

	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT e.id, e.title, e.description, e.center_id, e.creator_id, e.start_time, e.end_time, e.duration_minutes, e.is_published, e.is_active, e.created_at, e.updated_at 
		FROM exams e 
		LEFT JOIN student_center_enrollment sce ON e.center_id = sce.center_id AND sce.student_id = $1 AND sce.is_active = true
		WHERE (sce.student_id = $1 AND sce.is_active = true) OR e.center_id = $2
		ORDER BY e.start_time ASC`,
		studentID, centerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming exams: %w", err)
	}
	defer rows.Close()

	var exams []*models.Exam
	for rows.Next() {
		var exam models.Exam
		err := rows.Scan(
			&exam.ID, &exam.Title, &exam.Description, &exam.CenterID,
			&exam.CreatorID, &exam.StartTime, &exam.EndTime,
			&exam.DurationMinutes, &exam.IsPublished, &exam.IsActive,
			&exam.CreatedAt, &exam.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exam: %w", err)
		}
		exams = append(exams, &exam)
	}

	fmt.Printf("[GetUpcomingExams] Found %d exams for student %s\n", len(exams), studentID)
	return exams, nil
}

// GetAvailableExams gets available (published) exams for a student's center
func (es *ExamService) GetAvailableExams(studentID, centerID string) ([]*models.Exam, error) {
	// Only return published exams that belong to student's center and are upcoming/current
	fmt.Printf("[GetAvailableExams] Querying for studentID=%s, centerID=%s\n", studentID, centerID)

	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT e.id, e.title, e.description, e.center_id, e.creator_id, e.start_time, e.end_time, e.duration_minutes, e.is_published, e.is_active, e.status, e.created_at, e.updated_at 
		FROM exams e 
		JOIN student_center_enrollment sce ON e.center_id = sce.center_id 
		WHERE sce.student_id = $1 
		  AND sce.is_active = true
		  AND e.status = 'published'
		  AND e.is_active = true
		  AND e.start_time >= NOW()
		ORDER BY e.start_time ASC`,
		studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available exams: %w", err)
	}
	defer rows.Close()

	var exams []*models.Exam
	for rows.Next() {
		var exam models.Exam
		err := rows.Scan(
			&exam.ID, &exam.Title, &exam.Description, &exam.CenterID,
			&exam.CreatorID, &exam.StartTime, &exam.EndTime,
			&exam.DurationMinutes, &exam.IsPublished, &exam.IsActive,
			&exam.Status, &exam.CreatedAt, &exam.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exam: %w", err)
		}
		exams = append(exams, &exam)
	}

	fmt.Printf("[GetAvailableExams] Found %d exams for student %s\n", len(exams), studentID)
	return exams, nil
}

// GetExamHistory gets a student's exam history
func (es *ExamService) GetExamHistory(studentID, centerID string) ([]*models.ExamAttempt, error) {
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT ea.id, ea.exam_id, ea.student_id, ea.start_time, ea.submit_time, ea.time_taken_seconds, ea.score, ea.total_questions, ea.correct_answers, ea.is_submitted, ea.is_timed_out, ea.created_at 
		FROM exam_attempts ea 
		JOIN exams e ON ea.exam_id = e.id 
		JOIN student_center_enrollment sce ON e.center_id = sce.center_id 
		WHERE sce.student_id = $1 AND sce.is_active = true AND ea.is_submitted = true 
		ORDER BY ea.created_at DESC`,
		studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query exam history: %w", err)
	}
	defer rows.Close()

	var attempts []*models.ExamAttempt
	for rows.Next() {
		var attempt models.ExamAttempt
		err := rows.Scan(
			&attempt.ID, &attempt.ExamID, &attempt.StudentID,
			&attempt.StartTime, &attempt.SubmitTime, &attempt.TimeTakenSeconds,
			&attempt.Score, &attempt.TotalQuestions, &attempt.CorrectAnswers,
			&attempt.IsSubmitted, &attempt.IsTimedOut, &attempt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exam attempt: %w", err)
		}
		attempts = append(attempts, &attempt)
	}

	return attempts, nil
}

// HasStudentSubmittedExam checks if a student has already submitted a specific exam
func (es *ExamService) HasStudentSubmittedExam(examID, studentID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM exam_attempts WHERE exam_id = $1 AND student_id = $2 AND is_submitted = true`

	err := config.DBConnection.QueryRow(context.Background(), query, examID, studentID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check exam submission status: %w", err)
	}

	fmt.Printf("[HasStudentSubmittedExam] Exam %s, Student %s: submitted=%v\n", examID, studentID, count > 0)
	return count > 0, nil
}

// GetExamDetails gets exam details for a student
func (es *ExamService) GetExamDetails(examID, studentID, centerID string) (map[string]interface{}, error) {
	var exam models.Exam
	query := `SELECT e.id, e.title, e.description, e.center_id, e.creator_id, e.start_time, e.end_time, e.duration_minutes, e.is_published, e.is_active, e.status, e.created_at, e.updated_at 
	FROM exams e 
	JOIN student_center_enrollment sce ON e.center_id = sce.center_id 
	WHERE e.id = $1 AND sce.student_id = $2 AND sce.is_active = true AND e.status = 'published' AND e.is_active = true`

	err := config.DBConnection.QueryRow(context.Background(), query, examID, studentID).Scan(
		&exam.ID, &exam.Title, &exam.Description, &exam.CenterID,
		&exam.CreatorID, &exam.StartTime, &exam.EndTime,
		&exam.DurationMinutes, &exam.IsPublished, &exam.IsActive,
		&exam.Status, &exam.CreatedAt, &exam.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("exam not found or not available: %w", err)
	}

	// Fetch question count and total score
	var totalQuestions int
	var totalScore float64

	countQuery := `SELECT COUNT(*), COALESCE(SUM(score), 0) FROM questions WHERE exam_id = $1`
	err = config.DBConnection.QueryRow(context.Background(), countQuery, examID).Scan(&totalQuestions, &totalScore)

	if err != nil {
		fmt.Printf("[GetExamDetails] Error fetching question count: %v\n", err)
		totalQuestions = 0
		totalScore = 0
	}

	fmt.Printf("[GetExamDetails] Exam %s: %d questions, %.1f total score\n", examID, totalQuestions, totalScore)

	// Return exam with additional fields
	return map[string]interface{}{
		"id":               exam.ID,
		"title":            exam.Title,
		"description":      exam.Description,
		"center_id":        exam.CenterID,
		"creator_id":       exam.CreatorID,
		"start_time":       exam.StartTime,
		"end_time":         exam.EndTime,
		"duration_minutes": exam.DurationMinutes,
		"is_published":     exam.IsPublished,
		"is_active":        exam.IsActive,
		"created_at":       exam.CreatedAt,
		"updated_at":       exam.UpdatedAt,
		"total_questions":  totalQuestions,
		"total_score":      totalScore,
	}, nil
}

// StartExam creates a new exam attempt for a student
func (es *ExamService) StartExam(examID, studentID string) (*models.ExamAttempt, error) {
	var attempt models.ExamAttempt
	query := `INSERT INTO exam_attempts (exam_id, student_id, start_time, is_submitted, is_timed_out, created_at) 
	VALUES ($1, $2, NOW(), false, false, NOW()) 
	RETURNING id, exam_id, student_id, start_time, submit_time, time_taken_seconds, score, total_questions, correct_answers, is_submitted, is_timed_out, created_at`

	err := config.DBConnection.QueryRow(context.Background(), query, examID, studentID).Scan(
		&attempt.ID, &attempt.ExamID, &attempt.StudentID,
		&attempt.StartTime, &attempt.SubmitTime, &attempt.TimeTakenSeconds,
		&attempt.Score, &attempt.TotalQuestions, &attempt.CorrectAnswers,
		&attempt.IsSubmitted, &attempt.IsTimedOut, &attempt.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to start exam: %w", err)
	}

	return &attempt, nil
}

// SubmitExam submits an exam and calculates the score
func (es *ExamService) SubmitExam(examID, attemptID, studentID string) (map[string]interface{}, error) {
	// Get the exam attempt
	var attempt models.ExamAttempt
	query := `SELECT id, exam_id, student_id, start_time, created_at FROM exam_attempts WHERE id = $1 AND exam_id = $2 AND student_id = $3`

	err := config.DBConnection.QueryRow(context.Background(), query, attemptID, examID, studentID).Scan(
		&attempt.ID, &attempt.ExamID, &attempt.StudentID,
		&attempt.StartTime, &attempt.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("exam attempt not found: %w", err)
	}

	// Calculate time taken
	timeTaken := int(time.Now().Sub(attempt.StartTime).Seconds())
	attempt.TimeTakenSeconds = &timeTaken
	attempt.SubmitTime = &time.Time{}
	*attempt.SubmitTime = time.Now()
	attempt.IsSubmitted = true

	// Get questions for the exam
	questions, err := es.GetExamQuestions(examID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exam questions: %w", err)
	}

	attempt.TotalQuestions = len(questions)
	correctAnswers := 0
	wrongAnswers := 0
	unanswered := 0

	// Build question review data
	var questionReview []map[string]interface{}

	// Grade the exam
	for i, questionMap := range questions {
		// Get question ID from the map
		questionID, ok := questionMap["id"].(string)
		if !ok {
			fmt.Printf("[SubmitExam] Failed to get ID from question %d\n", i)
			continue
		}

		// Get question text
		questionText, _ := questionMap["text"].(string)

		// Get student's answer for this question
		var studentAnswer models.StudentAnswer
		answerQuery := `SELECT id, attempt_id, question_id, selected_option_id, is_correct FROM student_answers WHERE attempt_id = $1 AND question_id = $2`
		err = config.DBConnection.QueryRow(context.Background(), answerQuery, attemptID, questionID).Scan(
			&studentAnswer.ID, &studentAnswer.AttemptID, &studentAnswer.QuestionID,
			&studentAnswer.SelectedOptionID, &studentAnswer.IsCorrect,
		)

		// Get correct answer from question (we need to fetch it)
		var correctOption string
		correctQuery := `SELECT correct_option FROM questions WHERE id = $1`
		err = config.DBConnection.QueryRow(context.Background(), correctQuery, questionID).Scan(&correctOption)
		if err != nil {
			correctOption = "" // If no correct_option column, leave empty
		}

		reviewItem := map[string]interface{}{
			"question_id":     questionID,
			"question":        questionText,
			"selected_option": nil,
			"correct_option":  correctOption,
			"is_correct":      false,
		}

		if err == nil && studentAnswer.SelectedOptionID != nil && *studentAnswer.SelectedOptionID != "" {
			// Student answered this question
			reviewItem["selected_option"] = *studentAnswer.SelectedOptionID

			if studentAnswer.IsCorrect != nil && *studentAnswer.IsCorrect {
				correctAnswers++
				reviewItem["is_correct"] = true
			} else {
				wrongAnswers++
			}
		} else {
			// Student didn't answer this question
			unanswered++
		}

		questionReview = append(questionReview, reviewItem)
	}

	attempt.CorrectAnswers = &correctAnswers

	// Calculate score (percentage)
	score := float64(correctAnswers) / float64(attempt.TotalQuestions) * 100
	attempt.Score = &score

	// Update the exam attempt
	updateQuery := `UPDATE exam_attempts SET submit_time = $1, time_taken_seconds = $2, score = $3, correct_answers = $4, is_submitted = true WHERE id = $5`
	_, err = config.DBConnection.Exec(context.Background(), updateQuery, attempt.SubmitTime, timeTaken, score, correctAnswers, attemptID)

	if err != nil {
		return nil, fmt.Errorf("failed to update exam attempt: %w", err)
	}

	// Build full result response
	result := map[string]interface{}{
		"score":            score,
		"correct_answers":  correctAnswers,
		"wrong_answers":    wrongAnswers,
		"unanswered":       unanswered,
		"total_questions":  attempt.TotalQuestions,
		"time_spent":       timeTaken,
		"questions_review": questionReview,
	}

	fmt.Printf("[SubmitExam] Exam submitted - Score: %.2f%%, Correct: %d, Wrong: %d, Unanswered: %d\n",
		score, correctAnswers, wrongAnswers, unanswered)

	return result, nil
}

// GetExamQuestions gets all questions for an exam
func (es *ExamService) GetExamQuestions(examID, studentID string) ([]map[string]interface{}, error) {
	fmt.Printf("[ExamService] GetExamQuestions - Exam: %s, Student: %s\n", examID, studentID)

	// First check if exam exists
	var examExists bool
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM exams WHERE id = $1)`, examID).Scan(&examExists)
	if err != nil {
		fmt.Printf("[ExamService] Error checking exam existence: %v\n", err)
		return nil, fmt.Errorf("failed to check exam existence: %w", err)
	}

	if !examExists {
		fmt.Printf("[ExamService] Exam %s not found\n", examID)
		return nil, fmt.Errorf("exam not found")
	}

	fmt.Printf("[ExamService] Exam %s exists, fetching questions with options...\n", examID)

	// Fetch questions with options from the same table (old schema)
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT 
			id,
			text,
			option_a,
			option_b,
			option_c,
			option_d
		FROM questions 
		WHERE exam_id = $1 
		ORDER BY created_at`,
		examID)
	if err != nil {
		fmt.Printf("[ExamService] Query error: %v\n", err)
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer rows.Close()

	var questions []map[string]interface{}
	questionCount := 0
	for rows.Next() {
		var id, text, optionA, optionB, optionC, optionD string

		err := rows.Scan(
			&id,
			&text,
			&optionA,
			&optionB,
			&optionC,
			&optionD,
		)
		if err != nil {
			fmt.Printf("[ExamService] Scan error: %v\n", err)
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}

		// Build options map
		options := map[string]string{
			"A": optionA,
			"B": optionB,
			"C": optionC,
			"D": optionD,
		}

		question := map[string]interface{}{
			"id":      id,
			"text":    text,
			"options": options,
		}

		questionCount++
		questions = append(questions, question)
	}

	fmt.Printf("[ExamService] Found %d questions for exam %s\n", questionCount, examID)

	// Don't return error for empty questions - let the controller handle it
	return questions, nil
}

// GetExamAnalytics gets analytics for an exam
func (es *ExamService) GetExamAnalytics(examID, centerID string) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Get total attempts
	var totalAttempts int
	query := `SELECT COUNT(*) FROM exam_attempts WHERE exam_id = $1`
	err := config.DBConnection.QueryRow(context.Background(), query, examID).Scan(&totalAttempts)
	if err != nil {
		return nil, fmt.Errorf("failed to get total attempts: %w", err)
	}
	analytics["total_attempts"] = totalAttempts

	// Get average score
	var averageScore float64
	query = `SELECT AVG(score) FROM exam_attempts WHERE exam_id = $1 AND is_submitted = true`
	err = config.DBConnection.QueryRow(context.Background(), query, examID).Scan(&averageScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get average score: %w", err)
	}
	analytics["average_score"] = averageScore

	// Get pass rate (assuming 50% is passing)
	var passRate float64
	query = `SELECT COUNT(*) * 100.0 / NULLIF(COUNT(*), 0) FROM exam_attempts WHERE exam_id = $1 AND is_submitted = true AND score >= 50`
	err = config.DBConnection.QueryRow(context.Background(), query, examID).Scan(&passRate)
	if err != nil {
		return nil, fmt.Errorf("failed to get pass rate: %w", err)
	}
	analytics["pass_rate"] = passRate

	// Get highest and lowest scores
	var highestScore, lowestScore float64
	query = `SELECT MAX(score), MIN(score) FROM exam_attempts WHERE exam_id = $1 AND is_submitted = true`
	err = config.DBConnection.QueryRow(context.Background(), query, examID).Scan(&highestScore, &lowestScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get score range: %w", err)
	}
	analytics["highest_score"] = highestScore
	analytics["lowest_score"] = lowestScore

	return analytics, nil
}
