package services

import (
	"context"
	"fmt"
	"strings"

	"kankor-backend/config"
	"kankor-backend/models"
)

// StudentProfileService handles student profile-related business logic.
type StudentProfileService struct {
	config *config.Config
}

// NewStudentProfileService creates a new StudentProfileService instance.
func NewStudentProfileService(cfg *config.Config) *StudentProfileService {
	return &StudentProfileService{config: cfg}
}

// GetProfile returns a student's profile from the database.
func (sps *StudentProfileService) GetProfile(userID string) (*models.User, error) {
	var user models.User
	var avatar *string
	query := `SELECT id, email, full_name, phone_number, avatar, role,
		center_id, status, is_active, created_at, updated_at
		FROM users WHERE id = $1 AND role = 'student'`

	err := config.DBConnection.QueryRow(context.Background(), query, userID).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhoneNumber, &avatar,
		&user.Role, &user.CenterID, &user.Status, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	user.Avatar = avatar

	if err != nil {
		return nil, fmt.Errorf("student not found")
	}
	return &user, nil
}

// UpdateProfile updates the student's full_name and/or email.
func (sps *StudentProfileService) UpdateProfile(userID, fullName, email string) (*models.User, error) {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if strings.TrimSpace(fullName) != "" {
		setClauses = append(setClauses, fmt.Sprintf("full_name = $%d", argIdx))
		args = append(args, strings.TrimSpace(fullName))
		argIdx++
	}

	if strings.TrimSpace(email) != "" {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, strings.TrimSpace(email))
		argIdx++
	}

	if len(setClauses) == 0 {
		return sps.GetProfile(userID)
	}

	args = append(args, userID)

	query := fmt.Sprintf(
		`UPDATE users SET %s WHERE id = $%d AND role = 'student'
		RETURNING id, email, full_name, phone_number, avatar, role, center_id, status, is_active, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx,
	)

	var user models.User
	var avatar *string
	err := config.DBConnection.QueryRow(context.Background(), query, args...).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhoneNumber, &avatar,
		&user.Role, &user.CenterID, &user.Status, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	user.Avatar = avatar

	if err != nil {
		return nil, fmt.Errorf("failed to update profile")
	}

	return &user, nil
}
