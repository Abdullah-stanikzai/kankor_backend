package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/services"

	"github.com/gofiber/fiber/v2"
)

// CenterController handles educational center-related requests
type CenterController struct {
	config        *config.Config
	centerService *services.CenterService
}

// NewCenterController creates a new CenterController instance
func NewCenterController(cfg *config.Config) *CenterController {
	return &CenterController{
		config:        cfg,
		centerService: services.NewCenterService(cfg),
	}
}

// GetAllCenters gets all educational centers
// @Summary Get all centers
// @Description Get a list of all educational centers
// @Tags Centers
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Limit per page"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/centers [get]
func (cc *CenterController) GetAllCenters(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	centers, totalCount, err := cc.centerService.GetAllCenters(offset, limit)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get centers", http.StatusInternalServerError))
	}

	totalPages := (totalCount + limit - 1) / limit

	response := fiber.Map{
		"centers": centers,
		"pagination": fiber.Map{
			"current_page": page,
			"total_pages":  totalPages,
			"total_count":  totalCount,
			"per_page":     limit,
		},
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(response, "Centers retrieved successfully"))
}

// CreateCenter creates a new educational center
// @Summary Create center
// @Description Create a new educational center
// @Tags Centers
// @Accept json
// @Produce json
// @Param request body models.CreateEducationalCenterRequest true "Create Center Request"
// @Security BearerAuth
// @Success 201 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/centers [post]
func (cc *CenterController) CreateCenter(c *fiber.Ctx) error {
	var req models.CreateEducationalCenterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Center name is required", http.StatusBadRequest))
	}

	fmt.Printf("Creating center with Name: '%s', Description: '%s', AdminUserID: '%s'\n", req.Name, req.Description, req.AdminUserID)

	center, err := cc.centerService.CreateCenter(&req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to create center", http.StatusInternalServerError))
	}

	return c.Status(http.StatusCreated).JSON(config.SuccessResponse(center, "Center created successfully"))
}

// GetCenterByID gets an educational center by ID
// @Summary Get center by ID
// @Description Get an educational center by its ID
// @Tags Centers
// @Accept json
// @Produce json
// @Param id path string true "Center ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Router /super-admin/centers/{id} [get]
func (cc *CenterController) GetCenterByID(c *fiber.Ctx) error {
	centerID := c.Params("id")
	if centerID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Center ID is required", http.StatusBadRequest))
	}

	center, err := cc.centerService.GetCenterByID(centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Center not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(center, "Center retrieved successfully"))
}

// UpdateCenter updates an educational center
// @Summary Update center
// @Description Update an educational center
// @Tags Centers
// @Accept json
// @Produce json
// @Param id path string true "Center ID"
// @Param request body models.CreateEducationalCenterRequest true "Update Center Request"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Router /super-admin/centers/{id} [put]
func (cc *CenterController) UpdateCenter(c *fiber.Ctx) error {
	centerID := c.Params("id")
	if centerID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Center ID is required", http.StatusBadRequest))
	}

	var req models.CreateEducationalCenterRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Printf("DEBUG: UpdateCenter - Body parsing failed: %v\n", err)
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	fmt.Printf("DEBUG: UpdateCenter controller - Received request: CenterID=%s, Name=%s, AdminUserID=%s, Description=%s\n",
		centerID, req.Name, req.AdminUserID, req.Description)

	center, err := cc.centerService.UpdateCenter(centerID, &req)
	if err != nil {
		fmt.Printf("DEBUG: UpdateCenter controller - Service failed: %v\n", err)
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Center not found", http.StatusNotFound))
	}

	fmt.Printf("DEBUG: UpdateCenter controller - Success, returning center with AdminUserID: %v\n", center.AdminUserID)
	return c.Status(http.StatusOK).JSON(config.SuccessResponse(center, "Center updated successfully"))
}

// DeleteCenter deletes an educational center
// @Summary Delete center
// @Description Delete an educational center
// @Tags Centers
// @Accept json
// @Produce json
// @Param id path string true "Center ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/centers/{id} [delete]
func (cc *CenterController) DeleteCenter(c *fiber.Ctx) error {
	centerID := c.Params("id")
	if centerID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Center ID is required", http.StatusBadRequest))
	}

	err := cc.centerService.DeleteCenter(centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to delete center", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Center deleted successfully"))
}

// ActivateCenter activates an educational center
// @Summary Activate center
// @Description Activate an educational center
// @Tags Centers
// @Accept json
// @Produce json
// @Param id path string true "Center ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/centers/{id}/activate [patch]
func (cc *CenterController) ActivateCenter(c *fiber.Ctx) error {
	centerID := c.Params("id")
	if centerID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Center ID is required", http.StatusBadRequest))
	}

	center, err := cc.centerService.ActivateCenter(centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to activate center", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(center, "Center activated successfully"))
}

// DeactivateCenter deactivates an educational center
// @Summary Deactivate center
// @Description Deactivate an educational center
// @Tags Centers
// @Accept json
// @Produce json
// @Param id path string true "Center ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/centers/{id}/deactivate [patch]
func (cc *CenterController) DeactivateCenter(c *fiber.Ctx) error {
	centerID := c.Params("id")
	if centerID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Center ID is required", http.StatusBadRequest))
	}

	center, err := cc.centerService.DeactivateCenter(centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to deactivate center", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(center, "Center deactivated successfully"))
}
