package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Role levels for hierarchical permissions
const (
	RoleLevelNone      = 0
	RoleLevelUser      = 1
	RoleLevelManager   = 2
	RoleLevelAdmin     = 3
	RoleLevelSuperuser = 4
)

// RequireRoleMiddleware checks if the user has one of the specified roles.
func RequireRoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Роль пользователя не определена",
				"code":  "AUTH_ROLE_MISSING",
			})
			return
		}

		userRole := fmt.Sprintf("%v", roleValue)

		for _, requiredRole := range roles {
			if userRole == requiredRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":          "Недостаточно прав для выполнения операции",
			"code":           "AUTH_INSUFFICIENT_PRIVILEGES",
			"required_roles": roles,
			"user_role":      userRole,
		})
		c.Abort()
	}
}

// RequireRoleLevelMiddleware checks if the user has a role with at least minLevel.
func RequireRoleLevelMiddleware(minLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Роль пользователя не определена",
				"code":  "AUTH_ROLE_MISSING",
			})
			return
		}

		roleStr := fmt.Sprintf("%v", roleValue)

		var userLevel int
		switch roleStr {
		case "superuser":
			userLevel = RoleLevelSuperuser
		case "admin":
			userLevel = RoleLevelAdmin
		case "manager":
			userLevel = RoleLevelManager
		case "user":
			userLevel = RoleLevelUser
		default:
			userLevel = RoleLevelNone
		}

		if userLevel < minLevel {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Недостаточно прав для выполнения операции",
				"code":           "AUTH_INSUFFICIENT_PRIVILEGES",
				"required_level": minLevel,
				"user_level":     userLevel,
				"user_role":      roleStr,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions for common role checks using RequireRoleLevelMiddleware
func AdminRoleMiddleware() gin.HandlerFunc {
	return RequireRoleLevelMiddleware(RoleLevelAdmin)
}

func SuperUserRoleMiddleware() gin.HandlerFunc {
	return RequireRoleLevelMiddleware(RoleLevelSuperuser)
}

func ManagerRoleMiddleware() gin.HandlerFunc {
	return RequireRoleLevelMiddleware(RoleLevelManager)
}
