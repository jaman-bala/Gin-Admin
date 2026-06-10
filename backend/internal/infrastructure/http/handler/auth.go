package handler

import (
	"gin_auth_service/internal/application/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	usecase auth.UseCase
}

func NewAuthHandler(usecase auth.UseCase) *AuthHandler {
	return &AuthHandler{usecase: usecase}
}

// Login godoc
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param login body auth.LoginRequestDTO true "Login credentials"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req auth.LoginRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.usecase.Login(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Refresh godoc
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body auth.RefreshTokenRequest true "Refresh token"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req auth.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.usecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Logout godoc
// @Summary Logout user
// @Tags auth
// @Accept json
// @Security BearerAuth
// @Param body body auth.LogoutRequestDTO false "Optional refresh token to invalidate"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	accessToken := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
		return
	}

	var req auth.LogoutRequestDTO
	_ = c.ShouldBindJSON(&req) // optional body — ignore parse error

	if err := h.usecase.Logout(c.Request.Context(), accessToken, req.RefreshToken); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}