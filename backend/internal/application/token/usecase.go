package token

import (
	"context"
	"fmt"
	domainToken "gin_auth_service/internal/domain/token"
	"gin_auth_service/internal/pkg/jwt"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type usecase struct {
	repo domainToken.Repository
	jwt  jwt.JWTService
}

func NewUseCase(repo domainToken.Repository, jwtService jwt.JWTService) UseCase {
	return &usecase{repo: repo, jwt: jwtService}
}

func (uc *usecase) GenerateTokenPair(_ context.Context, userID, role string, isActive bool, accessExp, refreshExp time.Duration) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(accessExp)
	refreshExpiry := now.Add(refreshExp)

	gen := func(expiry time.Time, tokenType string) (string, error) {
		return uc.jwt.CreateToken(jwt.MapClaims{
			"user_id":   userID,
			"role":      role,
			"is_active": isActive,
			"exp":       expiry.Unix(),
			"type":      tokenType,
			"jti":       uuid.Must(uuid.NewV7()).String(),
			"iat":       now.Unix(),
		})
	}

	accessToken, err := gen(accessExpiry, "access")
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err := gen(refreshExpiry, "refresh")
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessExpiry: accessExpiry,
	}, nil
}

// jtiFromToken extracts the jti claim from a JWT without verifying the signature.
// Used only for blacklist key lookup; full verification is done by the caller.
func jtiFromToken(tokenString string) (string, error) {
	p := jwtv4.NewParser()
	t, _, err := p.ParseUnverified(tokenString, jwtv4.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}
	claims, ok := t.Claims.(jwtv4.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims type")
	}
	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return "", fmt.Errorf("missing or empty jti claim")
	}
	return jti, nil
}

func (uc *usecase) BlacklistToken(ctx context.Context, tokenString string, expiry time.Time) error {
	if tokenString == "" {
		return fmt.Errorf("token string is required")
	}
	jti, err := jtiFromToken(tokenString)
	if err != nil {
		return fmt.Errorf("cannot blacklist token: %w", err)
	}
	ttl := time.Until(expiry)
	if ttl <= 0 {
		ttl = time.Hour
	}
	return uc.repo.Set(ctx, "blacklist:"+jti, "1", ttl)
}

func (uc *usecase) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if tokenString == "" {
		return false, fmt.Errorf("token string is required")
	}
	jti, err := jtiFromToken(tokenString)
	if err != nil {
		// Malformed token has no JTI — not a blacklist concern;
		// signature verification downstream will reject it.
		return false, nil
	}
	return uc.repo.Exists(ctx, "blacklist:"+jti)
}

func (uc *usecase) GetTokenInfo(ctx context.Context, tokenString string) (*jwt.TokenInfo, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token string is required")
	}

	token, err := uc.jwt.ParseToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwtv4.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

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
	getBool := func(key string) bool {
		if val, ok := claims[key].(bool); ok {
			return val
		}
		return false
	}

	jti := getStr("jti")
	if jti == "" {
		return nil, fmt.Errorf("token missing jti claim")
	}

	// Use JTI directly — token is already fully parsed above, no second parse needed.
	isBlacklisted, err := uc.repo.Exists(ctx, "blacklist:"+jti)
	if err != nil {
		return nil, fmt.Errorf("failed to check blacklist status: %w", err)
	}
	if isBlacklisted {
		return nil, fmt.Errorf("token is blacklisted")
	}

	return &jwt.TokenInfo{
		UserID:    getStr("user_id"),
		Role:      getStr("role"),
		Type:      getStr("type"),
		IsActive:  getBool("is_active"),
		ExpiresAt: time.Unix(int64(getFloat("exp")), 0),
		IssuedAt:  time.Unix(int64(getFloat("iat")), 0),
		JTI:       jti,
	}, nil
}