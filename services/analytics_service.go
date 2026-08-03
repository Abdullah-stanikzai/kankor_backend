package services

import (
	"context"
	"fmt"
	"kankor-backend/config"
)

// AnalyticsService handles analytics-related business logic
type AnalyticsService struct {
	config *config.Config
}

// NewAnalyticsService creates a new AnalyticsService instance
func NewAnalyticsService(cfg *config.Config) *AnalyticsService {
	return &AnalyticsService{
		config: cfg,
	}
}

// GetGlobalAnalytics gets global system analytics
func (as *AnalyticsService) GetGlobalAnalytics() (map[string]interface{}, error) {
	// Query database for global analytics
	var totalCenters, totalExams, totalStudents, totalAdmins, activeExams, completedExams int
	var avgStudentScore float64
	
	// Get total centers
	err := config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM educational_centers WHERE is_active = true`).Scan(&totalCenters)
	if err != nil {
		return nil, fmt.Errorf("failed to get total centers: %w", err)
	}
	
	// Get total exams
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM exams`).Scan(&totalExams)
	if err != nil {
		return nil, fmt.Errorf("failed to get total exams: %w", err)
	}
	
	// Get total students
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM users WHERE role = 'student'`).Scan(&totalStudents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total students: %w", err)
	}
	

	// Get total admins
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM users WHERE role IN ('super_admin', 'center_admin')`).Scan(&totalAdmins)
	if err != nil {
		return nil, fmt.Errorf("failed to get total admins: %w", err)
	}
	
	// Get active exams
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM exams WHERE is_published = true AND is_active = true`).Scan(&activeExams)
	if err != nil {
		return nil, fmt.Errorf("failed to get active exams: %w", err)
	}
	
	

	// Get completed exams
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM exams WHERE is_published = true AND start_time < NOW() AND end_time < NOW()`).Scan(&completedExams)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed exams: %w", err)
	}
	
	// Get average student score
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COALESCE(AVG(score), 0) FROM exam_attempts WHERE score IS NOT NULL`).Scan(&avgStudentScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get average student score: %w", err)
	}
	
	// Calculate average completion rate
	var avgCompletionRate float64
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COALESCE(AVG(CASE WHEN is_submitted = true THEN 100.0 ELSE 0 END), 0) FROM exam_attempts`).Scan(&avgCompletionRate)
	if err != nil {
		return nil, fmt.Errorf("failed to get average completion rate: %w", err)
	}
	
	// Get top performing center
	var topPerformingCenter string
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT ec.name FROM educational_centers ec 
		JOIN users u ON ec.id = u.center_id 
		JOIN exam_attempts ea ON u.id = ea.student_id
		GROUP BY ec.id, ec.name 
		ORDER BY AVG(ea.score) DESC 
		LIMIT 1`).Scan(&topPerformingCenter)
	if err != nil {
		// If no data available, set default
		topPerformingCenter = "No data available"
	}
	
	// Get recent activity (last 7 days)
	var last7DaysExams, last7DaysStudents, last7DaysSubmissions int
	
	// Recent exams
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM exams WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&last7DaysExams)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent exams: %w", err)
	}
	
	// Recent students
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM users WHERE role = 'student' AND created_at >= NOW() - INTERVAL '7 days'`).Scan(&last7DaysStudents)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent students: %w", err)
	}
	
	// Recent submissions
	err = config.DBConnection.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM exam_attempts WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&last7DaysSubmissions)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent submissions: %w", err)
	}
	
	analytics := map[string]interface{}{
		"total_centers":         totalCenters,
		"total_exams":           totalExams,
		"total_students":        totalStudents,
		"total_admins":          totalAdmins,
		"active_exams":          activeExams,
		"completed_exams":       completedExams,
		"avg_completion_rate":   avgCompletionRate,
		"avg_student_score":     avgStudentScore,
		"top_performing_center": topPerformingCenter,
		"recent_activity": map[string]interface{}{
			"last_7_days_exams":      last7DaysExams,
			"last_7_days_students":   last7DaysStudents,
			"last_7_days_submissions": last7DaysSubmissions,
		},
	}

	return analytics, nil
}

// GetCentersAnalytics gets analytics by center
func (as *AnalyticsService) GetCentersAnalytics() ([]map[string]interface{}, error) {
	// In a real implementation, we would query the database
	// For now, we'll return a mock response

	analytics := []map[string]interface{}{
		{
			"center_id":           "center-1",
			"center_name":         "Kabul Education Center",
			"total_students":      450,
			"total_exams":         35,
			"active_exams":        5,
			"avg_completion_rate": 92.3,
			"avg_score":           78.5,
		},
		{
			"center_id":           "center-2",
			"center_name":         "Herat Education Center",
			"total_students":      380,
			"total_exams":         28,
			"active_exams":        3,
			"avg_completion_rate": 87.6,
			"avg_score":           74.2,
		},
		{
			"center_id":           "center-3",
			"center_name":         "Mazar Education Center",
			"total_students":      320,
			"total_exams":         22,
			"active_exams":        4,
			"avg_completion_rate": 90.1,
			"avg_score":           79.8,
		},
	}

	return analytics, nil
}

// GetExamsAnalytics gets analytics by exam
func (as *AnalyticsService) GetExamsAnalytics() ([]map[string]interface{}, error) {
	// In a real implementation, we would query the database
	// For now, we'll return a mock response

	analytics := []map[string]interface{}{
		{
			"exam_id":             "exam-1",
			"exam_title":          "Mathematics Weekly Exam",
			"total_participants":   120,
			"completion_rate":     94.2,
			"avg_score":           78.5,
			"highest_score":       98.0,
			"lowest_score":        45.0,
			"pass_rate":           88.3,
			"most_difficult_q":    "Q15",
			"easiest_q":           "Q3",
		},
		{
			"exam_id":             "exam-2",
			"exam_title":          "Physics Weekly Exam",
			"total_participants":   95,
			"completion_rate":     89.5,
			"avg_score":           72.3,
			"highest_score":       95.0,
			"lowest_score":        38.0,
			"pass_rate":           82.1,
			"most_difficult_q":    "Q12",
			"easiest_q":           "Q7",
		},
	}

	return analytics, nil
}

// GetStudentsAnalytics gets analytics by student
func (as *AnalyticsService) GetStudentsAnalytics() (map[string]interface{}, error) {
	// In a real implementation, we would query the database
	// For now, we'll return a mock response

	analytics := map[string]interface{}{
		"total_students":           2450,
		"active_students":          1890,
		"avg_score_all_students":   76.8,
		"completion_rate":          89.5,
		"top_performers": []map[string]interface{}{
			{
				"student_id":  "student-101",
				"name":        "Ahmad Rahimi",
				"avg_score":   94.5,
				"exams_taken": 12,
			},
			{
				"student_id":  "student-205",
				"name":        "Fatima Karimi",
				"avg_score":   93.8,
				"exams_taken": 10,
			},
		},
		"distribution_by_score": map[string]interface{}{
			"90-100": 15.2,
			"80-89":  28.7,
			"70-79":  32.1,
			"60-69":  18.5,
			"below_60": 5.5,
		},
	}

	return analytics, nil
}

// GetCenterAnalytics gets analytics for a specific center
func (as *AnalyticsService) GetCenterAnalytics(centerID string) (map[string]interface{}, error) {
	// In a real implementation, we would query the database for specific center data
	// For now, we'll return a mock response

	analytics := map[string]interface{}{
		"center_id":              centerID,
		"center_name":            "Sample Center",
		"total_students":         450,
		"total_exams":            35,
		"active_exams":           5,
		"upcoming_exams":         3,
		"avg_completion_rate":    92.3,
		"avg_student_score":      78.5,
		"top_performing_student": "Ahmad Rahimi",
		"recent_activity": map[string]interface{}{
			"last_7_days_exams":      8,
			"last_7_days_students":   120,
			"last_7_days_submissions": 480,
		},
		"monthly_trends": []map[string]interface{}{
			{
				"month":        "January",
				"exams":        12,
				"participants": 420,
				"avg_score":    77.8,
			},
			{
				"month":        "December",
				"exams":        10,
				"participants": 380,
				"avg_score":    76.5,
			},
		},
	}

	return analytics, nil
}