// Package config loads the minimal authentication configuration this phase
// needs from the environment. It holds no global state; callers pass a getenv
// function so tests can supply values without touching the real environment.
package config

import (
	"fmt"
	"strconv"
)

// Config holds the validated auth configuration (architecture §26). JWT secret
// and expiry are read here and never hardcoded elsewhere.
type Config struct {
	JWTSecret            string
	JWTExpirationMinutes int
}

// Load reads configuration using getenv (e.g. os.Getenv). JWT_SECRET is
// required and must be non-empty; JWT_EXPIRATION_MINUTES defaults to 120 and
// must be a positive integer when provided.
func Load(getenv func(string) string) (*Config, error) {
	secret := getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("config: required env var JWT_SECRET is missing")
	}

	exp := 120
	if raw := getenv("JWT_EXPIRATION_MINUTES"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("config: JWT_EXPIRATION_MINUTES must be an integer: %w", err)
		}
		if n <= 0 {
			return nil, fmt.Errorf("config: JWT_EXPIRATION_MINUTES must be positive, got %d", n)
		}
		exp = n
	}

	return &Config{JWTSecret: secret, JWTExpirationMinutes: exp}, nil
}
