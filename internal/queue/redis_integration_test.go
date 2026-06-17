//go:build integration

package queue

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNewRedisClientConnects requires a live Redis and runs only under the
// integration build tag: go test -tags integration ./internal/queue
func TestNewRedisClientConnects(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Fatalf("integration tag set but REDIS_ADDR is not configured; refusing to skip")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := NewRedisClient(ctx, addr, os.Getenv("REDIS_PASSWORD"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
