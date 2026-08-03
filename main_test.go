package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"kankor-backend/config"
)

func TestHealthEndpoint(t *testing.T) {
	// Setup
	app := fiber.New()

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Kankor Exam Platform API is running",
		})
	})

	// Create a test request
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)

	// Assertions
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestLoadConfig(t *testing.T) {
	cfg := config.LoadConfig()

	if cfg.DBHost == "" {
		t.Error("DBHost should not be empty")
	}

	if cfg.DBPort == "" {
		t.Error("DBPort should not be empty")
	}

	if cfg.JWTSecret == "" {
		t.Error("JWTSecret should not be empty")
	}
}
