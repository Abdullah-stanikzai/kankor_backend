package services

import (
	"context"
	"errors"
	"fmt"

	"kankor-backend/config"
	"kankor-backend/models"
)

// CenterService handles educational center-related business logic
type CenterService struct {
	config *config.Config
}

// NewCenterService creates a new CenterService instance
func NewCenterService(cfg *config.Config) *CenterService {
	return &CenterService{
		config: cfg,
	}
}

// GetAllCenters gets all educational centers with pagination
func (cs *CenterService) GetAllCenters(offset, limit int) ([]*models.EducationalCenter, int, error) {
	fmt.Printf("DEBUG: GetAllCenters called with offset=%d, limit=%d\n", offset, limit)

	rows, err := config.DBConnection.Query(context.Background(),
		`SELECT ec.id, ec.name, ec.description, ec.admin_user_id, ec.is_active, ec.created_at, ec.updated_at, u.full_name 
		 FROM educational_centers ec 
		 LEFT JOIN users u ON ec.admin_user_id = u.id 
		 WHERE ec.is_active = true 
		 ORDER BY ec.created_at DESC 
		 LIMIT $1 OFFSET $2`,
		limit, offset)

	if err != nil {
		fmt.Printf("DEBUG: Query failed: %v\n", err)
		return nil, 0, fmt.Errorf("failed to query centers: %w", err)
	}
	defer rows.Close()

	fmt.Printf("DEBUG: Query executed successfully\n")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query centers: %w", err)
	}
	defer rows.Close()

	var centers []*models.EducationalCenter
	for rows.Next() {
		var center models.EducationalCenter
		var adminFullName *string
		err := rows.Scan(
			&center.ID, &center.Name, &center.Description, &center.AdminUserID,
			&center.IsActive, &center.CreatedAt, &center.UpdatedAt, &adminFullName,
		)
		if err != nil {
			fmt.Printf("DEBUG: Row scan failed: %v\n", err)
			return nil, 0, fmt.Errorf("failed to scan center: %w", err)
		}
		center.AdminFullName = adminFullName

		// Debug logging for each center
		adminIDStr := "NULL"
		if center.AdminUserID != nil {
			adminIDStr = *center.AdminUserID
		}
		adminNameStr := "NULL"
		if adminFullName != nil {
			adminNameStr = *adminFullName
		}
		fmt.Printf("DEBUG: Center - ID: %s, Name: %s, AdminID: %s, AdminName: %s\n",
			center.ID, center.Name, adminIDStr, adminNameStr)

		centers = append(centers, &center)
	}

	// Get total count
	var totalCount int
	err = config.DBConnection.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM educational_centers WHERE is_active = true`).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count centers: %w", err)
	}

	return centers, totalCount, nil
}

// GetCenterByID gets an educational center by ID
func (cs *CenterService) GetCenterByID(centerID string) (*models.EducationalCenter, error) {
	var center models.EducationalCenter
	var adminFullName *string
	query := `SELECT ec.id, ec.name, ec.description, ec.admin_user_id, ec.is_active, ec.created_at, ec.updated_at, u.full_name 
		FROM educational_centers ec 
		LEFT JOIN users u ON ec.admin_user_id = u.id 
		WHERE ec.id = $1`

	err := config.DBConnection.QueryRow(context.Background(), query, centerID).Scan(
		&center.ID, &center.Name, &center.Description, &center.AdminUserID,
		&center.IsActive, &center.CreatedAt, &center.UpdatedAt, &adminFullName,
	)
	if err != nil {
		return nil, fmt.Errorf("center not found: %w", err)
	}
	center.AdminFullName = adminFullName

	return &center, nil
}

// CreateCenter creates a new educational center
func (cs *CenterService) CreateCenter(req *models.CreateEducationalCenterRequest) (*models.EducationalCenter, error) {
	var newCenter models.EducationalCenter

	// Make admin_user_id nullable to handle cases where it's not provided
	var adminUserID *string
	if req.AdminUserID != "" {
		adminUserID = &req.AdminUserID
	}

	query := `INSERT INTO educational_centers (name, description, admin_user_id, is_active, created_at, updated_at) 
VALUES ($1, $2, $3, $4, NOW(), NOW()) 
RETURNING id, name, description, admin_user_id, is_active, created_at, updated_at`

	err := config.DBConnection.QueryRow(context.Background(), query,
		req.Name, req.Description, adminUserID, true, // is_active = true
	).Scan(
		&newCenter.ID, &newCenter.Name, &newCenter.Description, &newCenter.AdminUserID,
		&newCenter.IsActive, &newCenter.CreatedAt, &newCenter.UpdatedAt,
	)
	if err != nil {
		fmt.Printf("Database error in CreateCenter: %v\n", err)
		return nil, fmt.Errorf("failed to create center: %w", err)
	}

	return &newCenter, nil
}

// UpdateCenter updates an educational center
func (cs *CenterService) UpdateCenter(centerID string, req *models.CreateEducationalCenterRequest) (*models.EducationalCenter, error) {
	fmt.Printf("DEBUG: UpdateCenter called - CenterID: %s, Name: %s, AdminUserID: %s, Description: %s\n",
		centerID, req.Name, req.AdminUserID, req.Description)

	var center models.EducationalCenter
	query := `UPDATE educational_centers SET name=$1, description=$2, admin_user_id=$3, updated_at=NOW() WHERE id=$4 RETURNING id, name, description, admin_user_id, is_active, created_at, updated_at`

	err := config.DBConnection.QueryRow(context.Background(), query,
		req.Name, req.Description, req.AdminUserID, centerID,
	).Scan(
		&center.ID, &center.Name, &center.Description, &center.AdminUserID,
		&center.IsActive, &center.CreatedAt, &center.UpdatedAt,
	)
	if err != nil {
		fmt.Printf("DEBUG: UpdateCenter failed: %v\n", err)
		return nil, fmt.Errorf("failed to update center: %w", err)
	}

	fmt.Printf("DEBUG: UpdateCenter successful - Updated Center ID: %s, AdminUserID: %v\n",
		center.ID, center.AdminUserID)

	// Verify the admin_user_id was set correctly
	if center.AdminUserID != nil && *center.AdminUserID == req.AdminUserID {
		fmt.Printf("DEBUG: AdminUserID verification successful - %s\n", *center.AdminUserID)
	} else if req.AdminUserID != "" {
		fmt.Printf("DEBUG: WARNING - AdminUserID not set correctly. Expected: %s, Got: %v\n",
			req.AdminUserID, center.AdminUserID)
	}

	return &center, nil
}

// DeleteCenter deletes an educational center
func (cs *CenterService) DeleteCenter(centerID string) error {
	query := `UPDATE educational_centers SET is_active=false, updated_at=NOW() WHERE id=$1`
	result, err := config.DBConnection.Exec(context.Background(), query, centerID)
	if err != nil {
		return fmt.Errorf("failed to delete center: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("center not found or already deleted")
	}
	return nil
}

// ActivateCenter activates an educational center
func (cs *CenterService) ActivateCenter(centerID string) (*models.EducationalCenter, error) {
	var center models.EducationalCenter
	query := `UPDATE educational_centers SET is_active=true, updated_at=NOW() WHERE id=$1 RETURNING id, name, description, admin_user_id, is_active, created_at, updated_at`

	err := config.DBConnection.QueryRow(context.Background(), query, centerID).Scan(
		&center.ID, &center.Name, &center.Description, &center.AdminUserID,
		&center.IsActive, &center.CreatedAt, &center.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to activate center: %w", err)
	}

	return &center, nil
}

// DeactivateCenter deactivates an educational center
func (cs *CenterService) DeactivateCenter(centerID string) (*models.EducationalCenter, error) {
	var center models.EducationalCenter
	query := `UPDATE educational_centers SET is_active=false, updated_at=NOW() WHERE id=$1 RETURNING id, name, description, admin_user_id, is_active, created_at, updated_at`

	err := config.DBConnection.QueryRow(context.Background(), query, centerID).Scan(
		&center.ID, &center.Name, &center.Description, &center.AdminUserID,
		&center.IsActive, &center.CreatedAt, &center.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate center: %w", err)
	}

	return &center, nil
}
