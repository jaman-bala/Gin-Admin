package auth

import (
	domainUser "gin_auth_service/internal/domain/user"
	"time"

	"uuid"
)

// LoginRequestDTO represents a login request.
// Password intentionally only requires non-empty, not strong_password: this
// is a credential check against an existing hash, not a password-creation
// endpoint, and a strength rule here would lock out any account whose
// password predates the rule (e.g. the bootstrap admin).
type LoginRequestDTO struct {
	Phone    string `json:"phone"     validate:"required,e164"  example:"+996500500500"`
	Password string `json:"password"  validate:"required"       example:"Password123"`
}

// LoginResponseDTO represents a login response containing tokens.
type LoginResponseDTO struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Message      string `json:"message"`
}

// TokenResponseDTO represents a response with a new token pair and user info.
type TokenResponseDTO struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         AuthUserDTO `json:"user"`
	ExpiresAt    time.Time   `json:"expires_at"`
	Message      string      `json:"message,omitempty"`
}

// AuthUserDTO is auth's own view of a user, shaped identically to
// application/user.UserResponseDTO on the wire. It's declared here — not
// imported from application/user — so this package depends only on the
// domain entity, not on a sibling use case's DTOs and (transitively) its
// business logic. The two are free to diverge; today they intentionally
// match so /auth/refresh's response shape is unchanged.
type AuthUserDTO struct {
	ID         uuid.UUID       `json:"id"`
	FirstName  string          `json:"first_name"`
	LastName   string          `json:"last_name"`
	MiddleName string          `json:"middle_name"`
	Phone      string          `json:"phone"`
	Role       domainUser.Role `json:"role"`
	Photo      string          `json:"photo"`
	Telegram   string          `json:"telegram"`
	IsActive   bool            `json:"is_active"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	DeletedAt  *time.Time      `json:"deleted_at"`
}

// FromDomain maps a domain user to the wire DTO. photoURL should already be
// resolved (presigned) by the caller — this package has no storage concerns.
func AuthUserFromDomain(u *domainUser.User, photoURL string) AuthUserDTO {
	return AuthUserDTO{
		ID:         u.ID,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		MiddleName: u.MiddleName,
		Phone:      u.Phone,
		Role:       u.Role,
		Photo:      photoURL,
		Telegram:   u.Telegram,
		IsActive:   u.IsActive,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
		DeletedAt:  u.DeletedAt,
	}
}

// RefreshTokenRequest represents a request to refresh an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"refresh_token"`
}

// LogoutRequestDTO carries the optional refresh token to invalidate on logout.
// The access token is taken from the Authorization header.
type LogoutRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}
