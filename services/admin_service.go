package services

import (
	"context"
	"errors"
	"fmt"

	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/utils"
)

// AdminService handles center admin-related business logic
type AdminService struct {
	config *config.Config
}

// NewAdminService creates a new AdminService instance
func NewAdminService(cfg *config.Config) *AdminService {
	return &AdminService{
		config: cfg,
	}
}

// GetAllCenterAdmins gets all center admins from the database
func (as *AdminService) GetAllCenterAdmins() ([]*models.User, error) {
	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT id, email, full_name, role, center_id, is_active, created_at, updated_at 
		 FROM users WHERE role = 'center_admin' ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query center admins: %w", err)
	}
	defer rows.Close()

	var admins []*models.User
	for rows.Next() {
		var admin models.User
		err := rows.Scan(&admin.ID, &admin.Email, &admin.FullName, &admin.Role, &admin.CenterID,
			&admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin row: %w", err)
		}
		admins = append(admins, &admin)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating admin rows: %w", err)
	}

	return admins, nil
}

// GetCenterAdminByID gets a center admin by ID from the database
func (as *AdminService) GetCenterAdminByID(adminID string) (*models.User, error) {
	if adminID == "" {
		return nil, errors.New("admin ID is required")
	}

	var admin models.User
	query := `SELECT id, email, full_name, role, is_active, created_at, updated_at 
			FROM users WHERE id = $1 AND role = 'center_admin'`

	err := config.DBConnection.QueryRow(context.Background(), query, adminID).Scan(
		&admin.ID, &admin.Email, &admin.FullName, &admin.Role,
		&admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("center admin not found")
		}
		return nil, fmt.Errorf("failed to query admin: %w", err)
	}

	return &admin, nil
}

// CreateCenterAdmin creates a new center admin in the database
func (as *AdminService) CreateCenterAdmin(req *models.CreateUserRequest) (*models.User, error) {
	fmt.Printf("Creating center admin with email: %s, name: %s\n", req.Email, req.FullName)
	fmt.Printf("Password length: %d\n", len(req.Password))
	fmt.Printf("Password characters: %s\n", req.Password)

	// Validate password strength
	if !utils.ValidatePasswordStrength(req.Password) {
		fmt.Printf("Password validation failed for: %s\n", req.Password)
		return nil, errors.New("password does not meet strength requirements: must contain uppercase, lowercase, number, and special character")
	}

	// Validate email format
	if !utils.ValidateEmail(req.Email) {
		fmt.Printf("Email validation failed for: %s\n", req.Email)
		return nil, errors.New("invalid email format")
	}

	// Check if email already exists
	fmt.Printf("Checking if email %s already exists\n", req.Email)
	var existingCount int
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE email = $1`, req.Email).Scan(&existingCount)
	if err != nil {
		fmt.Printf("Database error checking existing user: %v\n", err)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	fmt.Printf("Email %s count in database: %d\n", req.Email, existingCount)
	if existingCount > 0 {
		return nil, errors.New("user with this email already exists")
	}

	// Hash the password
	fmt.Printf("Hashing password for user %s\n", req.Email)
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		fmt.Printf("Password hashing error: %v\n", err)
		return nil, errors.New("failed to hash password")
	}
	fmt.Printf("Password hashed successfully\n")

	var admin models.User
	query := `INSERT INTO users (email, password_hash, full_name, role, center_id, is_active, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) 
			RETURNING id, email, full_name, role, center_id, is_active, created_at, updated_at`

	fmt.Printf("Executing query with email: %s, name: %s, role: %s, center_id: %v\n", req.Email, req.FullName, req.Role, req.CenterID)
	fmt.Printf("Query: %s\n", query)
	err = config.DBConnection.QueryRow(context.Background(), query,
		req.Email, passwordHash, req.FullName, req.Role, req.CenterID, true).Scan(
		&admin.ID, &admin.Email, &admin.FullName, &admin.Role, &admin.CenterID,
		&admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt)

	if err != nil {
		fmt.Printf("Database insertion error: %v\n", err)
		return nil, fmt.Errorf("failed to create center admin: %w", err)
	}
	fmt.Printf("Database insertion successful, admin ID: %s\n", admin.ID)

	fmt.Printf("Successfully created admin with ID: %s\n", admin.ID)
	return &admin, nil
}

// UpdateCenterAdmin updates a center admin in the database
func (as *AdminService) UpdateCenterAdmin(adminID, fullName, email string) (*models.User, error) {
	if adminID == "" {
		return nil, errors.New("admin ID is required")
	}

	// Check if admin exists
	var existingCount int
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE id = $1 AND role = 'center_admin'`, adminID).Scan(&existingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin existence: %w", err)
	}
	if existingCount == 0 {
		return nil, errors.New("center admin not found")
	}

	// Check if email is already taken by another user
	var emailCount int
	err = config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE email = $1 AND id != $2`, email, adminID).Scan(&emailCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if emailCount > 0 {
		return nil, errors.New("email is already taken by another user")
	}

	var admin models.User
	query := `UPDATE users SET full_name = $1, email = $2, updated_at = NOW() 
			WHERE id = $3 AND role = 'center_admin' 
			RETURNING id, email, full_name, role, is_active, created_at, updated_at`

	err = config.DBConnection.QueryRow(context.Background(), query, fullName, email, adminID).Scan(
		&admin.ID, &admin.Email, &admin.FullName, &admin.Role,
		&admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update center admin: %w", err)
	}

	return &admin, nil
}

// DeleteCenterAdmin deletes a center admin from the database
func (as *AdminService) DeleteCenterAdmin(adminID string) error {
	if adminID == "" {
		return errors.New("admin ID is required")
	}

	// First check if admin exists
	var existingCount int
	err := config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM users WHERE id = $1 AND role = 'center_admin'`, adminID).Scan(&existingCount)
	if err != nil {
		return fmt.Errorf("failed to check admin existence: %w", err)
	}
	if existingCount == 0 {
		return errors.New("center admin not found")
	}

	// Delete the admin
	_, err = config.DBConnection.Exec(context.Background(),
		`DELETE FROM users WHERE id = $1 AND role = 'center_admin'`, adminID)
	if err != nil {
		return fmt.Errorf("failed to delete center admin: %w", err)
	}

	return nil
}
