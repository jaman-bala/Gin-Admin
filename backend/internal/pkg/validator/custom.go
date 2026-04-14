// Package validator provides custom validation logic for the application.
package validator

import (
	"log/slog"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// InitCustomValidators registers custom validation functions with Gin's default validator.
func InitCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		slog.Info("Registering custom validators")
		// Register a custom password strength validator you can use via `validate:"strong_password"`
		if err := v.RegisterValidation("strong_password", ValidateStrongPassword); err != nil {
			slog.Error("Failed to register strong_password validator", "error", err)
		}
	} else {
		slog.Warn("Could not access Gin's default validator engine")
	}
}

// ValidateStrongPassword checks if a password meets complexity requirements
func ValidateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	// Minimum 8 chars, at least one uppercase, lowercase, and number
	if len(password) < 8 {
		return false
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	return hasUpper && hasLower && hasNumber
}
