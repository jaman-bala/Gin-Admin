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

// Privilege levels: higher means more privileges. Single source of truth for
// both the HTTP role middleware and the use-case authorization rules.
const (
	LevelNone      = 0
	LevelUser      = 1
	LevelAdmin     = 2
	LevelSuperuser = 3
)

// Level returns the privilege level of the role; unknown roles get LevelNone
// so a corrupted or forged role value never grants anything.
func (r Role) Level() int {
	switch r {
	case RoleSuperuser:
		return LevelSuperuser
	case RoleAdmin:
		return LevelAdmin
	case RoleUser:
		return LevelUser
	default:
		return LevelNone
	}
}

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
