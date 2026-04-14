package user

import (
	"gin_auth_service/internal/domain/user"
	"io"
	"time"

	"github.com/google/uuid"
)

// UserDTO represents a complete user data transfer object.
type UserDTO struct {
	ID         uuid.UUID  `json:"id"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	MiddleName string     `json:"middle_name"`
	Phone      string     `json:"phone" validate:"required,e164" example:"+996500500500"`
	Role       user.Role  `json:"role" default:"user"`
	Photo      string     `json:"photo" validate:"omitempty,url"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

// UserListResponse is the paginated response for the user list endpoint.
type UserListResponse struct {
	Users []*UserResponseDTO `json:"users"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// UserRequestDTO represents a request to create a new user.
type UserRequestDTO struct {
	FirstName  string    `json:"first_name" form:"first_name"`
	LastName   string    `json:"last_name" form:"last_name"`
	MiddleName string    `json:"middle_name" form:"middle_name"`
	Password   string    `json:"password" validate:"required,min=8" form:"password" example:"Password123"`
	Phone      string    `json:"phone" validate:"required,e164" form:"phone" example:"+996500500500"`
	Role       user.Role `json:"role" form:"role" default:"user"`
	Photo      string    `json:"photo" validate:"omitempty,url"`
}

// UserDashboardDTO represents user data for the dashboard.
type UserDashboardDTO struct {
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	MiddleName string    `json:"middle_name"`
	Password   string    `json:"password" validate:"required,min=8" example:"Password123"`
	Phone      string    `json:"phone" validate:"required,e164" example:"+996500500500"`
	Role       user.Role `json:"role" default:"user"`
	Photo      string    `json:"photo" validate:"omitempty,url"`
}

// UserResponseDTO represents a user in a response.
type UserResponseDTO struct {
	ID         uuid.UUID  `json:"id"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	MiddleName string     `json:"middle_name"`
	Phone      string     `json:"phone"`
	Role       user.Role  `json:"role"`
	Photo      string     `json:"photo"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

// UserUpdateDTO represents a request to update a user.
type UserUpdateDTO struct {
	FirstName  *string    `json:"first_name" form:"first_name"`
	LastName   *string    `json:"last_name" form:"last_name"`
	MiddleName *string    `json:"middle_name" form:"middle_name"`
	Phone      *string    `json:"phone" validate:"omitempty,e164" form:"phone" example:"+996500500500"`
	Password   *string    `json:"password" form:"password"`
	Role       *string    `json:"role" form:"role"`
	Photo      *string    `json:"photo"`
	IsActive   *bool      `json:"is_active" form:"is_active"`
}

// FileUpload represents a file uploaded by the user.
type FileUpload struct {
	Filename    string
	Size        int64
	ContentType string
	File        io.Reader
}

// FromModel maps a user entity to a response DTO.
func (dto *UserResponseDTO) FromModel(u *user.User) {
	dto.ID = u.ID
	dto.FirstName = u.FirstName
	dto.LastName = u.LastName
	dto.MiddleName = u.MiddleName
	dto.Phone = u.Phone
	dto.Role = u.Role
	dto.Photo = u.Photo
	dto.IsActive = u.IsActive
	dto.CreatedAt = u.CreatedAt
	dto.UpdatedAt = u.UpdatedAt
	dto.DeletedAt = u.DeletedAt
}

// ToModel converts a dashboard DTO to a user entity.
func (dto *UserDashboardDTO) ToModel() *user.User {
	u := &user.User{
		FirstName:  dto.FirstName,
		LastName:   dto.LastName,
		MiddleName: dto.MiddleName,
		Password:   dto.Password,
		Phone:      dto.Phone,
		Role:       dto.Role,
		Photo:      dto.Photo,
		IsActive:   true,
	}
	if u.Role == "" {
		u.Role = user.RoleUser
	}
	return u
}

// ApplyToModel applies updates from a DTO to an existing user entity.
func (dto *UserUpdateDTO) ApplyToModel(u *user.User) {
	if dto.FirstName != nil {
		u.FirstName = *dto.FirstName
	}
	if dto.LastName != nil {
		u.LastName = *dto.LastName
	}
	if dto.MiddleName != nil {
		u.MiddleName = *dto.MiddleName
	}
	if dto.Phone != nil {
		u.Phone = *dto.Phone
	}
	if dto.Role != nil {
		u.Role = user.Role(*dto.Role)
	}
	if dto.Photo != nil {
		u.Photo = *dto.Photo
	}
	if dto.IsActive != nil {
		u.IsActive = *dto.IsActive
	}
}
