// Package database provides the PostgreSQL connection pool used by the
// repositories at composition time.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool creates a pgx connection pool from a DSN and verifies
// connectivity with a ping. A failed ping closes the pool before returning.
// The concrete *pgxpool.Pool is returned so repositories can use queries and
// transactions; callers still only need Close() for lifecycle.
func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
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
