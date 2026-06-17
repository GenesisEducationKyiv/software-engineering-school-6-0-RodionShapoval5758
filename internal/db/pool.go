package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

type poolTxBeginner struct{ *pgxpool.Pool }

func (p *poolTxBeginner) BeginTx(ctx context.Context, opts pgx.TxOptions) (Tx, error) {
	return p.Pool.BeginTx(ctx, opts)
}

// WrapPool adapts *pgxpool.Pool to TxBeginner so it can be injected wherever a
// TxBeginner is required (services, relay) while keeping the pool concrete for
// callers that need Stats/Close directly.
func WrapPool(p *pgxpool.Pool) TxBeginner {
	return &poolTxBeginner{p}
}
