package user

import (
	"gin_auth_service/internal/domain/user"
	"time"

	"uuid"
)

// UserListResponse is the paginated response for the user list endpoint.
type UserListResponse struct {
	Users []*UserResponseDTO `json:"users"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// UserRequestDTO represents a request to create a new user.
// Role is a plain string, not user.Role: swag treats a field typed with a
// named enum type plus its own "oneof" validator as two separate enum
// sources on the same property (an allOf $ref alongside a sibling `enum:`),
// which the generated OpenAPI spec carries fine but the orval/Zod codegen
// cannot — it emits a chained, invalid `.enum(...).enum(...)`. A plain
// string + validate:"oneof=..." (matching UserUpdateDTO.Role below) gives a
// single clean enum in the spec and keeps the generated frontend client
// buildable.
type UserRequestDTO struct {
	FirstName  string `json:"first_name" validate:"required"`
	LastName   string `json:"last_name"  validate:"required"`
	MiddleName string `json:"middle_name"`
	Password   string `json:"password" validate:"required,strong_password" example:"Password123"`
	Phone      string `json:"phone"    validate:"required,e164"             example:"+996500500500"`
	Telegram   string `json:"telegram"`
	Role       string `json:"role" validate:"required,oneof=user admin superuser"`
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
	Telegram   string     `json:"telegram"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

// UserUpdateDTO represents a request to update a user.
type UserUpdateDTO struct {
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	MiddleName *string `json:"middle_name"`
	Phone      *string `json:"phone"     validate:"omitempty,e164" example:"+996500500500"`
	Password   *string `json:"password"  validate:"omitempty,strong_password"`
	Role       *string `json:"role"      validate:"omitempty,oneof=user admin superuser"`
	Telegram   *string `json:"telegram"`
	IsActive   *bool   `json:"is_active"`
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
	dto.Telegram = u.Telegram
	dto.IsActive = u.IsActive
	dto.CreatedAt = u.CreatedAt
	dto.UpdatedAt = u.UpdatedAt
	dto.DeletedAt = u.DeletedAt
}

// ApplyToModel applies non-nil fields from the DTO to an existing user entity.
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
	if dto.Telegram != nil {
		u.Telegram = *dto.Telegram
	}
	if dto.IsActive != nil {
		u.IsActive = *dto.IsActive
	}
}

// UserSelfUpdateDTO is the subset of fields a user may change on their own profile.
// Role and IsActive are intentionally absent to prevent privilege escalation.
// Changing the password requires CurrentPassword: a stolen access token alone
// must not be enough to take over the account permanently.
type UserSelfUpdateDTO struct {
	FirstName       *string `json:"first_name"`
	LastName        *string `json:"last_name"`
	MiddleName      *string `json:"middle_name"`
	Phone           *string `json:"phone"            validate:"omitempty,e164"            example:"+996500500500"`
	Password        *string `json:"password"         validate:"omitempty,strong_password"`
	CurrentPassword *string `json:"current_password"`
	Telegram        *string `json:"telegram"`
}

func (dto *UserSelfUpdateDTO) ApplyToModel(u *user.User) {
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
	if dto.Telegram != nil {
		u.Telegram = *dto.Telegram
	}
}
