// Package config loads and validates application configuration from the
// environment. It holds no global state; callers pass a getenv function.
package config

import (
	"fmt"
	"strconv"
)

// Config holds the validated application configuration (architecture §26).
type Config struct {
	AppEnv                string
	HTTPPort              string
	DatabaseURL           string
	RedisAddr             string
	RedisPassword         string
	JWTSecret             string
	JWTExpirationMinutes  int
	OrderWorkerCount      int
	OrderMaxRetries       int
	TransientFailureTotal string
	MetricsPort           string
	LogLevel              string
}

// Load reads configuration using getenv (e.g. os.Getenv). Required variables
// must be present and non-empty; numeric and LOG_LEVEL values are validated.
func Load(getenv func(string) string) (*Config, error) {
	cfg := &Config{
		AppEnv:        valueOr(getenv, "APP_ENV", "development"),
		HTTPPort:      valueOr(getenv, "HTTP_PORT", "8080"),
		MetricsPort:   valueOr(getenv, "METRICS_PORT", "9090"),
		DatabaseURL:   getenv("DATABASE_URL"),
		RedisAddr:     getenv("REDIS_ADDR"),
		RedisPassword: getenv("REDIS_PASSWORD"),
		JWTSecret:     getenv("JWT_SECRET"),
		LogLevel:      valueOr(getenv, "LOG_LEVEL", "info"),
	}

	required := []struct{ name, value string }{
		{"DATABASE_URL", cfg.DatabaseURL},
		{"REDIS_ADDR", cfg.RedisAddr},
		{"JWT_SECRET", cfg.JWTSecret},
	}
	for _, r := range required {
		if r.value == "" {
			return nil, fmt.Errorf("config: required env var %s is missing", r.name)
		}
	}

	jwtExp, err := intOr(getenv, "JWT_EXPIRATION_MINUTES", 120)
	if err != nil {
		return nil, err
	}
	cfg.JWTExpirationMinutes = jwtExp

	workers, err := intOr(getenv, "ORDER_WORKER_COUNT", 3)
	if err != nil {
		return nil, err
	}
	cfg.OrderWorkerCount = workers

	maxRetries, err := intOr(getenv, "ORDER_MAX_RETRIES", 3)
	if err != nil {
		return nil, err
	}
	if maxRetries < 0 {
		return nil, fmt.Errorf("config: ORDER_MAX_RETRIES must be >= 0")
	}
	cfg.OrderMaxRetries = maxRetries
	cfg.TransientFailureTotal = getenv("ORDER_TRANSIENT_FAILURE_TOTAL")

	if !validLogLevel(cfg.LogLevel) {
		return nil, fmt.Errorf("config: invalid LOG_LEVEL %q", cfg.LogLevel)
	}

	return cfg, nil
}

func valueOr(getenv func(string) string, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return fallback
}

func intOr(getenv func(string) string, key string, fallback int) (int, error) {
	raw := getenv(key)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", key, err)
	}
	return n, nil
}

func validLogLevel(level string) bool {
	switch level {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
