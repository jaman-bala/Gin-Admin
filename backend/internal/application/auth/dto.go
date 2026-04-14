package auth

import (
	"gin_auth_service/internal/application/user"
	"time"
)

// LoginRequestDTO represents a login request.
type LoginRequestDTO struct {
	Phone    string `json:"phone" validate:"required,e164" example:"+996500500500"`
	Password string `json:"password" validate:"required,min=8" example:"Password123"`
}

// LoginResponseDTO represents a login response containing tokens.
type LoginResponseDTO struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Message      string `json:"message"`
}

// TokenResponseDTO represents a response with a new access token and user info.
type TokenResponseDTO struct {
	AccessToken string               `json:"access_token"`
	User        user.UserResponseDTO `json:"user"`
	ExpiresAt   time.Time            `json:"expires_at"`
	Message     string               `json:"message,omitempty"`
}

// RefreshTokenRequest represents a request to refresh an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"refresh_token"`
}
