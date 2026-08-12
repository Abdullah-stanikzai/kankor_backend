package controllers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"kankor-backend/config"
	"kankor-backend/services"
)

// StudentActivationController handles student activation/deactivation by center admins.
type StudentActivationController struct {
	config        *config.Config
	activationSvc *services.StudentActivationService
}

// NewStudentActivationController creates a new StudentActivationController instance.
func NewStudentActivationController(cfg *config.Config) *StudentActivationController {
	return &StudentActivationController{
		config:        cfg,
		activationSvc: services.NewStudentActivationService(cfg),
	}
}

// Search searches for a student by email (required) and optionally phone.
// GET /api/v1/center-admin/students/search?email=...&phone=...
func (sc *StudentActivationController) Search(c *fiber.Ctx) error {
	adminCenterID := c.Locals("user_center_id")
	if adminCenterID == nil || adminCenterID.(string) == "" {
		return c.Status(http.StatusForbidden).JSON(
			config.ErrorResponse("Center admin center ID not found", http.StatusForbidden),
		)
	}

	email := c.Query("email")
	phone := c.Query("phone")

	if email == "" {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("email query parameter is required", http.StatusBadRequest),
		)
	}

	result, err := sc.activationSvc.SearchStudent(email, phone, adminCenterID.(string))
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(
			config.ErrorResponse(err.Error(), http.StatusNotFound),
		)
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(result, "Student found"),
	)
}

// Activate assigns a student to the admin's center and sets status to "active".
// PATCH /api/v1/center-admin/students/:id/activate
func (sc *StudentActivationController) Activate(c *fiber.Ctx) error {
	adminCenterID := c.Locals("user_center_id")
	if adminCenterID == nil || adminCenterID.(string) == "" {
		return c.Status(http.StatusForbidden).JSON(
			config.ErrorResponse("Center admin center ID not found", http.StatusForbidden),
		)
	}

	studentID := c.Params("id")
	if studentID == "" {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("student ID is required", http.StatusBadRequest),
		)
	}

	user, err := sc.activationSvc.ActivateStudent(studentID, adminCenterID.(string))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse(err.Error(), http.StatusBadRequest),
		)
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(user, "Student activated successfully"),
	)
}

// Deactivate sets a student's status to "inactive" while keeping center assignment.
// PATCH /api/v1/center-admin/students/:id/deactivate
func (sc *StudentActivationController) Deactivate(c *fiber.Ctx) error {
	adminCenterID := c.Locals("user_center_id")
	if adminCenterID == nil || adminCenterID.(string) == "" {
		return c.Status(http.StatusForbidden).JSON(
			config.ErrorResponse("Center admin center ID not found", http.StatusForbidden),
		)
	}

	studentID := c.Params("id")
	if studentID == "" {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse("student ID is required", http.StatusBadRequest),
		)
	}

	user, err := sc.activationSvc.DeactivateStudent(studentID, adminCenterID.(string))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			config.ErrorResponse(err.Error(), http.StatusBadRequest),
		)
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(user, "Student deactivated successfully"),
	)
}
