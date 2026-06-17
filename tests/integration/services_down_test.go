//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIntegrationFailsWhenPostgresUnreachable(t *testing.T) {
	dsn := requirePostgresEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	badDSN := dsn + "_invalid_host"
	pool, err := pgxpool.New(ctx, badDSN)
	if err == nil {
		pool.Close()
		t.Fatal("expected connection error for invalid DSN")
	}
}
