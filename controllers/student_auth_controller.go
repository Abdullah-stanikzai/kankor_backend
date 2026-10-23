package controllers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"kankor-backend/config"
	"kankor-backend/middleware"
	"kankor-backend/models"
	"kankor-backend/services"
	"kankor-backend/utils"
)

// StudentAuthController handles student-specific authentication requests.
// This is separate from the admin AuthController.
type StudentAuthController struct {
	config         *config.Config
	studentAuthSvc *services.StudentAuthService
}

// NewStudentAuthController creates a new StudentAuthController instance.
func NewStudentAuthController(cfg *config.Config) *StudentAuthController {
	return &StudentAuthController{
		config:         cfg,
		studentAuthSvc: services.NewStudentAuthService(cfg),
	}
}

// Register handles student registration.
// POST /api/v1/auth/student/register
func (sc *StudentAuthController) Register(c *fiber.Ctx) error {
	var req models.StudentRegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("Invalid request body", http.StatusBadRequest),
		)
	}

	// Validate required fields
	if req.FullName == "" {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("full_name is required", http.StatusBadRequest),
		)
	}
	if !utils.ValidateEmail(req.Email) {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("Valid email is required", http.StatusBadRequest),
		)
	}
	if req.Phone == "" {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("phone is required", http.StatusBadRequest),
		)
	}
	if len(req.Password) < 8 {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("Password must be at least 8 characters", http.StatusBadRequest),
		)
	}

	user, token, err := sc.studentAuthSvc.Register(&req)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(
			config.ErrorResponse(err.Error(), http.StatusConflict),
		)
	}

	response := models.LoginResponse{
		User:         *user,
		Token:        token,
		RefreshToken: sc.createRefreshToken(user),
		ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(),
	}

	return c.Status(http.StatusCreated).JSON(
		config.SuccessResponse(response, "Registration successful"),
	)
}

// Login handles student login.
// POST /api/v1/auth/student/login
func (sc *StudentAuthController) Login(c *fiber.Ctx) error {
	var req models.StudentLoginRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("Invalid request body", http.StatusBadRequest),
		)
	}

	if !utils.ValidateEmail(req.Email) {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("Valid email is required", http.StatusBadRequest),
		)
	}

	user, token, err := sc.studentAuthSvc.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(
			config.ErrorResponse(err.Error(), http.StatusUnauthorized),
		)
	}

	response := models.LoginResponse{
		User:         *user,
		Token:        token,
		RefreshToken: sc.createRefreshToken(user),
		ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(),
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(response, "Login successful"),
	)
}

// RefreshToken exchanges a valid long-lived refresh token for a new access
// token without asking the student to enter credentials again.
func (sc *StudentAuthController) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("refresh_token is required", http.StatusBadRequest))
	}
	claims, err := middleware.ValidateRefreshJWT(req.RefreshToken, sc.config.JWTSecret)
	if err != nil || claims.Role != "student" {
		return c.Status(http.StatusUnauthorized).JSON(config.ErrorResponse("Invalid or expired refresh token", http.StatusUnauthorized))
	}
	user, err := sc.studentAuthSvc.GetProfile(claims.UserID)
	if err != nil || !user.IsActive {
		return c.Status(http.StatusUnauthorized).JSON(config.ErrorResponse("Student account is no longer active", http.StatusUnauthorized))
	}
	centerID := ""
	if user.CenterID != nil {
		centerID = *user.CenterID
	}
	token, err := utils.GenerateJWT(user.ID, user.Role, centerID, sc.config.JWTSecret)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to refresh session", http.StatusInternalServerError))
	}
	return c.Status(http.StatusOK).JSON(config.SuccessResponse(fiber.Map{
		"token":         token,
		"refresh_token": sc.createRefreshToken(user),
		"expires_at":    time.Now().Add(24 * time.Hour).Unix(),
	}, "Token refreshed successfully"))
}

func (sc *StudentAuthController) createRefreshToken(user *models.User) string {
	centerID := ""
	if user.CenterID != nil {
		centerID = *user.CenterID
	}
	token, err := middleware.GenerateRefreshJWT(user.ID, user.Role, centerID, sc.config.JWTSecret)
	if err != nil {
		return ""
	}
	return token
}

// GetMe returns the currently authenticated student's profile.
// GET /api/v1/auth/student/me
func (sc *StudentAuthController) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(http.StatusUnauthorized).JSON(
			config.ErrorResponse("User not authenticated", http.StatusUnauthorized),
		)
	}

	user, err := sc.studentAuthSvc.GetProfile(userID.(string))
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(
			config.ErrorResponse("Student not found", http.StatusNotFound),
		)
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(user, "Profile retrieved successfully"),
	)
}

// UpdateProfile updates the student's profile.
// PUT /api/v1/auth/student/profile
func (sc *StudentAuthController) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(http.StatusUnauthorized).JSON(
			config.ErrorResponse("User not authenticated", http.StatusUnauthorized),
		)
	}

	var req models.UpdateStudentProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("Invalid request body", http.StatusBadRequest),
		)
	}

	user, err := sc.studentAuthSvc.UpdateProfile(userID.(string), &req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse(err.Error(), http.StatusBadRequest),
		)
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(user, "Profile updated successfully"),
	)
}
