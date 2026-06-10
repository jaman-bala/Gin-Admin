package middleware

import (
	"gin_auth_service/internal/application/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeaderKey = "Authorization"
	BearerSchema           = "Bearer "
	AccessTokenCookieName  = "access_token"
)

// AuthMiddleware validates the JWT token and sets user info in context.
func AuthMiddleware(authUseCase auth.UseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		if cookieToken, err := c.Cookie(AccessTokenCookieName); err == nil && cookieToken != "" {
			tokenString = cookieToken
		} else {
			authHeader := c.GetHeader(AuthorizationHeaderKey)
			if strings.HasPrefix(authHeader, BearerSchema) {
				tokenString = strings.TrimPrefix(authHeader, BearerSchema)
			}
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization token",
				"code":  "AUTH_TOKEN_MISSING",
			})
			c.Abort()
			return
		}

		user, err := authUseCase.GetUserInfoFromToken(c.Request.Context(), tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token or user not found",
				"code":    "AUTH_TOKEN_INVALID",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("id", user.ID)
		c.Set("user", user)
		c.Set("role", string(user.Role))

		c.Next()
	}
}
