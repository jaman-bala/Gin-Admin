package user

import (
	"gin_auth_service/internal/pkg/hash"
	"time"

	"uuid"
)

// Role defines the role of a user.
type Role string

const (
	RoleSuperuser Role = "superuser"
	RoleAdmin     Role = "admin"
	RoleUser      Role = "user"
)

type PatchData map[string]any

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
	Telegram   string

	IsActive bool

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	// Version is the optimistic-lock counter. Repository.Patch checks it in
	// the WHERE clause and rejects the write if it no longer matches — set
	// from whatever GetID/FindByPhone/GetWithFilters last read, never by
	// callers.
	Version int
}

// CheckPassword checks if the password matches.
func (u *User) CheckPassword(password string) error {
	return hash.CheckPassword(u.Password, password)
}
