# PHASE-PLAN1 — Foundation & Project Skeleton

## Goal

Stand up the runnable skeleton of `order-service-go`: configuration, structured logging, a consistent error model, database and Redis connection setup, migrations for the full schema, base HTTP server with core middleware, a `/health` endpoint, and the local Docker environment. No business features yet.

## Scope (build exactly this)

- `go.mod` (module path `order-service-go`, explicit stable `go` directive) + `go.sum`.
- `internal/config` — load config from env (architecture §26 vars). Validate required vars; return typed `Config` + error. No global state.
- `internal/logger` — structured logger (`slog`) with the required fields available (request_id, user_id, method, path, status_code, duration_ms, order_id, error).
- `internal/errors` — `AppError{Code, Message, HTTPStatus, cause}`, the error codes from architecture §24, helper constructors, and a single HTTP write function producing `{ "error": { "code", "message" } }`.
- `internal/database` — postgres pool constructor (pgx) behind a small interface; ping on startup. Live connection used only under `integration` build tag.
- `internal/queue` — Redis client constructor behind an interface (connection only; producer logic is Phase 5). Live connection only under `integration` tag.
- `internal/middleware` — `request_id`, `logging`, `recovery`.
- `internal/server` (or `internal/http`) — router (chi), wires middleware, mounts `GET /health`.
- `cmd/api/main.go` — compose config → logger → db → router → listen. `cmd/worker/main.go` — minimal stub that compiles (loops/no-op), real logic Phase 6.
- `migrations/` — up+down SQL for users, customers, products, orders, order_items, plus indexes (architecture §8). Exact DDL from architecture.
- `docker-compose.yml`, `Dockerfile`, `Makefile`, `.env.example` — from architecture §27–29 and §26.

## Entry condition

Empty folder. This phase starts the module.

## Exit condition

`go build ./...` succeeds, `/health` returns 200 via `httptest`, config/logger/errors unit-tested green, migrations files present and internally consistent (up/down paired, valid SQL), `go test ./...` green with no external services required.

## Must-exist checklist (all true to pass)

- [ ] `go.mod` with module `order-service-go` and explicit `go` directive; `go mod tidy` leaves no diff.
- [ ] `internal/config` parses all architecture §26 env vars and errors on missing required ones.
- [ ] `internal/logger` produces structured output and supports the required fields.
- [ ] `internal/errors` defines all §24 codes and a single JSON error writer.
- [ ] `internal/database` postgres constructor + interface; `internal/queue` redis constructor + interface.
- [ ] `request_id`, `logging`, `recovery` middleware wired into the router.
- [ ] `GET /health` returns 200 with a small JSON body.
- [ ] `cmd/api/main.go` and `cmd/worker/main.go` both build.
- [ ] Migrations for all 5 tables + indexes, each with paired `.up.sql`/`.down.sql`.
- [ ] `docker-compose.yml`, `Dockerfile`, `Makefile`, `.env.example` present and consistent with architecture.
- [ ] Unit tests: config parse (valid+missing), error JSON shape, health handler. All green without DB/Redis.

## Must-NOT-exist checklist (all absent to pass)

- [ ] No business logic (auth/customer/product/order) — foundation only.
- [ ] No global mutable state / package-level singletons for config or db.
- [ ] No `panic`/`log.Fatal` outside `main`.
- [ ] No hardcoded secrets (JWT secret, DB password) in Go code.
- [ ] No live DB/Redis connection required by default `go test ./...`.
- [ ] No `TODO`/`FIXME`/`panic("not implemented")`/debug prints.
- [ ] No unused dependencies in `go.mod`.

## Positive tests

- config loads a complete valid env into `Config`.
- logger writes a record containing a provided `request_id`.
- error writer renders a known `AppError` as `{"error":{"code","message"}}` with correct HTTP status.
- `GET /health` → 200 + expected body.
- recovery middleware turns a panic in a handler into a 500 JSON error (not a crash).
- request_id middleware sets/propagates an id.

## Negative tests

- config errors when a required var is missing/empty.
- unknown/invalid `LOG_LEVEL` handled deliberately (default or error — pick and test).
- error writer for an unmapped error falls back to `INTERNAL_ERROR` / 500 without leaking internals.

## Session log

After each meaningful step, append a concise objective entry to a `SESSION-LOG.md` in this folder (what was done), for later LLM-wiki consolidation.
