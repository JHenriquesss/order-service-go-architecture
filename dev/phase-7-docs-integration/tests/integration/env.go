//go:build integration

package integration

import (
	"fmt"
	"os"
	"testing"
)

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("integration tag set but %s is not configured; refusing to skip", key)
	}
	return v
}

func requirePostgresEnv(t *testing.T) string {
	t.Helper()
	return requireEnv(t, "DATABASE_URL")
}

func requireRedisEnv(t *testing.T) string {
	t.Helper()
	return requireEnv(t, "REDIS_ADDR")
}

func requireAPIEnv(t *testing.T) string {
	t.Helper()
	apiBase := os.Getenv("API_BASE_URL")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	return apiBase
}

func requireE2EEnv(t *testing.T) (databaseURL, redisAddr, apiBase string) {
	t.Helper()
	databaseURL = requireEnv(t, "DATABASE_URL")
	redisAddr = requireEnv(t, "REDIS_ADDR")
	apiBase = requireAPIEnv(t)
	return databaseURL, redisAddr, apiBase
}

func uniqueSuffix() string {
	return fmt.Sprintf("%d", os.Getpid())
}
