package token

import (
	"context"
	"gin_auth_service/internal/pkg/jwt"
	"time"
)

// UseCase provides high-level operations for token management.
type UseCase interface {
	BlacklistToken(ctx context.Context, tokenString string, expiry time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error)
	GetTokenInfo(ctx context.Context, tokenString string) (*jwt.TokenInfo, error)
}
