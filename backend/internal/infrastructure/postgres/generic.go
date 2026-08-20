package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// base provides shared query helpers for concrete repositories.
// Repositories embed it to get access to the db handle and to the
// generic pagination helper below.
type base struct {
	db *sqlx.DB
}

// q returns the active transaction from ctx when running inside
// TxManager.WithinTx, otherwise the pooled DB handle.
func (b base) q(ctx context.Context) sqlx.ExtContext {
	if dbtx, ok := txFromContext(ctx); ok {
		return dbtx
	}
	return b.db
}

// selectPage runs a COUNT query and a LIMIT/OFFSET data query over the same
// table and WHERE clause, returning the rows and the total match count.
//
// Row is the db-tagged struct to scan rows into and must be instantiated
// explicitly at the call site (e.g. r.selectPage[UserDB](...)), since it only
// appears in the return type. This is a Go 1.27 generic method: the type
// parameter belongs to the method itself, not to the receiver.
//
// where must be either empty or start with " WHERE" and use $1..$n
// placeholders matching args; LIMIT and OFFSET are appended as the next two
// placeholders.
func (b base) selectPage[Row any](ctx context.Context, table, where, orderBy string, page, limit int, args ...any) ([]Row, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	q := b.q(ctx)

	var total int64
	if err := sqlx.GetContext(ctx, q, &total, "SELECT COUNT(*) FROM "+table+where, args...); err != nil {
		return nil, 0, err
	}

	n := len(args)
	dataQuery := fmt.Sprintf(
		"SELECT * FROM %s%s ORDER BY %s LIMIT $%d OFFSET $%d",
		table, where, orderBy, n+1, n+2,
	)
	var rows []Row
	if err := sqlx.SelectContext(ctx, q, &rows, dataQuery, append(args, limit, (page-1)*limit)...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// mapSlice converts a slice of DB rows into domain values, typically using a
// method expression like (*AuditLogDB).ToEntity as fn.
func mapSlice[S, D any](src []S, fn func(*S) D) []D {
	out := make([]D, 0, len(src))
	for i := range src {
		out = append(out, fn(&src[i]))
	}
	return out
}
