# Order Service Go

## About

REST API for customer, product, and order management with asynchronous order processing. Orders are persisted in PostgreSQL, enqueued to Redis, and processed by a separate worker process.

## Features

- JWT authentication with role-based authorization (ADMIN, OPERATOR)
- Customer, product, and order CRUD
- Asynchronous order processing (CREATED → PROCESSING → PAID | FAILED)
- Order cancel and ship lifecycle operations
- Structured JSON logging with request correlation
- Operational metrics endpoint
- OpenAPI / Swagger documentation

## Architecture

Layered design: Handler → Service → Repository → PostgreSQL. The API publishes order-created events to Redis after the database transaction commits; a worker pool consumes messages and drives payment processing.

See [`docs/architecture.md`](docs/architecture.md) for details.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.23+ |
| HTTP router | chi |
| Database | PostgreSQL 16 |
| Queue | Redis 7 |
| Auth | JWT (HMAC) |
| Documentation | OpenAPI 3 / Swagger UI |

## How to Run

### Prerequisites

- Go 1.23+
- Docker and Docker Compose (for PostgreSQL and Redis)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (`migrate`) for local `make migrate-up` / `make seed`

### One command (full stack)

From the repository root:

```bash
make compose-up
```

This starts PostgreSQL and Redis (with healthchecks), applies SQL migrations via a one-shot `migrate` service, then builds and runs the API, worker, and Prometheus. The API listens on `http://localhost:8080`; worker metrics on `http://localhost:9090`; Prometheus on `http://localhost:9091`.

If host port `5432` is already in use, set `POSTGRES_HOST_PORT` (for example `POSTGRES_HOST_PORT=5433 make compose-up`) and point `DATABASE_URL` at that port for host-side tools and integration tests.

Verify:

- `GET http://localhost:8080/health` → 200
- `GET http://localhost:8080/swagger/index.html` → Swagger UI

Stop the stack:

```bash
make compose-down
```

### Local processes (infra in Docker)

Start only Postgres and Redis, apply migrations, seed the default admin, then run API and worker on the host:

```bash
docker compose up -d postgres redis
export DATABASE_URL=postgres://orders:orders@localhost:5432/orders?sslmode=disable
export REDIS_ADDR=localhost:6379
export JWT_SECRET=change-me
make migrate-up
make seed
make run-api    # terminal 1
make run-worker # terminal 2
```

Default admin after seed or first API start: `admin@example.com` / `123456`.

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `APP_ENV` | Environment name | `development` |
| `HTTP_PORT` | API listen port | `8080` |
| `DATABASE_URL` | PostgreSQL DSN | `postgres://orders:orders@localhost:5432/orders?sslmode=disable` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password (optional) | |
| `JWT_SECRET` | HMAC signing secret | `change-me` |
| `JWT_EXPIRATION_MINUTES` | Token lifetime | `120` |
| `ORDER_WORKER_COUNT` | Worker goroutines | `3` |
| `ORDER_MAX_RETRIES` | Max re-tries after the first processing attempt (default 3 → 4 total tries) | `3` |
| `ORDER_TRANSIENT_FAILURE_TOTAL` | Worker-only: order total that triggers simulated infra failure (integration / dead-letter verification) | `42.42` |
| `LOG_LEVEL` | Log level | `info` |

## Database Migrations

SQL migrations live in `migrations/`. With Docker Compose, migrations run automatically via the `migrate` service before API and worker start.

For a host-only workflow:

```bash
export DATABASE_URL=postgres://orders:orders@localhost:5432/orders?sslmode=disable
make migrate-up
make seed
```

Rollback one step: `make migrate-down` (requires `DATABASE_URL`).

## Authentication

1. `POST /api/auth/login` with email and password to obtain a JWT.
2. Send `Authorization: Bearer <access_token>` on protected routes.

Roles: **ADMIN** (full access) and **OPERATOR** (customers, products, orders). See architecture §15 for the authorization matrix.

## API Documentation

**Swagger UI:** [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

OpenAPI spec: [`docs/openapi.yaml`](docs/openapi.yaml). Human-readable summary: [`docs/api.md`](docs/api.md).

Swagger UI is mounted at `GET /swagger/index.html` when the root application integrates this phase (typically via `http-swagger` serving `docs/openapi.yaml`).

## Main Endpoints

| Area | Endpoints |
|------|-----------|
| Auth | `POST /api/auth/login`, `POST /api/auth/register` |
| Customers | `GET/POST /api/customers`, `GET/PUT /api/customers/{id}`, `PATCH .../deactivate` |
| Products | `GET/POST /api/products`, `GET/PUT /api/products/{id}`, `PATCH .../deactivate` |
| Orders | `GET/POST /api/orders`, `GET /api/orders/{id}`, `PATCH .../cancel`, `PATCH .../ship` |
| Health | `GET /health` |
| Metrics | `GET /metrics` |

## Order Processing Flow

1. Client creates an order via `POST /api/orders` (status `CREATED`).
2. API commits the order and items to PostgreSQL, then LPUSHes an `ORDER_CREATED` message to Redis.
3. Worker dequeues the message, transitions the order to `PROCESSING`, simulates payment, then sets `PAID` or `FAILED`.
4. Metrics counters update on create, process, and failure.

## Resilience

When `Service.Process` returns a **transient** error (database load/update failure), the worker re-publishes the message to `orders:processing` with an incremented `retry_count`. After **`ORDER_MAX_RETRIES` re-tries** (default **3**), the message is moved to the dead-letter queue **`orders:dead-letter`** and logged with `order_id` and final `retry_count`.

**Retry semantics:** default `ORDER_MAX_RETRIES=3` means **1 initial attempt + up to 3 re-tries** (4 processing attempts total). A message dead-lettered after exhausting retries carries `retry_count: 3`. Set `ORDER_MAX_RETRIES=0` to dead-letter on the first transient failure with no re-tries.

**Payment declines are not retried.** When payment fails, `Service.Process` marks the order `FAILED` and returns `nil` (terminal business outcome). Those messages are never requeued and never dead-lettered.

Inspect dead-lettered messages:

```bash
redis-cli LRANGE orders:dead-letter 0 -1
```

**Integration verification:** with `make compose-up`, the worker is configured with `ORDER_TRANSIENT_FAILURE_TOTAL=42.42`. Create an order whose total is **42.42** to trigger simulated infrastructure failures; after retries exhaust, the message appears in `orders:dead-letter`. Orders with total **13.37** still fail payment once (`FAILED`) and do not enter the dead-letter queue.

Re-publishing uses immediate re-enqueue (no backoff). Worker shutdown mid-retry follows existing at-most-once queue semantics: an in-flight message may be lost if the process exits before requeue/dead-letter completes.

## Metrics

Both the **API** (`:8080`) and **worker** (`:9090`, `METRICS_PORT`) expose Prometheus exposition at `GET /metrics`. A dev **Prometheus** service scrapes both jobs when you bring the stack up with `make compose-up`.

| Process | Endpoints | Port (default) |
|---------|-----------|----------------|
| API | `GET /metrics`, `GET /health` | 8080 |
| Worker | `GET /metrics`, `GET /health` | 9090 (`METRICS_PORT`) |
| Prometheus UI | targets + query | 9091 (host) → 9090 (container) |

Exposed series:

- `orders_created_total` (API increments on create)
- `orders_processed_total`, `orders_failed_total` (worker increments on process/failure)
- `orders_processing_duration_seconds_{sum,count,bucket}` (worker, successful processing)

Average processing duration is **not** an exposed series. Derive it in PromQL:

```promql
rate(orders_processing_duration_seconds_sum[5m]) / rate(orders_processing_duration_seconds_count[5m])
```

Verify scrape targets after `make compose-up`: open `http://localhost:9091/targets` — both `api` and `worker` jobs should be **UP**.

## Tests

### Unit tests (default)

```bash
go test ./...
go test -race ./...
```

Integration tests are excluded unless the `integration` build tag is set.

### Integration tests (live Postgres + Redis + API + worker)

**One-command stack**, then run the suite:

```bash
make compose-up
export DATABASE_URL=postgres://orders:orders@localhost:5432/orders?sslmode=disable
export REDIS_ADDR=localhost:6379
export API_BASE_URL=http://localhost:8080
export WORKER_METRICS_URL=http://localhost:9090
make test-integration
```

Or use the Makefile defaults (same DSN/addresses as above):

```bash
make compose-up
make test-integration
```

The suite covers: Postgres customer round-trip, Redis queue publish/consume, E2E happy path (login → customer → product → order → worker → **PAID** + `orders_created_total` on API and `orders_processed_total` on worker), failure path → **FAILED** + `orders_failed_total` on worker, **transient failure → `orders:dead-letter`** (product total 42.42), **payment decline absent from dead-letter**, worker `/health` + `/metrics`, unauthenticated **401**, and unreachable Postgres (bad DSN fails loudly).

When the `integration` tag is set, missing or unreachable services cause tests to **fail** (not skip).

Tear down: `make compose-down`.

## Project Structure

```text
cmd/api/          HTTP API entrypoint
cmd/worker/       Queue consumer entrypoint
internal/         Domain packages (auth, customer, product, order, queue, worker, metrics, ...)
migrations/       SQL migrations
docs/             OpenAPI spec and architecture docs
tests/integration Integration tests (build tag: integration)
```

See [`docs/architecture.md`](docs/architecture.md) and architecture §16 for the full tree.

## Business Rules

Documented in [`docs/business-rules.md`](docs/business-rules.md). Key examples:

- Inactive customers/products cannot be used in new orders
- Only `CREATED` orders can be canceled; only `PAID` orders can be shipped
- Worker processes only `CREATED` orders; failures move to `FAILED`

## Future Improvements

- Outbox pattern for reliable queue publishing (architecture §21)
- Rate limiting and CORS middleware
- Kubernetes deployment manifests
- CI pipeline with testcontainers for integration tests
