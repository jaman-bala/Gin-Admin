package postgres

import (
	"context"
	"fmt"

	"gin_auth_service/internal/domain/tx"

	"github.com/jmoiron/sqlx"
)

type txCtxKey struct{}

// TxManager implements tx.Manager on top of sqlx. The opened transaction
// travels inside the context; repositories pick it up automatically via
// base.q, so several repository calls compose into one atomic unit without
// the usecase layer knowing anything about sqlx.
type TxManager struct {
	db *sqlx.DB
}

func NewTxManager(db *sqlx.DB) *TxManager { return &TxManager{db: db} }

func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	dbtx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = dbtx.Rollback()
			panic(p)
		}
	}()

	if err := fn(context.WithValue(ctx, txCtxKey{}, dbtx)); err != nil {
		_ = dbtx.Rollback()
		return err
	}
	return dbtx.Commit()
}

// txFromContext extracts the active transaction, if any.
func txFromContext(ctx context.Context) (*sqlx.Tx, bool) {
	dbtx, ok := ctx.Value(txCtxKey{}).(*sqlx.Tx)
	return dbtx, ok
}

var _ tx.Manager = (*TxManager)(nil)
