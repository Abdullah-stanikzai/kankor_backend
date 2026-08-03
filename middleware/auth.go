package middleware

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"kankor-backend/config"
	"kankor-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware handles JWT authentication
func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(config.ErrorResponse("Authorization header required", 401))
		}

		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			fmt.Printf("Received token: %s\n", tokenString[:20])
			fmt.Printf("Full token length: %d\n", len(tokenString))
		} else {
			fmt.Printf("Invalid auth header format: %s\n", authHeader)
			return c.Status(401).JSON(config.ErrorResponse("Invalid authorization header format", 401))
		}

		fmt.Printf("Validating token with secret: %s\n", cfg.JWTSecret)
		claims, err := ValidateJWT(tokenString, cfg.JWTSecret)
		if err != nil {
			fmt.Printf("Token validation failed: %v\n", err)
			fmt.Printf("Token string: %s...\n", tokenString[:30])
			return c.Status(401).JSON(config.ErrorResponse("Invalid or expired token", 401))
		}

		fmt.Printf("✅ Token valid - User ID: %s, Role: %s, Center: %s\n", claims.UserID, claims.Role, claims.CenterID)

		// Add user info to context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_role", claims.Role)
		c.Locals("user_center_id", claims.CenterID)
		c.Locals("userClaims", claims) // Store full claims for controllers that need it

		return c.Next()
	}
}

// RoleMiddleware checks if the user has the required role
func RoleMiddleware(requiredRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole == nil {
			return c.Status(401).JSON(config.ErrorResponse("User role not found in token", 401))
		}

		userRoleStr := userRole.(string)

		// Super admin has access to all routes
		if userRoleStr == "super_admin" {
			return c.Next()
		}

		for _, role := range requiredRoles {
			if userRoleStr == role {
				return c.Next()
			}
		}

		return c.Status(403).JSON(config.ErrorResponse("Insufficient permissions", 403))
	}
}

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	CenterID string `json:"center_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a new JWT token
func GenerateJWT(userID, role, centerID string, secret string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours

	claims := &JWTClaims{
		UserID:   userID,
		Role:     role,
		CenterID: centerID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT validates the JWT token and returns the claims
func ValidateJWT(tokenString string, secret string) (*JWTClaims, error) {
	claims := &JWTClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GetUserFromContext extracts user information from the context
func GetUserFromContext(c *fiber.Ctx) (*models.User, error) {
	userID := c.Locals("user_id")
	if userID == nil {
		return nil, errors.New("user not found in context")
	}

	role := c.Locals("user_role").(string)
	// centerID := c.Locals("user_center_id") // Not stored in User model

	user := &models.User{
		ID:   userID.(string),
		Role: role,
	}

	// Note: CenterID is available in context via c.Locals("user_center_id") but not stored in User model

	return user, nil
}
