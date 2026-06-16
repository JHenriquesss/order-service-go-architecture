run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./...

test-cover:
	go test ./... -cover

compose-up:
	docker compose up --build

compose-down:
	docker compose down

migrate-up:
	migrate -path migrations -database "$${DATABASE_URL}" up

migrate-down:
	migrate -path migrations -database "$${DATABASE_URL}" down
