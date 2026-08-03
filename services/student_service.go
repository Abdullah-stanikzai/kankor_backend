package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/utils"
)

// StudentService handles student-related business logic
type StudentService struct {
	config *config.Config
}

// NewStudentService creates a new StudentService instance
func NewStudentService(cfg *config.Config) *StudentService {
	return &StudentService{
		config: cfg,
	}
}

// GetAllStudents gets all students in a center
func (ss *StudentService) GetAllStudents(centerID string) ([]*models.User, error) {
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT u.id, u.email, u.full_name, u.phone_number, u.role, u.is_active, u.created_at, u.updated_at 
		 FROM users u 
		 WHERE u.center_id = $1 AND u.role = 'student' AND u.is_active = true 
		 ORDER BY u.created_at DESC`,
		centerID)
	if err != nil {
		fmt.Printf("[GetAllStudents] Failed to query students for centerID %s: %v\n", centerID, err)
		return nil, fmt.Errorf("failed to query students: %w", err)
	}
	defer rows.Close()

	var students []*models.User
	for rows.Next() {
		var student models.User
		var phoneNumber *string
		err := rows.Scan(&student.ID, &student.Email, &student.FullName, &phoneNumber,
			&student.Role, &student.IsActive, &student.CreatedAt, &student.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan student: %w", err)
		}
		if phoneNumber != nil {
			student.PhoneNumber = *phoneNumber
		}
		students = append(students, &student)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating students: %w", err)
	}

	return students, nil
}

// GetStudentByID gets a student by ID
func (ss *StudentService) GetStudentByID(studentID, centerID string) (*models.User, error) {
	if studentID == "" {
		return nil, errors.New("student ID is required")
	}

	var student models.User
	var phoneNumber *string
	query := `SELECT u.id, u.email, u.full_name, u.phone_number, u.role, u.is_active, u.created_at, u.updated_at 
			FROM users u 
			WHERE u.id = $1 AND u.center_id = $2 AND u.role = 'student' AND u.is_active = true`

	err := config.DBConnection.QueryRow(context.Background(), query, studentID, centerID).Scan(
		&student.ID, &student.Email, &student.FullName, &phoneNumber,
		&student.Role, &student.IsActive, &student.CreatedAt, &student.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("student not found")
		}
		return nil, fmt.Errorf("failed to query student: %w", err)
	}
	if phoneNumber != nil {
		student.PhoneNumber = *phoneNumber
	}

	return &student, nil
}

// EnrollStudent enrolls a student in a center
func (ss *StudentService) EnrollStudent(studentID, centerID string) (*models.StudentCenterEnrollment, error) {
	// Check if student exists
	var studentCount int
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE id = $1 AND role = 'student'`, studentID).Scan(&studentCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check student existence: %w", err)
	}
	if studentCount == 0 {
		return nil, errors.New("student not found")
	}

	// Check if already enrolled
	var existingCount int
	err = config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM student_center_enrollment WHERE student_id = $1 AND center_id = $2 AND is_active = true`,
		studentID, centerID).Scan(&existingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing enrollment: %w", err)
	}
	if existingCount > 0 {
		return nil, errors.New("student is already enrolled in this center")
	}

	var enrollment models.StudentCenterEnrollment
	query := `INSERT INTO student_center_enrollment (student_id, center_id, enrollment_date, is_active, created_at) 
			VALUES ($1, $2, $3, $4, NOW()) 
			RETURNING id, student_id, center_id, enrollment_date, is_active, created_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		studentID, centerID, time.Now(), true).Scan(
		&enrollment.ID, &enrollment.StudentID, &enrollment.CenterID,
		&enrollment.EnrollmentDate, &enrollment.IsActive, &enrollment.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to enroll student: %w", err)
	}

	return &enrollment, nil
}

// CreateStudent creates a new student and enrolls them in a center
func (ss *StudentService) CreateStudent(name, email, phone string, centerID string) (*models.User, error) {
	// Check if email already exists
	var existingCount int
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE email = $1`, email).Scan(&existingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}
	if existingCount > 0 {
		fmt.Printf("[CreateStudent] Email already exists: %s\n", email)
		return nil, errors.New("email already exists")
	}

	// Create user with student role
	var student models.User
	defaultPassword := "11111111"
	passwordHash, err := utils.HashPassword(defaultPassword)
	if err != nil {
		fmt.Printf("[CreateStudent] Failed to hash default password: %v\n", err)
		return nil, fmt.Errorf("failed to hash default password: %w", err)
	}
	query := `INSERT INTO users (email, password_hash, full_name, phone_number, role, center_id, is_active, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) 
			RETURNING id, email, full_name, phone_number, role, center_id, is_active, created_at, updated_at`

	fmt.Printf("[CreateStudent] Inserting student: email=%s, name=%s, phone=%s, centerID=%s\n", email, name, phone, centerID)
	// Print all educational_centers UUIDs for debugging
	rows, err := config.DBConnection.Query(context.Background(), "SELECT id, name FROM educational_centers")
	if err == nil {
		fmt.Printf("[CreateStudent] All educational_centers:\n")
		for rows.Next() {
			var eid, ename string
			_ = rows.Scan(&eid, &ename)
			fmt.Printf("  - id: %s, name: %s\n", eid, ename)
		}
		rows.Close()
	}
	// Use centerID (UUID string) directly for users.center_id
	err = config.DBConnection.QueryRow(context.Background(), query,
		email, passwordHash, name, phone, "student", centerID, true).Scan(
		&student.ID, &student.Email, &student.FullName, &student.PhoneNumber,
		&student.Role, &student.CenterID, &student.IsActive, &student.CreatedAt, &student.UpdatedAt)
	if err != nil {
		fmt.Printf("[CreateStudent] Failed to create student: %v\n", err)
		return nil, fmt.Errorf("failed to create student: %w", err)
	}

	return &student, nil
}

// UpdateStudent updates a student's details
func (ss *StudentService) UpdateStudent(studentID, name, email, phone, centerID string) (*models.User, error) {
	query := `UPDATE users 
		SET full_name = $1, email = $2, phone_number = $3, updated_at = NOW() 
		WHERE id = $4 AND center_id = $5 AND role = 'student' AND is_active = true 
		RETURNING id, email, full_name, phone_number, role, center_id, is_active, created_at, updated_at`

	var student models.User
	var phoneNumber *string
	err := config.DBConnection.QueryRow(context.Background(), query,
		name, email, phone, studentID, centerID).Scan(
		&student.ID, &student.Email, &student.FullName, &phoneNumber,
		&student.Role, &student.CenterID, &student.IsActive, &student.CreatedAt, &student.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("student not found")
		}
		return nil, fmt.Errorf("failed to update student: %w", err)
	}
	if phoneNumber != nil {
		student.PhoneNumber = *phoneNumber
	}

	return &student, nil
}

// DeleteStudent permanently deletes a student from the database
func (ss *StudentService) DeleteStudent(studentID, centerID string) error {
	query := `DELETE FROM users 
		WHERE id = $1 AND center_id = $2 AND role = 'student' AND is_active = true`

	result, err := config.DBConnection.Exec(context.Background(), query, studentID, centerID)
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("student not found")
	}

	return nil
}
