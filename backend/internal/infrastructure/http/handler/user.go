package handler

import (
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

// NewUserHandler creates a new instance of UserHandler.
func NewUserHandler(usecase user.UseCase) *UserHandler {
	return &UserHandler{
		usecase: usecase,
	}
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
	ctx := c.Request.Context()

	page := 1
	limit := 10
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
	search := c.Query("search")

	var isActive *bool
	if ia := c.Query("is_active"); ia != "" {
		v := ia == "true"
		isActive = &v
	}

	result, err := h.usecase.GetAll(ctx, page, limit, search, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx := c.Request.Context()
	u, err := h.usecase.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
	phone := c.Param("phone")
	ctx := c.Request.Context()
	u, err := h.usecase.GetByPhone(ctx, phone)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

// UpdateMe godoc
// @Summary Update own profile
// @Tags users
// @Security BearerAuth
// @Accept json,multipart/form-data
// @Produce json
// @Param first_name formData string false "First Name"
// @Param last_name formData string false "Last Name"
// @Param middle_name formData string false "Middle Name"
// @Param phone formData string false "Phone"
// @Param photo formData file false "Profile Photo"
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

	var req user.UserUpdateDTO
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var photoFile *user.FileUpload
	if file, err := c.FormFile("photo"); err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()
			photoFile = &user.FileUpload{
				Filename:    file.Filename,
				Size:        file.Size,
				ContentType: file.Header.Get("Content-Type"),
				File:        src,
			}
		}
	}

	ctx := c.Request.Context()
	u, err := h.usecase.Patch(ctx, id, req, photoFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, u)
}

// Patch godoc
// @Summary Update user
// @Tags users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "User ID"
// @Param first_name formData string false "First Name"
// @Param last_name formData string false "Last Name"
// @Param middle_name formData string false "Middle Name"
// @Param phone formData string false "Phone"
// @Param role formData string false "Role"
// @Param is_active formData bool false "Is Active"
// @Param photo formData file false "Profile Photo"
// @Router /api/v1/users/{id} [patch]
func (h *UserHandler) Patch(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req user.UserUpdateDTO
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var photoFile *user.FileUpload
	if file, err := c.FormFile("photo"); err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()
			photoFile = &user.FileUpload{
				Filename:    file.Filename,
				Size:        file.Size,
				ContentType: file.Header.Get("Content-Type"),
				File:        src,
			}
		}
	}

	ctx := c.Request.Context()
	u, err := h.usecase.Patch(ctx, id, req, photoFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Param first_name formData string true "First name"
// @Param last_name formData string true "Last name"
// @Param middle_name formData string false "Middle name"
// @Param phone formData string true "Phone number"
// @Param password formData string true "Password"
// @Param role formData string false "Role" Enums(user, admin, superuser) default(user)
// @Param photo formData file false "User photo"
// @Success 200 {object} user.UserResponseDTO
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req user.UserRequestDTO
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle photo upload
	var photoFile *user.FileUpload
	if file, err := c.FormFile("photo"); err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()
			photoFile = &user.FileUpload{
				Filename:    file.Filename,
				Size:        file.Size,
				ContentType: file.Header.Get("Content-Type"),
				File:        src, // ✅ fix: передаём io.Reader в MinIO
			}
		}
	}

	ctx := c.Request.Context()
	u, err := h.usecase.Create(ctx, req, photoFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set entity_id in context for AuditMiddleware
	c.Set("entity_id", u.ID)

	c.JSON(http.StatusOK, u)
}

// Delete godoc
// @Summary Delete user
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	ctx := c.Request.Context()
	if err := h.usecase.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
