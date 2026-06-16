//go:build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNewPostgresPoolConnects requires a live database and runs only under the
// integration build tag: go test -tags integration ./internal/database
func TestNewPostgresPoolConnects(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := NewPostgresPool(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
