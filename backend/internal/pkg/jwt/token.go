// Package jwt provides JWT token management services.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTService defines the interface for working with JWT tokens.
// revive:disable:stutter
type JWTService interface {
	// Создает новый токен
	CreateToken(claims MapClaims) (string, error)
	// Парсит и валидирует токен
	ParseToken(tokenString string) (*jwt.Token, error)
	// Валидирует токен
	ValidateToken(token *jwt.Token) error
}

// MapClaims тип для claims токена
type MapClaims jwt.MapClaims

// TokenInfo информация о токене
type TokenInfo struct {
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Type      string    `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	JTI       string    `json:"jti"`
}

// jwtService реализация JWT сервиса
type jwtService struct {
	secretKey string
}

// NewJWTService создает новый JWT сервис
func NewJWTService(secretKey string) JWTService {
	return &jwtService{
		secretKey: secretKey,
	}
}

// CreateToken создает новый JWT токен
func (s *jwtService) CreateToken(claims MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))

	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ParseToken парсит и валидирует JWT токен
func (s *jwtService) ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем алгоритм подписи
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

// ValidateToken валидирует JWT токен
func (s *jwtService) ValidateToken(token *jwt.Token) error {
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}

	// Проверяем, что токен не истек
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return fmt.Errorf("token expired")
			}
		}
	}

	return nil
}
