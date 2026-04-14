package middleware

import (
	"gin_auth_service/internal/application/token"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenBlacklistMiddleware checks if the token is blacklisted.
func TokenBlacklistMiddleware(service token.UseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		isBlacklisted, err := service.IsTokenBlacklisted(c.Request.Context(), tokenString)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check token status"})
			c.Abort()
			return
		}

		if isBlacklisted {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is blacklisted"})
			c.Abort()
			return
		}

		c.Next()
	}
}
