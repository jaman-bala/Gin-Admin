package token

import (
	"context"
	"time"
)

// Repository defines the interface for token storage (usually cache).
// Get is deliberately absent: token lookups only ever check membership
// (Exists, for blacklisting) or write (Set); nothing reads a stored token
// value back, so that method isn't part of this contract.
type Repository interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
