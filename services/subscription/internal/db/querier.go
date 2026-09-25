package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx, allowing store methods
// to run against either a pool connection or an in-progress transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Tx is a subset of pgx.Tx covering only what our code uses: DBTX operations
// plus explicit Commit and Rollback. pgx.Tx satisfies this interface.
type Tx interface {
	DBTX
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TxBeginner can start a transaction that returns a Tx. *pgxpool.Pool satisfies
// this via WrapPool; tests can inject a lightweight fake.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (Tx, error)
}
