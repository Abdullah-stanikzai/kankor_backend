package controllers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"kankor-backend/config"
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
		User:      *user,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
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
		User:      *user,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(response, "Login successful"),
	)
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
