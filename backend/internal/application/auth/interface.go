package auth

import (
	"context"
	"gin_auth_service/internal/application/user"
	domainUser "gin_auth_service/internal/domain/user"

	"github.com/google/uuid"
)

// UseCase provides authentication and registration logic.
type UseCase interface {
	Login(ctx context.Context, req LoginRequestDTO) (*LoginResponseDTO, error)
	Logout(ctx context.Context, tokenString string) error
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponseDTO, error)
	GetUserInfoFromToken(ctx context.Context, tokenString string) (*domainUser.User, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*user.UserResponseDTO, error)
}
