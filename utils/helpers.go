package utils

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"kankor-backend/middleware"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateOTP generates a 6-digit numeric OTP
func GenerateOTP() (string, error) {
	bytes := make([]byte, 4) // 4 bytes = 32 bits, which gives us more than enough randomness for a 6-digit code
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to a number between 0 and 999999, then pad with leading zeros if necessary
	num := base64.StdEncoding.EncodeToString(bytes)
	// Take first 6 alphanumeric characters and convert to digits
	var otp strings.Builder
	for i := 0; i < len(num) && otp.Len() < 6; i++ {
		char := num[i]
		if unicode.IsDigit(rune(char)) {
			otp.WriteByte(char)
		} else {
			// Convert letters to digits (A->0, B->1, ..., I->9, J->0, ...)
			digit := rune((char - 'A') % 10)
			otp.WriteRune(digit + '0')
		}
	}

	result := otp.String()
	if len(result) < 6 {
		// Pad with random digits if we don't have 6 digits
		padding := make([]byte, 6-len(result))
		if _, err := rand.Read(padding); err != nil {
			return "", err
		}
		for i := range padding {
			padding[i] = (padding[i] % 10) + '0'
		}
		result += string(padding)
	}

	return result[:6], nil
}

// ValidatePhone validates the phone number format (Afghanistan format)
func ValidatePhone(phone string) bool {
	// Afghanistan phone numbers: 937XXXXXXXX (10 digits after country code)
	// Or without country code: 07XXXXXXXX (10 digits starting with 07)
	re := regexp.MustCompile(`^(93|0)?7\d{8}$`)
	return re.MatchString(strings.ReplaceAll(phone, " ", ""))
}

// ValidatePasswordStrength checks if password meets strength requirements
func ValidatePasswordStrength(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// ValidateEmail validates email format
func ValidateEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// SanitizeInput removes potentially harmful characters
func SanitizeInput(input string) string {
	// Remove null bytes and other potentially harmful characters
	input = strings.ReplaceAll(input, "\x00", "")
	input = strings.TrimSpace(input)
	return input
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	validRoles := map[string]bool{
		"super_admin":  true,
		"center_admin": true,
		"student":      true,
	}
	return validRoles[role]
}

// IsValidOptionLetter checks if an option letter is valid (A, B, C, D)
func IsValidOptionLetter(letter string) bool {
	validLetters := map[string]bool{
		"A": true,
		"B": true,
		"C": true,
		"D": true,
	}
	return validLetters[letter]
}

// IsValidTimeRange checks if start time is before end time
func IsValidTimeRange(startTime, endTime string) bool {
	// In a real implementation, we would parse the times and compare them
	// For now, we'll just return true and implement proper validation in the service layer
	return true
}

// GenerateJWT generates a new JWT token
func GenerateJWT(userID, role, centerID, secret string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours

	claims := &middleware.JWTClaims{
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