package services

import (
	"errors"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"
)

// StudentProfileService handles student profile-related business logic
type StudentProfileService struct {
	config *config.Config
}

// NewStudentProfileService creates a new StudentProfileService instance
func NewStudentProfileService(cfg *config.Config) *StudentProfileService {
	return &StudentProfileService{
		config: cfg,
	}
}

// GetProfile gets the student's profile
func (sps *StudentProfileService) GetProfile(userID string) (*models.User, error) {
	// In a real implementation, we would query the database
	// For now, we'll return a mock response

	// Check if user exists (in a real implementation, this would be a DB query)
	if userID == "" {
		return nil, errors.New("user not found")
	}

	profile := &models.User{
		ID:          userID,
		PhoneNumber: "93700111111",
		FullName:    "Ahmad Rahimi",
		Email:       "ahmad@example.com",
		Role:        "student",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return profile, nil
}

// UpdateProfile updates the student's profile
func (sps *StudentProfileService) UpdateProfile(userID, fullName, email string) (*models.User, error) {
	// In a real implementation, we would update in the database
	// For now, we'll return a mock response

	// Check if user exists (in a real implementation, this would be a DB query)
	if userID == "" {
		return nil, errors.New("user not found")
	}

	profile := &models.User{
		ID:          userID,
		PhoneNumber: "93700111111", // This wouldn't change
		FullName:    fullName,
		Email:       email,
		Role:        "student",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return profile, nil
}