package middleware

import (
	"fmt"
	"net/http"

	domainUser "gin_auth_service/internal/domain/user"

	"github.com/gin-gonic/gin"
)

// Role levels come from domain/user so the middleware and the use-case
// authorization rules can never disagree. Higher level = more privileges.
const (
	RoleLevelNone      = domainUser.LevelNone
	RoleLevelUser      = domainUser.LevelUser
	RoleLevelAdmin     = domainUser.LevelAdmin
	RoleLevelSuperuser = domainUser.LevelSuperuser
)

func roleLevel(role string) int { return domainUser.Role(role).Level() }

// RequireRoleLevelMiddleware rejects requests whose role level is below minLevel.
func RequireRoleLevelMiddleware(minLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "User role not defined",
				"code":  "AUTH_ROLE_MISSING",
			})
			return
		}

		if roleLevel(fmt.Sprintf("%v", roleValue)) < minLevel {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient permissions",
				"code":           "AUTH_INSUFFICIENT_PRIVILEGES",
				"required_level": minLevel,
				"user_role":      roleValue,
			})
			return
		}

		c.Next()
	}
}
