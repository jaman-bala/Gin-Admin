package user

import (
	"context"

	"uuid"
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
	PatchFields(ctx context.Context, id uuid.UUID, fields PatchData) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByPhone(ctx context.Context, phone string) (*User, error)
	// LockActiveSuperusers locks every active superuser row until the
	// surrounding transaction ends and returns how many there are. It must be
	// called inside tx.Manager.WithinTx; concurrent callers serialize on it,
	// which is what keeps "never remove the last superuser" race-free.
	LockActiveSuperusers(ctx context.Context) (int, error)
}
