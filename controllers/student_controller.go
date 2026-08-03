package controllers

import (
	"fmt"
	"net/http"

	"kankor-backend/config"
	"kankor-backend/services"

	"github.com/gofiber/fiber/v2"
)

// StudentController handles student-related requests for center admins
type StudentController struct {
	config         *config.Config
	studentService *services.StudentService
}

// NewStudentController creates a new StudentController instance
func NewStudentController(cfg *config.Config) *StudentController {
	return &StudentController{
		config:         cfg,
		studentService: services.NewStudentService(cfg),
	}
}

// GetAllStudents gets all students in a center
// @Summary Get all students
// @Description Get all students enrolled in a center
// @Tags Students
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/students [get]
func (sc *StudentController) GetAllStudents(c *fiber.Ctx) error {
	// Get user info from context
	centerID := c.Locals("user_center_id").(string)
	fmt.Printf("[GetAllStudents] centerID from context: %s\n", centerID)

	students, err := sc.studentService.GetAllStudents(centerID)
	if err != nil {
		fmt.Printf("[GetAllStudents] Service error: %v\n", err)
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to get students", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(students, "Students retrieved successfully"))
}

// GetStudentByID gets a student by ID
// @Summary Get student by ID
// @Description Get a student by their ID
// @Tags Students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Router /center-admin/students/{id} [get]
func (sc *StudentController) GetStudentByID(c *fiber.Ctx) error {
	studentID := c.Params("id")
	if studentID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Student ID is required", http.StatusBadRequest))
	}

	// Get user info from context
	centerID := c.Locals("user_center_id").(string)

	student, err := sc.studentService.GetStudentByID(studentID, centerID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(config.ErrorResponse("Student not found", http.StatusNotFound))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(student, "Student retrieved successfully"))
}

// CreateStudent creates a new student
// @Summary Create student
// @Description Create a new student and enroll in center
// @Tags Students
// @Accept json
// @Produce json
// @Param request body struct{Name string `json:"name"` Email string `json:"email"` Phone string `json:"phone"`} true "Create Student Request"
// @Security BearerAuth
// @Success 201 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/students [post]
func (sc *StudentController) CreateStudent(c *fiber.Ctx) error {
	var req struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
		Phone string `json:"phone" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(config.ErrorResponse("Invalid request body", 400))
	}

	// Get center ID from context
	centerID := c.Locals("user_center_id").(string)
	if centerID == "" {
		return c.Status(400).JSON(config.ErrorResponse("Center ID is required", 400))
	}

	student, err := sc.studentService.CreateStudent(req.Name, req.Email, req.Phone, centerID)
	if err != nil {
		return c.Status(500).JSON(config.ErrorResponse(err.Error(), 500))
	}

	return c.Status(201).JSON(config.SuccessResponse(student, "Student created successfully"))
}

// EnrollStudent enrolls a student in a center
// @Summary Enroll student
// @Description Enroll a student in an educational center
// @Tags Students
// @Accept json
// @Produce json
// @Param request body struct{StudentID string `json:"student_id"`} true "Enroll Student Request"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/students/enroll [post]
func (sc *StudentController) EnrollStudent(c *fiber.Ctx) error {
	var req struct {
		StudentID string `json:"student_id" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Invalid request body", http.StatusBadRequest))
	}

	// Get user info from context
	centerID := c.Locals("user_center_id").(string)

	enrollment, err := sc.studentService.EnrollStudent(req.StudentID, centerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(config.ErrorResponse("Failed to enroll student", http.StatusInternalServerError))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(enrollment, "Student enrolled successfully"))
}

// UpdateStudent updates a student's details
// @Summary Update student
// @Description Update a student's name, email, or phone
// @Tags Students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Param request body struct{Name string `json:"name"` Email string `json:"email"` Phone string `json:"phone"`} true "Update Student Request"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/students/{id} [put]
func (sc *StudentController) UpdateStudent(c *fiber.Ctx) error {
	studentID := c.Params("id")
	if studentID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Student ID is required", http.StatusBadRequest))
	}

	var req struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
		Phone string `json:"phone"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(config.ErrorResponse("Invalid request body", 400))
	}

	// Get user info from context
	centerID := c.Locals("user_center_id").(string)

	student, err := sc.studentService.UpdateStudent(studentID, req.Name, req.Email, req.Phone, centerID)
	if err != nil {
		return c.Status(500).JSON(config.ErrorResponse(err.Error(), 500))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(student, "Student updated successfully"))
}

// DeleteStudent deletes a student
// @Summary Delete student
// @Description Delete a student from the system
// @Tags Students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Security BearerAuth
// @Success 200 {object} config.APIResponse
// @Failure 400 {object} config.APIResponse
// @Failure 401 {object} config.APIResponse
// @Failure 404 {object} config.APIResponse
// @Failure 500 {object} config.APIResponse
// @Router /center-admin/students/{id} [delete]
func (sc *StudentController) DeleteStudent(c *fiber.Ctx) error {
	studentID := c.Params("id")
	if studentID == "" {
		return c.Status(http.StatusBadRequest).JSON(config.ErrorResponse("Student ID is required", http.StatusBadRequest))
	}

	// Get user info from context
	centerID := c.Locals("user_center_id").(string)

	err := sc.studentService.DeleteStudent(studentID, centerID)
	if err != nil {
		return c.Status(500).JSON(config.ErrorResponse(err.Error(), 500))
	}

	return c.Status(http.StatusOK).JSON(config.SuccessResponse(nil, "Student deleted successfully"))
}
