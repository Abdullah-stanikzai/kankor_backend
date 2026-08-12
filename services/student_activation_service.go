package services

import (
	"context"
	"fmt"
	"strings"

	"kankor-backend/config"
	"kankor-backend/models"
)

// StudentActivationService handles student activation/deactivation by center admins.
type StudentActivationService struct {
	config *config.Config
}

// NewStudentActivationService creates a new StudentActivationService instance.
func NewStudentActivationService(cfg *config.Config) *StudentActivationService {
	return &StudentActivationService{config: cfg}
}

// SearchStudent searches for a student by email (required) and optionally by phone.
// Security: only returns the student if center_id IS NULL (never activated)
// OR center_id == adminCenterID (belongs to this admin's center).
func (s *StudentActivationService) SearchStudent(email, phone, adminCenterID string) (*models.StudentSearchResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Build query dynamically based on provided filters
	query := `SELECT id, full_name, email, phone_number, center_id, status, is_active, created_at::text
		FROM users
		WHERE role = 'student' AND email = $1`
	args := []interface{}{email}
	argIdx := 2

	if phone = strings.TrimSpace(phone); phone != "" {
		query += fmt.Sprintf(" AND phone_number = $%d", argIdx)
		args = append(args, phone)
		argIdx++
	}

	fmt.Printf("[SearchStudent] Querying: email=%s phone=%s adminCenterID=%s\n", email, phone, adminCenterID)

	var result models.StudentSearchResult
	var phoneOut *string
	var centerID *string

	err := config.DBConnection.QueryRow(context.Background(), query, args...).Scan(
		&result.ID, &result.FullName, &result.Email, &phoneOut,
		&centerID, &result.Status, &result.IsActive, &result.CreatedAt,
	)
	if err != nil {
		fmt.Printf("[SearchStudent] DB error: %v\n", err)
		// pgx returns "no rows in result set" when no match
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("no student found with that email")
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	if phoneOut != nil {
		result.Phone = *phoneOut
	}
	result.CenterID = centerID

	fmt.Printf("[SearchStudent] Found student: id=%s center_id=%v status=%s\n", result.ID, centerID, result.Status)

	// Security check: only show if unassigned OR belongs to this admin's center
	if centerID == nil {
		// Student has never been activated — show to any center admin
		result.BelongsToUs = false
		fmt.Printf("[SearchStudent] Student is unassigned (center_id=NULL) — showing\n")
	} else if *centerID == adminCenterID {
		// Student belongs to this admin's center
		result.BelongsToUs = true
		fmt.Printf("[SearchStudent] Student belongs to this admin's center — showing\n")
	} else {
		// Student belongs to another center — do not expose
		fmt.Printf("[SearchStudent] Student belongs to another center (%s != %s) — hiding\n", *centerID, adminCenterID)
		return nil, fmt.Errorf("student not found")
	}

	return &result, nil
}

// ActivateStudent assigns a student to the admin's center and sets status to "active".
// Security: only works if student's center_id IS NULL (never been activated).
func (s *StudentActivationService) ActivateStudent(studentID, adminCenterID string) (*models.User, error) {
	// Verify student exists, is a student, and center_id IS NULL
	var currentCenterID *string
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT center_id FROM users WHERE id = $1 AND role = 'student' AND is_active = true`,
		studentID).Scan(&currentCenterID)
	if err != nil {
		return nil, fmt.Errorf("student not found")
	}

	if currentCenterID != nil {
		if *currentCenterID == adminCenterID {
			return nil, fmt.Errorf("student is already assigned to your center")
		}
		return nil, fmt.Errorf("student belongs to another center — cannot activate")
	}

	// Activate: assign to center and set status to active
	var user models.User
	var avatar *string
	query := `UPDATE users
		SET center_id = $1, status = 'active', updated_at = NOW()
		WHERE id = $2 AND role = 'student' AND is_active = true
		RETURNING id, email, full_name, phone_number, avatar, role, center_id, status, is_active, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query, adminCenterID, studentID).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhoneNumber, &avatar,
		&user.Role, &user.CenterID, &user.Status, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	user.Avatar = avatar

	if err != nil {
		return nil, fmt.Errorf("failed to activate student: %w", err)
	}

	return &user, nil
}

// DeactivateStudent sets a student's status to "inactive" while keeping center_id.
// Security: only works if student's center_id == adminCenterID.
func (s *StudentActivationService) DeactivateStudent(studentID, adminCenterID string) (*models.User, error) {
	var user models.User
	var avatar *string
	query := `UPDATE users
		SET status = 'inactive', updated_at = NOW()
		WHERE id = $1 AND role = 'student' AND center_id = $2 AND is_active = true
		RETURNING id, email, full_name, phone_number, avatar, role, center_id, status, is_active, created_at, updated_at`

	err := config.DBConnection.QueryRow(context.Background(), query, studentID, adminCenterID).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhoneNumber, &avatar,
		&user.Role, &user.CenterID, &user.Status, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	user.Avatar = avatar

	if err != nil {
		return nil, fmt.Errorf("student not found or does not belong to your center")
	}

	return &user, nil
}
