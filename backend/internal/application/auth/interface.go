package auth

import (
	"context"
	domainUser "gin_auth_service/internal/domain/user"
)

// UseCase provides authentication logic.
type UseCase interface {
	Login(ctx context.Context, req LoginRequestDTO) (*LoginResponseDTO, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponseDTO, error)
	GetUserInfoFromToken(ctx context.Context, tokenString string) (*domainUser.User, error)
}