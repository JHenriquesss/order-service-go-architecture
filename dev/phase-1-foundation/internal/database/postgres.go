// Package database provides the PostgreSQL connection pool behind a small
// interface so callers depend on a port, not on pgx directly.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the minimal database port used at composition time. *pgxpool.Pool
// satisfies it; repositories (later phases) depend on richer interfaces.
type Pool interface {
	Ping(ctx context.Context) error
	Close()
}

// NewPostgresPool creates a pgx connection pool from a DSN and verifies
// connectivity with a ping. A failed ping closes the pool before returning.
func NewPostgresPool(ctx context.Context, dsn string) (Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("database: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	return pool, nil
}
