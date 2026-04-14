package token

import (
	"context"
	"fmt"
	"gin_auth_service/internal/domain/token"
	"gin_auth_service/internal/pkg/jwt"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"
)

// usecase provides high-level operations for token management.
type usecase struct {
	repo token.Repository
	jwt  jwt.JWTService
}

// NewUseCase creates a new instance of UseCase.
func NewUseCase(repo token.Repository, jwtService jwt.JWTService) UseCase {
	return &usecase{
		repo: repo,
		jwt:  jwtService,
	}
}

func (uc *usecase) BlacklistToken(ctx context.Context, tokenString string, expiry time.Time) error {
	if tokenString == "" {
		return fmt.Errorf("token string is required")
	}

	tokenKey := fmt.Sprintf("blacklist:%s", tokenString)
	blacklistExpiry := time.Until(expiry)
	if blacklistExpiry <= 0 {
		blacklistExpiry = time.Hour
	}

	return uc.repo.Set(ctx, tokenKey, "blacklisted", blacklistExpiry)
}

func (uc *usecase) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if tokenString == "" {
		return false, fmt.Errorf("token string is required")
	}

	tokenKey := fmt.Sprintf("blacklist:%s", tokenString)
	return uc.repo.Exists(ctx, tokenKey)
}

func (uc *usecase) GetTokenInfo(ctx context.Context, tokenString string) (*jwt.TokenInfo, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token string is required")
	}

	token, err := uc.jwt.ParseToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	isBlacklisted, err := uc.IsTokenBlacklisted(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to check blacklist status: %w", err)
	}
	if isBlacklisted {
		return nil, fmt.Errorf("token is blacklisted")
	}

	claims, ok := token.Claims.(jwtv4.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Helper function for claims extraction
	getStr := func(key string) string {
		if val, ok := claims[key].(string); ok {
			return val
		}
		return ""
	}
	getFloat := func(key string) float64 {
		if val, ok := claims[key].(float64); ok {
			return val
		}
		return 0
	}

	tokenInfo := &jwt.TokenInfo{
		UserID:    getStr("user_id"),
		Role:      getStr("role"),
		Type:      getStr("type"),
		ExpiresAt: time.Unix(int64(getFloat("exp")), 0),
		IssuedAt:  time.Unix(int64(getFloat("iat")), 0),
		JTI:       getStr("jti"),
	}

	return tokenInfo, nil
}
