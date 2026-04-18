package persistence

import (
	"context"
	"database/sql"
)

type txContextKey struct{}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// withTxContext 将事务句柄写入上下文，供 Repository 在事务内执行 SQL。
func withTxContext(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// executorFromContext 优先从上下文读取事务执行器，否则回退到 db。
func executorFromContext(ctx context.Context, db *sql.DB) sqlExecutor {
	if tx, ok := ctx.Value(txContextKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return db
}
