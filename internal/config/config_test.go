package config

import "testing"

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func completeEnv() map[string]string {
	return map[string]string{
		"APP_ENV":                "production",
		"HTTP_PORT":              "9090",
		"DATABASE_URL":           "postgres://orders:orders@localhost:5432/orders?sslmode=disable",
		"REDIS_ADDR":             "localhost:6379",
		"REDIS_PASSWORD":         "secret",
		"JWT_SECRET":             "a-secret",
		"JWT_EXPIRATION_MINUTES": "60",
		"ORDER_WORKER_COUNT":     "5",
		"ORDER_MAX_RETRIES":      "2",
		"LOG_LEVEL":              "debug",
	}
}

func TestLoadParsesCompleteEnv(t *testing.T) {
	cfg, err := Load(envFrom(completeEnv()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != "production" || cfg.HTTPPort != "9090" {
		t.Errorf("app meta not parsed: %+v", cfg)
	}
	if cfg.JWTExpirationMinutes != 60 || cfg.OrderWorkerCount != 5 || cfg.OrderMaxRetries != 2 {
		t.Errorf("numeric fields not parsed: %+v", cfg)
	}
	if cfg.LogLevel != "debug" || cfg.RedisPassword != "secret" {
		t.Errorf("string fields not parsed: %+v", cfg)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	env := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"REDIS_ADDR":   "localhost:6379",
		"JWT_SECRET":   "s",
	}
	cfg, err := Load(envFrom(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != "development" || cfg.HTTPPort != "8080" {
		t.Errorf("defaults not applied: %+v", cfg)
	}
	if cfg.JWTExpirationMinutes != 120 || cfg.OrderWorkerCount != 3 || cfg.OrderMaxRetries != 3 || cfg.LogLevel != "info" {
		t.Errorf("numeric/level defaults not applied: %+v", cfg)
	}
}

func TestLoadErrorsOnMissingRequired(t *testing.T) {
	for _, missing := range []string{"DATABASE_URL", "REDIS_ADDR", "JWT_SECRET"} {
		env := completeEnv()
		env[missing] = ""
		if _, err := Load(envFrom(env)); err == nil {
			t.Errorf("expected error when %s missing", missing)
		}
	}
}

func TestLoadErrorsOnInvalidInt(t *testing.T) {
	env := completeEnv()
	env["ORDER_WORKER_COUNT"] = "notanumber"
	if _, err := Load(envFrom(env)); err == nil {
		t.Error("expected error on non-integer ORDER_WORKER_COUNT")
	}
}

func TestLoadErrorsOnInvalidLogLevel(t *testing.T) {
	env := completeEnv()
	env["LOG_LEVEL"] = "verbose"
	if _, err := Load(envFrom(env)); err == nil {
		t.Error("expected error on invalid LOG_LEVEL")
	}
}

func TestLoadRejectsNegativeOrderMaxRetries(t *testing.T) {
	env := completeEnv()
	env["ORDER_MAX_RETRIES"] = "-1"
	if _, err := Load(envFrom(env)); err == nil {
		t.Error("expected error on negative ORDER_MAX_RETRIES")
	}
}
