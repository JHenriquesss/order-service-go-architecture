package config

import "testing"

func TestLoadRequiresWorkerCountAtLeastOne(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://local"
		case "REDIS_ADDR":
			return "localhost:6379"
		case "JWT_SECRET":
			return "secret"
		case "ORDER_WORKER_COUNT":
			return "0"
		default:
			return ""
		}
	}
	if _, err := Load(getenv); err == nil {
		t.Fatal("expected error for zero workers")
	}
}

func TestLoadDefaultsWorkerCount(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://local"
		case "REDIS_ADDR":
			return "localhost:6379"
		case "JWT_SECRET":
			return "secret"
		default:
			return ""
		}
	}
	cfg, err := Load(getenv)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.OrderWorkerCount != 3 {
		t.Fatalf("workers %d", cfg.OrderWorkerCount)
	}
}
