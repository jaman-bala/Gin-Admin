// Package jwt provides JWT token management services.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService defines the interface for working with JWT tokens.
// revive:disable:stutter
type JWTService interface {
	CreateToken(claims MapClaims) (string, error)
	ParseToken(tokenString string) (*jwt.Token, error)
}

// MapClaims is an alias for jwt.MapClaims.
type MapClaims jwt.MapClaims

// TokenInfo holds decoded claims from a validated JWT.
type TokenInfo struct {
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Type      string    `json:"type"`
	IsActive  bool      `json:"is_active"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	JTI       string    `json:"jti"`
}

type jwtService struct {
	secretKey string
}

func NewJWTService(secretKey string) JWTService {
	return &jwtService{secretKey: secretKey}
}

func (s *jwtService) CreateToken(claims MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

func (s *jwtService) ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	return token, nil
}
