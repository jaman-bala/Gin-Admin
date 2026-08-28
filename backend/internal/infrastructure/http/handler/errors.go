package handler

import (
	"errors"
	"log/slog"
	"net/http"

	appErrors "gin_auth_service/pkg/errors"

	"github.com/gin-gonic/gin"
)

// respondError maps a domain error to the appropriate HTTP status and aborts the chain.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appErrors.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, appErrors.ErrConflict),
		errors.Is(err, appErrors.ErrPhoneAlreadyExists),
		errors.Is(err, appErrors.ErrStaleWrite):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, appErrors.ErrInvalidCredentials),
		errors.Is(err, appErrors.ErrInvalidToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, appErrors.ErrAccountBlocked):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, appErrors.ErrInvalidUUID),
		errors.Is(err, appErrors.ErrInvalidPatchField):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		slog.Error("unhandled use case error", "error", err, "path", c.Request.URL.Path, "method", c.Request.Method)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
