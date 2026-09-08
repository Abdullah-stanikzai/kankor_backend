package routes

import (
	"kankor-backend/config"
	"kankor-backend/controllers"
	"kankor-backend/middleware"
	"kankor-backend/services"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes sets up all the API routes
func SetupRoutes(app *fiber.App, cfg *config.Config) {
	// Public routes
	public := app.Group("/api/v1")

	// Authentication routes (admin/center-admin)
	authController := controllers.NewAuthController(cfg)
	auth := public.Group("/auth")
	auth.Post("/login", authController.Login)
	auth.Post("/register", authController.Register)
	auth.Post("/refresh-token", authController.RefreshToken)
	auth.Post("/logout", authController.Logout)

	// Student authentication routes (separate from admin auth)
	studentAuthController := controllers.NewStudentAuthController(cfg)
	studentAuth := public.Group("/auth/student")
	studentAuth.Post("/register", studentAuthController.Register)
	studentAuth.Post("/login", studentAuthController.Login)

	// Protected student auth routes (require JWT)
	studentAuthProtected := public.Group("/auth/student")
	studentAuthProtected.Use(middleware.AuthMiddleware(cfg))
	studentAuthProtected.Use(middleware.RoleMiddleware("student"))
	studentAuthProtected.Get("/me", studentAuthController.GetMe)
	studentAuthProtected.Put("/profile", studentAuthController.UpdateProfile)

	// Protected routes for Super Admin
	superAdmin := public.Group("/super-admin")
	superAdmin.Use(middleware.AuthMiddleware(cfg))
	superAdmin.Use(middleware.RoleMiddleware("super_admin"))

	// Educational centers routes
	centerController := controllers.NewCenterController(cfg)
	superAdmin.Get("/centers", centerController.GetAllCenters)
	superAdmin.Post("/centers", centerController.CreateCenter)
	superAdmin.Get("/centers/:id", centerController.GetCenterByID)
	superAdmin.Put("/centers/:id", centerController.UpdateCenter)
	superAdmin.Delete("/centers/:id", centerController.DeleteCenter)
	superAdmin.Patch("/centers/:id/activate", centerController.ActivateCenter)
	superAdmin.Patch("/centers/:id/deactivate", centerController.DeactivateCenter)

	// Center admin routes
	adminController := controllers.NewAdminController(cfg)
	superAdmin.Get("/center-admins", adminController.GetAllCenterAdmins)
	superAdmin.Post("/center-admins", adminController.CreateCenterAdmin)
	superAdmin.Put("/center-admins/:id", adminController.UpdateCenterAdmin)
	superAdmin.Delete("/center-admins/:id", adminController.DeleteCenterAdmin)
	superAdmin.Get("/center-admins/:id", adminController.GetCenterAdminByID)

	// System analytics
	analyticsController := controllers.NewAnalyticsController(cfg)
	superAdmin.Get("/analytics/global", analyticsController.GetGlobalAnalytics)
	superAdmin.Get("/analytics/centers", analyticsController.GetCentersAnalytics)
	superAdmin.Get("/analytics/exams", analyticsController.GetExamsAnalytics)
	superAdmin.Get("/analytics/students", analyticsController.GetStudentsAnalytics)

	// System settings
	settingsController := controllers.NewSystemSettingsController(cfg)
	superAdmin.Get("/settings", settingsController.GetSystemSettings)
	superAdmin.Put("/settings", settingsController.UpdateSystemSettings)

	// Protected routes for Center Admin
	centerAdmin := public.Group("/center-admin")
	centerAdmin.Use(middleware.AuthMiddleware(cfg))
	centerAdmin.Use(middleware.RoleMiddleware("center_admin"))

	// Exam management
	examController := controllers.NewExamController(cfg)
	centerAdmin.Post("/exams", examController.CreateExam)
	centerAdmin.Get("/exams", examController.GetAllExams)
	centerAdmin.Get("/exams/:id", examController.GetExamByID)
	centerAdmin.Put("/exams/:id", examController.UpdateExam)
	centerAdmin.Delete("/exams/:id", examController.DeleteExam)
	centerAdmin.Patch("/exams/:id/publish", examController.PublishExam)
	centerAdmin.Patch("/exams/:id/unpublish", examController.UnpublishExam)
	centerAdmin.Patch("/exams/:id/status", examController.UpdateExamStatus) // NEW: Update exam status

	// Question routes
	centerAdmin.Get("/exams/:examId/questions", examController.GetExamQuestionsAdmin)
	centerAdmin.Post("/exams/:examId/questions", examController.CreateQuestion)
	centerAdmin.Get("/exams/:examId/questions/:questionId", examController.GetQuestionByIDAdmin)
	centerAdmin.Put("/exams/:examId/questions/:questionId", examController.UpdateQuestionAdmin)
	centerAdmin.Delete("/exams/:examId/questions/:questionId", examController.DeleteQuestionAdmin)
	centerAdmin.Delete("/exams/:id/questions/:questionId", examController.DeleteExamQuestion)

	// Student management
	studentController := controllers.NewStudentController(cfg)
	centerAdmin.Post("/students", studentController.CreateStudent)
	centerAdmin.Get("/students", studentController.GetAllStudents)
	centerAdmin.Post("/students/enroll", studentController.EnrollStudent)

	// Student Activation (MUST be registered BEFORE /students/:id to avoid route shadowing)
	studentActivationController := controllers.NewStudentActivationController(cfg)
	centerAdmin.Get("/students/search", studentActivationController.Search)
	centerAdmin.Patch("/students/:id/activate", studentActivationController.Activate)
	centerAdmin.Patch("/students/:id/deactivate", studentActivationController.Deactivate)

	// Wildcard :id routes (registered LAST so they don't capture "search", "activate", etc.)
	centerAdmin.Get("/students/:id", studentController.GetStudentByID)
	centerAdmin.Put("/students/:id", studentController.UpdateStudent)
	centerAdmin.Delete("/students/:id", studentController.DeleteStudent)

	// Question Bank routes
	geminiService := services.NewGeminiService(cfg.GeminiAPIKey)
	questionBankService := services.NewQuestionBankService(geminiService, cfg.PDFExtractorURL)
	questionBankController := controllers.NewQuestionBankController(cfg, questionBankService)
	centerAdmin.Post("/question-bank/upload", questionBankController.UploadPDF)
	centerAdmin.Post("/question-bank/save", questionBankController.SaveQuestions)
	centerAdmin.Get("/question-bank", questionBankController.GetQuestions)
	centerAdmin.Get("/question-bank/subjects", questionBankController.GetSubjects)
	centerAdmin.Get("/question-bank/:id", questionBankController.GetQuestion)
	centerAdmin.Put("/question-bank/:id", questionBankController.UpdateQuestion)
	centerAdmin.Delete("/question-bank/:id", questionBankController.DeleteQuestion)

	// Results and analytics
	centerAdmin.Get("/exams/:examId/results", examController.GetExamResults)
	centerAdmin.Get("/exams/:examId/results/:attemptId", examController.GetExamResult)
	centerAdmin.Get("/exams/:examId/analytics", examController.GetExamAnalytics)
	centerAdmin.Get("/centers/:centerId/analytics", analyticsController.GetCenterAnalytics)

	// Protected routes for Students
	student := public.Group("/student")
	student.Use(middleware.AuthMiddleware(cfg))
	student.Use(middleware.RoleMiddleware("student"))

	// Student profile
	studentProfileController := controllers.NewStudentProfileController(cfg)
	student.Get("/profile", studentProfileController.GetProfile)
	student.Put("/profile", studentProfileController.UpdateProfile)

	// Student exam routes
	student.Get("/exams/upcoming", examController.GetUpcomingExams)
	student.Get("/exams/available", examController.GetAvailableExams)
	student.Get("/exams/history", examController.GetExamHistory)
	// Consolidated sync endpoint (MUST be registered before /exams/:id to avoid route shadowing)
	student.Get("/exams/sync", examController.GetSyncData)
	student.Get("/exams/:id", examController.GetExamDetails)
	student.Get("/exams/:id/submission-status", examController.CheckExamSubmissionStatus)
	student.Post("/exams/:id/start", examController.StartExam)
	// DEPRECATED: Using exam engine routes below instead
	// student.Get("/exams/:id/questions", examController.GetExamQuestions)
	// student.Post("/exams/:id/submit", examController.SubmitExam)
	student.Get("/exams/:id/results", examController.GetStudentExamResults)

	// Exam Engine routes (NEW - For complete exam taking experience)
	examEngineService := services.NewExamEngineService(cfg)
	examEngineController := controllers.NewExamEngineController(examEngineService)

	// Get questions for an exam (formatted for taking)
	student.Get("/exams/:exam_id/questions", examEngineController.GetQuestionsForExam)

	// Save/update answer (auto-save)
	student.Post("/answers", examEngineController.SaveAnswer)

	// Submit exam and get instant results
	student.Post("/exams/:exam_id/submit", examEngineController.SubmitExam)

	// Get exam result
	student.Get("/exams/:exam_id/result", examEngineController.GetExamResult)

	// Get ALL exam results for student
	student.Get("/results", examEngineController.GetAllResults)
}
