package app

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
)

type dbPinger struct{ pool *pgxpool.Pool }

func (d *dbPinger) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

type natsPinger struct{ nc *natsgo.Conn }

func (n *natsPinger) Ping(_ context.Context) error {
	if n.nc.Status() != natsgo.CONNECTED {
		return errors.New("not connected")
	}
	return nil
}
