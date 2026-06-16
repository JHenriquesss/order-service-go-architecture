package config

import "testing"

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadReadsSecretAndExpiry(t *testing.T) {
	cfg, err := Load(env(map[string]string{"JWT_SECRET": "abc", "JWT_EXPIRATION_MINUTES": "30"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWTSecret != "abc" || cfg.JWTExpirationMinutes != 30 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadDefaultsExpiry(t *testing.T) {
	cfg, err := Load(env(map[string]string{"JWT_SECRET": "abc"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWTExpirationMinutes != 120 {
		t.Fatalf("default expiry = %d, want 120", cfg.JWTExpirationMinutes)
	}
}

func TestLoadRequiresSecret(t *testing.T) {
	if _, err := Load(env(map[string]string{})); err == nil {
		t.Fatal("missing JWT_SECRET must error")
	}
}

func TestLoadRejectsBadExpiry(t *testing.T) {
	for _, v := range []string{"abc", "0", "-5"} {
		_, err := Load(env(map[string]string{"JWT_SECRET": "abc", "JWT_EXPIRATION_MINUTES": v}))
		if err == nil {
			t.Fatalf("JWT_EXPIRATION_MINUTES=%q must error", v)
		}
	}
}
