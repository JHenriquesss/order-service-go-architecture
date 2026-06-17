# Architecture

This document summarizes the Order Service Go architecture. The authoritative specification lives in the repository root `order-service-go-architecture.md`.

## Style

Layered architecture with clear boundaries:

- **Handler** — HTTP parsing, validation shape, response mapping
- **Service** — business rules and orchestration
- **Repository** — PostgreSQL persistence
- **Queue** — Redis-backed asynchronous order processing
- **Worker** — consumes queue messages and drives order lifecycle

Dependency direction flows inward. Business rules never live in handlers, SQL, or DTO mappers.

## Processes

| Process | Role |
|---------|------|
| `cmd/api` | REST API, enqueues orders after DB commit |
| `cmd/worker` | Bounded worker pool consuming Redis queue |

## Data stores

- **PostgreSQL** — users, customers, products, orders, order_items
- **Redis** — `orders:processing` list (LPUSH / BRPOP)

## Order lifecycle

```text
CREATED → PROCESSING → PAID | FAILED
CREATED → CANCELED
PAID → SHIPPED
```

## Package layout

See architecture §16. Key packages: `internal/auth`, `customer`, `product`, `order`, `queue`, `worker`, `metrics`, `middleware`, `config`, `database`, `errors`.

## API documentation

OpenAPI spec: [`openapi.yaml`](openapi.yaml). Swagger UI: `GET /swagger/index.html` (wired at root integration).

## Metrics

`GET /metrics` exposes plain-text counters per architecture §22 (`orders_created_total`, `orders_processed_total`, `orders_failed_total`, processing duration stats).

## Error format

All API errors use `{ "error": { "code", "message" } }` (architecture §24).
