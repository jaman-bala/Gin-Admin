package user

import (
	"context"

	"github.com/google/uuid"
)

// FilterParams holds pagination and filter options for user queries.
type FilterParams struct {
	Page     int
	Limit    int
	Search   string
	IsActive *bool
}

// Repository defines the interface for user persistence.
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetWithFilters(ctx context.Context, params FilterParams) ([]*User, int, error)
	GetID(ctx context.Context, id uuid.UUID) (*User, error)
	Patch(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByPhone(ctx context.Context, phone string) (*User, error)
}
