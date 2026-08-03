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

// AuthController handles authentication-related requests
type AuthController struct {
	config      *config.Config
	authService *services.AuthService
}

// NewAuthController creates a new AuthController instance
func NewAuthController(cfg *config.Config) *AuthController {
	return &AuthController{
		config:      cfg,
		authService: services.NewAuthService(cfg),
	}
}

// Login authenticates a user with email and password
// @Summary User Login
// @Description Authenticate user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login Request"
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Router /auth/login [post]
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req models.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate email format
	if !utils.ValidateEmail(req.Email) {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid email format", http.StatusBadRequest))
	}

	// Authenticate user
	user, token, err := ac.authService.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(config.ErrorResponse(err.Error(), http.StatusUnauthorized))
	}

	response := models.LoginResponse{
		User:      *user,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(response, "Login successful"))
}

// Register registers a new user
// @Summary User Registration
// @Description Register a new user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "Register Request"
// @Success 201 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 409 {object} config.APIResponse
// @Router /auth/register [post]
func (ac *AuthController) Register(c *fiber.Ctx) error {
	var req models.CreateUserRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate email format
	if !utils.ValidateEmail(req.Email) {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid email format", http.StatusBadRequest))
	}

	// Validate password strength
	if !utils.ValidatePasswordStrength(req.Password) {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Password does not meet strength requirements", http.StatusBadRequest))
	}

	// Validate role
	if !utils.IsValidRole(req.Role) {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid role", http.StatusBadRequest))
	}

	// Register user
	user, token, err := ac.authService.Register(&req)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(config.ErrorResponse(err.Error(), http.StatusConflict))
	}

	response := models.LoginResponse{
		User:      *user,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	return c.Status(http.StatusCreated).JSON(config.SuccessResponse(response, "User registered successfully"))
}

// RefreshToken refreshes the JWT token
// @Summary Refresh Token
// @Description Refresh the JWT token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body struct{RefreshToken string `json:"refresh_token"`} true "Refresh Token Request"
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Router /auth/refresh-token [post]
func (ac *AuthController) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	newToken, err := ac.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(config.ErrorResponse("Invalid or expired refresh token", http.StatusUnauthorized))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(fiber.Map{
		"token": newToken,
	}, "Token refreshed successfully"))
}

// Logout logs out the user
// @Summary Logout
// @Description Logout the user and invalidate the token
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Router /auth/logout [post]
func (ac *AuthController) Logout(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(http.StatusUnauthorized).JSON(config.ErrorResponse("User not authenticated", http.StatusUnauthorized))
	}

	// Call the auth service to handle logout
	if err := ac.authService.Logout(userID.(string)); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to logout", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Logged out successfully"))
}