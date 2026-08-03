package controllers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"kankor-backend/config"
	"kankor-backend/models"
	"kankor-backend/services"
)

// AdminController handles center admin-related requests for super admins
type AdminController struct {
	config      *config.Config
	adminService *services.AdminService
}

// NewAdminController creates a new AdminController instance
func NewAdminController(cfg *config.Config) *AdminController {
	return &AdminController{
		config:       cfg,
		adminService: services.NewAdminService(cfg),
	}
}

// GetAllCenterAdmins gets all center admins
// @Summary Get all center admins
// @Description Get all center admins
// @Tags Admins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/center-admins [get]
func (ac *AdminController) GetAllCenterAdmins(c *fiber.Ctx) error {
	admins, err := ac.adminService.GetAllCenterAdmins()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get center admins", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(admins, "Center admins retrieved successfully"))
}

// GetCenterAdminByID gets a center admin by ID
// @Summary Get center admin by ID
// @Description Get a center admin by their ID
// @Tags Admins
// @Accept json
// @Produce json
// @Param id path string true "Admin ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Router /super-admin/center-admins/{id} [get]
func (ac *AdminController) GetCenterAdminByID(c *fiber.Ctx) error {
	adminID := c.Params("id")
	if adminID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Admin ID is required", http.StatusBadRequest))
	}

	admin, err := ac.adminService.GetCenterAdminByID(adminID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Center admin not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(admin, "Center admin retrieved successfully"))
}

// CreateCenterAdmin creates a new center admin
// @Summary Create center admin
// @Description Create a new center admin
// @Tags Admins
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "Create Center Admin Request"
// @Security BearerAuth
// @Success 201 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/center-admins [post]
func (ac *AdminController) CreateCenterAdmin(c *fiber.Ctx) error {
	var req models.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Validate required fields
	if req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Password is required", http.StatusBadRequest))
	}

	if req.FullName == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Full name is required", http.StatusBadRequest))
	}

	// Ensure the role is center_admin
	req.Role = "center_admin"

	admin, err := ac.adminService.CreateCenterAdmin(&req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to create center admin", http.StatusInternalServerError))
	}

	return c.Status(http.StatusCreated).JSON(config.SuccessResponse(admin, "Center admin created successfully"))
}

// UpdateCenterAdmin updates a center admin
// @Summary Update center admin
// @Description Update a center admin
// @Tags Admins
// @Accept json
// @Produce json
// @Param id path string true "Admin ID"
// @Param request body struct{FullName string `json:"full_name"`;Email string `json:"email"`} true "Update Center Admin Request"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Router /super-admin/center-admins/{id} [put]
func (ac *AdminController) UpdateCenterAdmin(c *fiber.Ctx) error {
	adminID := c.Params("id")
	if adminID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Admin ID is required", http.StatusBadRequest))
	}

	var req struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	admin, err := ac.adminService.UpdateCenterAdmin(adminID, req.FullName, req.Email)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Center admin not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(admin, "Center admin updated successfully"))
}

// DeleteCenterAdmin deletes a center admin
// @Summary Delete center admin
// @Description Delete a center admin
// @Tags Admins
// @Accept json
// @Produce json
// @Param id path string true "Admin ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /super-admin/center-admins/{id} [delete]
func (ac *AdminController) DeleteCenterAdmin(c *fiber.Ctx) error {
	adminID := c.Params("id")
	if adminID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Admin ID is required", http.StatusBadRequest))
	}

	err := ac.adminService.DeleteCenterAdmin(adminID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to delete center admin", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Center admin deleted successfully"))
}