package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/utils"
)

// StudentAuthService handles student authentication business logic.
// This is completely separate from the admin auth service.
type StudentAuthService struct {
	config *config.Config
}

// NewStudentAuthService creates a new StudentAuthService instance.
func NewStudentAuthService(cfg *config.Config) *StudentAuthService {
	return &StudentAuthService{config: cfg}
}

// Register creates a new student account with validated data.
func (s *StudentAuthService) Register(req *models.StudentRegisterRequest) (*models.User, string, error) {
	// Sanitize inputs
	email := strings.ToLower(strings.TrimSpace(req.Email))
	fullName := strings.TrimSpace(req.FullName)
	phone := strings.TrimSpace(req.Phone)

	// --- Validation ---
	if fullName == "" {
		return nil, "", fmt.Errorf("full_name is required")
	}
	if email == "" || !utils.ValidateEmail(email) {
		return nil, "", fmt.Errorf("valid email is required")
	}
	if phone == "" {
		return nil, "", fmt.Errorf("phone number is required")
	}
	if req.Password == "" || len(req.Password) < 8 {
		return nil, "", fmt.Errorf("password must be at least 8 characters")
	}

	// Check email uniqueness
	var emailCount int
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE email = $1`, email).Scan(&emailCount)
	if err != nil {
		return nil, "", fmt.Errorf("database error checking email: %w", err)
	}
	if emailCount > 0 {
		return nil, "", fmt.Errorf("email already exists")
	}

	// Check phone uniqueness
	var phoneCount int
	err = config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE phone_number = $1`, phone).Scan(&phoneCount)
	if err != nil {
		return nil, "", fmt.Errorf("database error checking phone: %w", err)
	}
	if phoneCount > 0 {
		return nil, "", fmt.Errorf("phone number already registered")
	}

	// Hash password
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to secure password: %w", err)
	}

	// Insert into users table as a student with status='pending'
	var newUser models.User
	query := `INSERT INTO users (email, password_hash, full_name, phone_number, role, status, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'student', 'pending', true, NOW(), NOW())
		RETURNING id, email, full_name, phone_number, role, center_id, status, is_active, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		email, passwordHash, fullName, phone,
	).Scan(
		&newUser.ID, &newUser.Email, &newUser.FullName, &newUser.PhoneNumber,
		&newUser.Role, &newUser.CenterID, &newUser.Status, &newUser.IsActive, &newUser.CreatedAt, &newUser.UpdatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create student account: %w", err)
	}

	// Generate JWT token (center_id is empty — student hasn't been activated yet)
	token, err := utils.GenerateJWT(newUser.ID, newUser.Role, "", s.config.JWTSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &newUser, token, nil
}

// Login authenticates a student and returns a JWT token.
func (s *StudentAuthService) Login(email, password string) (*models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || !utils.ValidateEmail(email) {
		return nil, "", fmt.Errorf("valid email is required")
	}
	if password == "" {
		return nil, "", fmt.Errorf("password is required")
	}

	// Look up user by email — must be a student
	var user models.User
	var avatar *string
	query := `SELECT id, email, password_hash, full_name, phone_number, avatar,
		role, center_id, status, is_active, created_at, updated_at
		FROM users WHERE email = $1 AND role = 'student' AND is_active = true`

	err := config.DBConnection.QueryRow(context.Background(), query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.PhoneNumber,
		&avatar, &user.Role, &user.CenterID, &user.Status, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	user.Avatar = avatar

	if err != nil {
		// Don't leak whether the email exists — generic message
		return nil, "", fmt.Errorf("invalid email or password")
	}

	// Verify password
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid email or password")
	}

	// Generate JWT with center_id (will be empty for pending/unactivated students)
	centerIDStr := ""
	if user.CenterID != nil {
		centerIDStr = *user.CenterID
	}
	token, err := utils.GenerateJWT(user.ID, user.Role, centerIDStr, s.config.JWTSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &user, token, nil
}

// GetProfile returns a student's profile from the database.
func (s *StudentAuthService) GetProfile(userID string) (*models.User, error) {
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

// UpdateProfile updates a student's profile fields (full_name, phone, avatar).
// Email cannot be changed through this endpoint.
func (s *StudentAuthService) UpdateProfile(userID string, req *models.UpdateStudentProfileRequest) (*models.User, error) {
	// Build dynamic UPDATE query — only set fields that are provided
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.FullName != nil {
		name := strings.TrimSpace(*req.FullName)
		if len(name) < 2 {
			return nil, fmt.Errorf("full_name must be at least 2 characters")
		}
		setClauses = append(setClauses, fmt.Sprintf("full_name = $%d", argIdx))
		args = append(args, name)
		argIdx++
	}

	if req.Phone != nil {
		phone := strings.TrimSpace(*req.Phone)
		if phone == "" {
			return nil, fmt.Errorf("phone cannot be empty")
		}
		// Check uniqueness (exclude current user)
		var phoneCount int
		err := config.DBConnection.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM users WHERE phone_number = $1 AND id != $2`,
			phone, userID).Scan(&phoneCount)
		if err != nil {
			return nil, fmt.Errorf("database error checking phone: %w", err)
		}
		if phoneCount > 0 {
			return nil, fmt.Errorf("phone number already in use")
		}
		setClauses = append(setClauses, fmt.Sprintf("phone_number = $%d", argIdx))
		args = append(args, phone)
		argIdx++
	}

	if req.Avatar != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar = $%d", argIdx))
		args = append(args, *req.Avatar)
		argIdx++
	}

	if len(setClauses) == 0 {
		// Nothing to update — return current profile
		return s.GetProfile(userID)
	}

	// Always touch updated_at
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	// Add userID as the last argument
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
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return &user, nil
}
