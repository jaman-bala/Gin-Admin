package token

import (
	"context"
	"gin_auth_service/internal/pkg/jwt"
	"time"
)

// TokenPair holds a generated access/refresh token pair.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	AccessExpiry time.Time
}

// UseCase provides high-level operations for token management.
type UseCase interface {
	GenerateTokenPair(ctx context.Context, userID, role string, isActive bool, accessExp, refreshExp time.Duration) (*TokenPair, error)
	BlacklistToken(ctx context.Context, tokenString string, expiry time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error)
	GetTokenInfo(ctx context.Context, tokenString string) (*jwt.TokenInfo, error)
}
