package main

import (
	"fmt"
	"log"
	"os"

	"kankor-backend/config"
	"kankor-backend/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	fmt.Printf("[Config] PDF Extractor URL: %s\n", cfg.PDFExtractorURL)
	fmt.Printf("[Config] Database: %s@%s:%s/%s\n", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// Connect to database
	config.ConnectDB(cfg)
	defer config.CloseDB()

	// Initialize Fiber app with error handling
	app := fiber.New(fiber.Config{
		ErrorHandler: config.ErrorHandler,
		BodyLimit:    20 * 1024 * 1024, // 20MB max request body size
	})

	// Middleware - order matters!
	app.Use(recover.New()) // Must be first to catch panics
	app.Use(logger.New())
	app.Use(cors.New())

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Kankor Exam Platform API is running",
		})
	})

	// Setup routes
	routes.SetupRoutes(app, cfg)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
