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

### Start infrastructure

From the repository root:

```bash
docker compose up -d postgres redis
```

Apply migrations (see below), then start the API and worker:

```bash
go run ./cmd/api
go run ./cmd/worker
```

Or use Docker Compose for all services:

```bash
docker compose up --build
```

The API listens on `http://localhost:8080`.

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
| `LOG_LEVEL` | Log level | `info` |

## Database Migrations

SQL migrations live in `migrations/`. Apply with your preferred tool or the project Makefile:

```bash
make migrate-up
```

Default admin user is seeded on API startup: `admin@example.com` / `123456`.

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

## Metrics

`GET /metrics` returns plain-text counters:

```text
orders_created_total
orders_processed_total
orders_failed_total
orders_processing_duration_seconds_sum
orders_processing_duration_seconds_count
orders_processing_duration_seconds_avg
```

## Tests

### Unit tests (default)

```bash
go test ./...
```

Integration tests are excluded unless the `integration` build tag is set.

### Integration tests

Requires live PostgreSQL, Redis, and a running API + worker (root integration).

```bash
export DATABASE_URL=postgres://orders:orders@localhost:5432/orders?sslmode=disable
export REDIS_ADDR=localhost:6379
export API_BASE_URL=http://localhost:8080

go test -tags=integration ./...
```

When the `integration` tag is set, missing or unreachable services cause tests to **fail** (not skip).

This isolated phase folder ships the integration test bodies and run procedure; the full live run is executed after phases 1–6 are integrated at the repository root.

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
