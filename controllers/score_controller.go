package controllers

import (
	"net/http"

	"kankor-backend/config"
	"kankor-backend/services"

	"github.com/gofiber/fiber/v2"
)

// ScoreController handles center-admin student score requests.
type ScoreController struct {
	scoreService *services.ScoreService
}

func NewScoreController(cfg *config.Config) *ScoreController {
	return &ScoreController{scoreService: services.NewScoreService(cfg)}
}

// GetCenterScores returns all submitted scores for the authenticated center.
func (sc *ScoreController) GetCenterScores(c *fiber.Ctx) error {
	centerID, ok := c.Locals("user_center_id").(string)
	if !ok || centerID == "" {
		return c.Status(http.StatusForbidden).JSON(
			config.ErrorResponse("Center assignment is required", http.StatusForbidden),
		)
	}

	response, err := sc.scoreService.GetCenterScores(centerID, nil)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			config.ErrorResponse("Failed to load student scores", http.StatusInternalServerError),
		)
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(response, "Student scores retrieved successfully"),
	)
}

// GetStudentScores returns one student's scores after applying the same center
// ownership restriction as the complete list endpoint.
func (sc *ScoreController) GetStudentScores(c *fiber.Ctx) error {
	centerID, ok := c.Locals("user_center_id").(string)
	if !ok || centerID == "" {
		return c.Status(http.StatusForbidden).JSON(
			config.ErrorResponse("Center assignment is required", http.StatusForbidden),
		)
	}

	studentID := c.Params("studentId")
	response, err := sc.scoreService.GetCenterScores(centerID, &studentID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			config.ErrorResponse("Failed to load student scores", http.StatusInternalServerError),
		)
	}

	return c.Status(http.StatusOK).JSON(
		config.SuccessResponse(response, "Student scores retrieved successfully"),
	)
}
