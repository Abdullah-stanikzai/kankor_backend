package controllers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"kankor-backend/config"
	"kankor-backend/services"
)

// SystemSettingsController handles system settings-related requests
type SystemSettingsController struct {
	config         *config.Config
	settingsService *services.SystemSettingsService
}

// NewSystemSettingsController creates a new SystemSettingsController instance
func NewSystemSettingsController(cfg *config.Config) *SystemSettingsController {
	return &SystemSettingsController{
		config:         cfg,
		settingsService: services.NewSystemSettingsService(cfg),
	}
}

// GetSystemSettings gets all system settings
// @Summary Get system settings
// @Description Get all system settings
// @Tags System Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/settings [get]
func (ssc *SystemSettingsController) GetSystemSettings(c *fiber.Ctx) error {
	settings, err := ssc.settingsService.GetSystemSettings()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get system settings", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(settings, "System settings retrieved successfully"))
}

// UpdateSystemSettings updates system settings
// @Summary Update system settings
// @Description Update system settings
// @Tags System Settings
// @Accept json
// @Produce json
// @Param request body map[string]string true "Update System Settings Request"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/settings [put]
func (ssc *SystemSettingsController) UpdateSystemSettings(c *fiber.Ctx) error {
	var req map[string]string
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	settings, err := ssc.settingsService.UpdateSystemSettings(req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to update system settings", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(settings, "System settings updated successfully"))
}