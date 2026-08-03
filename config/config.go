package config

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	JWTSecret       string
	RedisAddr       string
	AppEnv          string
	GeminiAPIKey    string
	PDFExtractorURL string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "123"),
		DBName:     getEnv("DB_NAME", "kankor_db"),
		JWTSecret:  getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		// RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		AppEnv:          getEnv("APP_ENV", "development"),
		GeminiAPIKey:    getEnv("GEMINI_API_KEY", ""),
		PDFExtractorURL: getEnv("PDF_EXTRACTOR_URL", "http://localhost:5000"),
	}
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ErrorHandler custom error handler for Fiber
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a *fiber.Error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Return error as JSON
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    code,
			"message": err.Error(),
		},
		"timestamp": c.App().Config().Prefork,
	})
}

// Response structure for API responses
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// SuccessResponse creates a successful API response
func SuccessResponse(data interface{}, message string) *APIResponse {
	return &APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: "",
	}
}

// ErrorResponse creates an error API response
func ErrorResponse(message string, code int) *APIResponse {
	return &APIResponse{
		Success: false,
		Data: fiber.Map{
			"code":    code,
			"message": message,
		},
		Message:   "",
		Timestamp: "",
	}
}
