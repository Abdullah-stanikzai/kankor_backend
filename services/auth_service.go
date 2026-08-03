package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/utils"
)

// AuthService handles authentication-related business logic
type AuthService struct {
	config *config.Config
}

// NewAuthService creates a new AuthService instance
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		config: cfg,
	}
}

// Register registers a new user in the database and returns the user with a JWT token
func (as *AuthService) Register(req *models.CreateUserRequest) (*models.User, string, error) {
	// Check if user with email already exists
	var existingCount int
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE email = $1`,
		req.Email).Scan(&existingCount)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingCount > 0 {
		// User already exists
		return nil, "", fmt.Errorf("user with email %s already exists", req.Email)
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	var newUser models.User
	query := `INSERT INTO users (email, password_hash, full_name, role, is_active, created_at, updated_at) 
	VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
	RETURNING id, email, full_name, role, is_active, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query,
		req.Email, passwordHash, req.FullName, req.Role, true,
	).Scan(
		&newUser.ID, &newUser.Email, &newUser.FullName, &newUser.Role,
		&newUser.IsActive, &newUser.CreatedAt, &newUser.UpdatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT token for the newly registered user
	centerID := ""
	if newUser.Role == "center_admin" && newUser.CenterID != nil {
		centerID = *newUser.CenterID
	}
	token, err := utils.GenerateJWT(newUser.ID, newUser.Role, centerID, as.config.JWTSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &newUser, token, nil
}

// Login authenticates a user with email and password against the database
func (as *AuthService) Login(email, password string) (*models.User, string, error) {
	// Look up the user by email from the database
	var user models.User
	var centerID *string

	// Try to query with center_id column first (new schema)
	query := `SELECT id, email, password_hash, full_name, role, center_id, is_active, created_at, updated_at 
			FROM users WHERE email = $1 AND is_active = true`

	err := config.DBConnection.QueryRow(context.Background(), query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Role,
		&centerID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)

	// If column doesn't exist, try without center_id (old schema)
	if err != nil && err.Error() == "no rows in result set" {
		return nil, "", fmt.Errorf("invalid email or password")
	} else if err != nil {
		fmt.Printf("Error with new schema query: %v, trying old schema\n", err)
		// Try old schema without center_id
		queryOld := `SELECT id, email, password_hash, full_name, role, is_active, created_at, updated_at 
				FROM users WHERE email = $1 AND is_active = true`

		err = config.DBConnection.QueryRow(context.Background(), queryOld, email).Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Role,
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			if err.Error() == "no rows in result set" {
				return nil, "", fmt.Errorf("invalid email or password")
			}
			return nil, "", fmt.Errorf("failed to query database: %w", err)
		}

		// If using old schema, try to get center_id from educational_centers table
		if user.Role == "center_admin" {
			var fetchedCenterID *string
			centerQuery := `SELECT id FROM educational_centers WHERE admin_user_id = $1 LIMIT 1`
			_ = config.DBConnection.QueryRow(context.Background(), centerQuery, user.ID).Scan(&fetchedCenterID)
			centerID = fetchedCenterID
		}
	}

	// Verify password hash
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid email or password")
	}

	// Set center_id in user object
	user.CenterID = centerID

	// Generate JWT token with center_id
	centerIDStr := ""
	if centerID != nil {
		centerIDStr = *centerID
	}
	fmt.Printf("Generated JWT token for user %s with center_id: %s\n", user.ID, centerIDStr)
	token, err := utils.GenerateJWT(user.ID, user.Role, centerIDStr, as.config.JWTSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &user, token, nil
}

// RefreshToken refreshes the JWT token
func (as *AuthService) RefreshToken(refreshToken string) (string, error) {
	// In a real implementation, we would:
	// 1. Validate the refresh token against the database
	// 2. Check if it hasn't expired
	// 3. Check if it hasn't been revoked
	// 4. Generate a new JWT token

	// For now, we'll just return an error to indicate this needs implementation
	return "", errors.New("refresh token functionality not implemented yet")
}

// Logout invalidates the user's session by revoking tokens
func (as *AuthService) Logout(userID string) error {
	// In a real implementation, we would:
	// 1. Add the token to a blacklist/revoked tokens table
	// 2. Or invalidate all refresh tokens for this user
	// 3. Clear any active sessions

	// For now, we'll just log the logout action
	log.Printf("User %s logged out", userID)
	return nil
}

// AuthenticateUser authenticates a user by phone number and password
// This would be used for login without OTP
func (as *AuthService) AuthenticateUser(phoneNumber, password string) (*models.User, string, error) {
	// In a real implementation, we would:
	// 1. Look up the user by phone number
	// 2. Compare the password hash with the stored hash
	// 3. Generate a JWT token if authentication succeeds

	// For now, we'll simulate authentication
	user := &models.User{
		ID:          "test-user-id",
		PhoneNumber: phoneNumber,
		FullName:    "Test User",
		Role:        "student",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Role, "", as.config.JWTSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}
