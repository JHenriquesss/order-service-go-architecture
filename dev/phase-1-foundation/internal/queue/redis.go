// Package queue provides the Redis connection behind a small interface. This
// phase wires the connection only; producer logic arrives in Phase 5.
package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Client is the minimal queue port used at composition time.
type Client interface {
	Ping(ctx context.Context) error
	Close() error
}

type redisClient struct {
	inner *redis.Client
}

func (c *redisClient) Ping(ctx context.Context) error { return c.inner.Ping(ctx).Err() }
func (c *redisClient) Close() error                   { return c.inner.Close() }

// NewRedisClient connects to Redis and verifies connectivity with a ping. A
// failed ping closes the client before returning.
func NewRedisClient(ctx context.Context, addr, password string) (Client, error) {
	inner := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	if err := inner.Ping(ctx).Err(); err != nil {
		_ = inner.Close()
		return nil, fmt.Errorf("queue: ping redis: %w", err)
	}
	return &redisClient{inner: inner}, nil
}
