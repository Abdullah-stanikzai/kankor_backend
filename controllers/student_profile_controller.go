package controllers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"kankor-backend/config"
	"kankor-backend/services"
)

// StudentProfileController handles student profile-related requests
type StudentProfileController struct {
	config        *config.Config
	profileService *services.StudentProfileService
}

// NewStudentProfileController creates a new StudentProfileController instance
func NewStudentProfileController(cfg *config.Config) *StudentProfileController {
	return &StudentProfileController{
		config:        cfg,
		profileService: services.NewStudentProfileService(cfg),
	}
}

// GetProfile gets the student's profile
// @Summary Get profile
// @Description Get the authenticated student's profile
// @Tags Student Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /student/profile [get]
func (spc *StudentProfileController) GetProfile(c *fiber.Ctx) error {
	// Get user info from context
	userID := c.Locals("user_id").(string)

	profile, err := spc.profileService.GetProfile(userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get profile", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(profile, "Profile retrieved successfully"))
}

// UpdateProfile updates the student's profile
// @Summary Update profile
// @Description Update the authenticated student's profile
// @Tags Student Profile
// @Accept json
// @Produce json
// @Param request body struct{FullName string `json:"full_name"`;Email string `json:"email"`} true "Update Profile Request"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /student/profile [put]
func (spc *StudentProfileController) UpdateProfile(c *fiber.Ctx) error {
	var req struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Get user info from context
	userID := c.Locals("user_id").(string)

	// Validate inputs
	if req.FullName != "" && len(req.FullName) < 2 {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Full name must be at least 2 characters", http.StatusBadRequest))
	}

	if req.Email != "" {
		// Validate email format
		if !isValidEmail(req.Email) {
			return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid email format", http.StatusBadRequest))
		}
	}

	profile, err := spc.profileService.UpdateProfile(userID, req.FullName, req.Email)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to update profile", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(profile, "Profile updated successfully"))
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	// Simple email validation - in a real implementation, use a proper email validation library
	for i, char := range email {
		if char == '@' {
			return i > 0 && len(email) > i+1
		}
	}
	return false
}