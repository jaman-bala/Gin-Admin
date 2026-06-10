package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Role levels mirror domain/user roles. Higher level = more privileges.
const (
	RoleLevelNone      = 0
	RoleLevelUser      = 1
	RoleLevelAdmin     = 2
	RoleLevelSuperuser = 3
)

func roleLevel(role string) int {
	switch role {
	case "superuser":
		return RoleLevelSuperuser
	case "admin":
		return RoleLevelAdmin
	case "user":
		return RoleLevelUser
	default:
		return RoleLevelNone
	}
}

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
