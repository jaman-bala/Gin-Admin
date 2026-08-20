// Package tx defines the transaction boundary abstraction for usecases.
package tx

import "context"

// Manager runs fn atomically: every repository operation performed with the
// ctx passed into fn is committed together or rolled back on error. Usecases
// depend on this interface, not on the database layer.
type Manager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
