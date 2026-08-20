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
)

// AuthMiddleware validates the JWT token and sets user info in context.
// Tokens are accepted from the Authorization header only: cookie-based auth
// would require CSRF protection, which this template intentionally avoids.
func AuthMiddleware(authUseCase auth.UseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if after, ok := strings.CutPrefix(authHeader, BearerSchema); ok {
			tokenString = after
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
			// Do not echo err.Error() to the client — it may contain
			// infrastructure details (Redis, parser internals).
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "AUTH_TOKEN_INVALID",
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
