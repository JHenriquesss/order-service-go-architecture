DATABASE_URL ?= postgres://orders:orders@localhost:5432/orders?sslmode=disable
REDIS_ADDR ?= localhost:6379
API_BASE_URL ?= http://localhost:8080

.PHONY: run-api run-worker test test-cover compose-up compose-down migrate-up migrate-down seed test-integration

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./...

test-cover:
	go test ./... -cover

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

seed: migrate-up
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed

test-integration:
	DATABASE_URL="$(DATABASE_URL)" REDIS_ADDR="$(REDIS_ADDR)" API_BASE_URL="$(API_BASE_URL)" go test -tags=integration ./...
