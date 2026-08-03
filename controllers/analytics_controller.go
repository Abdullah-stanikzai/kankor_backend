package controllers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"kankor-backend/config"
	"kankor-backend/services"
)

// AnalyticsController handles analytics-related requests
type AnalyticsController struct {
	config         *config.Config
	analyticsService *services.AnalyticsService
}

// NewAnalyticsController creates a new AnalyticsController instance
func NewAnalyticsController(cfg *config.Config) *AnalyticsController {
	return &AnalyticsController{
		config:         cfg,
		analyticsService: services.NewAnalyticsService(cfg),
	}
}

// GetGlobalAnalytics gets global system analytics
// @Summary Get global analytics
// @Description Get global system analytics
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/analytics/global [get]
func (ac *AnalyticsController) GetGlobalAnalytics(c *fiber.Ctx) error {
	analytics, err := ac.analyticsService.GetGlobalAnalytics()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get global analytics", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(analytics, "Global analytics retrieved successfully"))
}

// GetCentersAnalytics gets analytics by center
// @Summary Get centers analytics
// @Description Get analytics by center
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/analytics/centers [get]
func (ac *AnalyticsController) GetCentersAnalytics(c *fiber.Ctx) error {
	analytics, err := ac.analyticsService.GetCentersAnalytics()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get centers analytics", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(analytics, "Centers analytics retrieved successfully"))
}

// GetExamsAnalytics gets analytics by exam
// @Summary Get exams analytics
// @Description Get analytics by exam
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/analytics/exams [get]
func (ac *AnalyticsController) GetExamsAnalytics(c *fiber.Ctx) error {
	analytics, err := ac.analyticsService.GetExamsAnalytics()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get exams analytics", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(analytics, "Exams analytics retrieved successfully"))
}

// GetStudentsAnalytics gets analytics by student
// @Summary Get students analytics
// @Description Get analytics by student
// @Tags Analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/analytics/students [get]
func (ac *AnalyticsController) GetStudentsAnalytics(c *fiber.Ctx) error {
	analytics, err := ac.analyticsService.GetStudentsAnalytics()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get students analytics", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(analytics, "Students analytics retrieved successfully"))
}

// GetCenterAnalytics gets analytics for a specific center
// @Summary Get center analytics
// @Description Get analytics for a specific center
// @Tags Analytics
// @Accept json
// @Produce json
// @Param centerId path string true "Center ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/centers/{centerId}/analytics [get]
func (ac *AnalyticsController) GetCenterAnalytics(c *fiber.Ctx) error {
	centerID := c.Params("centerId")
	if centerID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Center ID is required", http.StatusBadRequest))
	}

	analytics, err := ac.analyticsService.GetCenterAnalytics(centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get center analytics", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(analytics, "Center analytics retrieved successfully"))
}