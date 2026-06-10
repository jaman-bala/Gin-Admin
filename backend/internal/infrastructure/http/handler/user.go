package handler

import (
	"gin_auth_service/internal/application/file"
	"gin_auth_service/internal/application/user"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	usecase user.UseCase
}

func NewUserHandler(usecase user.UseCase) *UserHandler {
	return &UserHandler{usecase: usecase}
}

// GetMe godoc
// @Summary Get current user profile
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} user.UserResponseDTO
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	uid, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	id, ok := uid.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}
	dto, err := h.usecase.GetMe(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

// GetAll godoc
// @Summary Get all users with pagination
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by name or phone"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {object} user.UserListResponse
// @Router /api/v1/users [get]
func (h *UserHandler) GetAll(c *gin.Context) {
	page, limit := 1, 10
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	var isActive *bool
	if ia := c.Query("is_active"); ia != "" {
		v := ia == "true"
		isActive = &v
	}

	result, err := h.usecase.GetAll(c.Request.Context(), page, limit, c.Query("search"), isActive)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary Get user by ID
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Produce json
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	u, err := h.usecase.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// GetByPhone godoc
// @Summary Get user by phone
// @Tags users
// @Security BearerAuth
// @Param phone path string true "User phone"
// @Produce json
// @Router /api/v1/users/phone/{phone} [get]
func (h *UserHandler) GetByPhone(c *gin.Context) {
	u, err := h.usecase.GetByPhone(c.Request.Context(), c.Param("phone"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// UpdateMe godoc
// @Summary Update own profile
// @Tags users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Router /api/v1/users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	uid, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	id, ok := uid.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	var req user.UserSelfUpdateDTO
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.usecase.PatchSelf(c.Request.Context(), id, req, extractPhoto(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// Patch godoc
// @Summary Update user (admin only)
// @Tags users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "User ID"
// @Router /api/v1/users/{id} [patch]
func (h *UserHandler) Patch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req user.UserUpdateDTO
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.usecase.Patch(c.Request.Context(), id, req, extractPhoto(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

// Create godoc
// @Summary Create user (admin only)
// @Tags users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req user.UserRequestDTO
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.usecase.Create(c.Request.Context(), req, extractPhoto(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, u)
}

// Delete godoc
// @Summary Delete user (admin only)
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	if err := h.usecase.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// extractPhoto reads an uploaded photo from the multipart form, if present.
// Returns nil when the field is absent, has an empty filename, or has zero size
// (browsers submit empty file inputs as a part with an empty filename).
func extractPhoto(c *gin.Context) *file.FileUpload {
	f, err := c.FormFile("photo")
	if err != nil || f == nil || f.Size == 0 || f.Filename == "" {
		return nil
	}
	src, err := f.Open()
	if err != nil {
		return nil
	}
	return &file.FileUpload{
		Filename:    f.Filename,
		Size:        f.Size,
		ContentType: f.Header.Get("Content-Type"),
		File:        src,
	}
}