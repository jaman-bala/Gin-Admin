package user

import (
	"gin_auth_service/internal/pkg/hash"
	"time"

	"github.com/google/uuid"
)

// Role defines the role of a user.
type Role string

const (
	RoleSuperuser Role = "superuser"
	RoleAdmin     Role = "admin"
	RoleUser      Role = "user"
)

// User represents a user entity in the system.
type User struct {
	ID         uuid.UUID
	FirstName  string
	LastName   string
	MiddleName string
	Phone      string
	Password   string
	Role       Role
	Photo      string

	IsActive bool

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// CheckPassword checks if the password matches.
func (u *User) CheckPassword(password string) error {
	return hash.CheckPassword(u.Password, password)
}
