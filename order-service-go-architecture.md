# Order Service Go — Architecture

## 1. Project

**Repository:** `order-service-go`

**Purpose:** REST API for customer, product, and order management with asynchronous order processing using Go, PostgreSQL, Redis, JWT authentication, structured logs, tests, metrics, and OpenAPI documentation.

## 2. Scope

### Included

- Customer CRUD.
- Product CRUD.
- Order creation.
- Order total calculation.
- Order status flow.
- Background order processing.
- Redis-backed queue.
- JWT authentication.
- Role-based authorization.
- Structured logging.
- Basic operational metrics.
- Swagger/OpenAPI documentation.
- Unit and integration tests.
- Docker Compose environment.

### Not Included in the First Version

- Payment gateway integration.
- Inventory reservation.
- Multi-tenant support.
- Frontend application.
- Event streaming with Kafka.
- Kubernetes deployment.

## 3. Technology Stack

| Area | Technology |
|---|---|
| Language | Go |
| HTTP router | Chi or Gin |
| Database | PostgreSQL |
| Queue/cache | Redis |
| Authentication | JWT |
| Migrations | golang-migrate |
| SQL access | pgx or sqlc |
| Logging | slog or zap |
| Documentation | Swagger/OpenAPI |
| Tests | testing, testify, httptest |
| Containers | Docker, Docker Compose |
| Metrics | Prometheus-style HTTP metrics |

## 4. Architectural Style

Use a modular monolith with clear internal boundaries.

```text
HTTP Handler -> Service -> Repository -> Database
                    |
                    v
                Queue Producer -> Redis -> Worker -> Service -> Database
```

The API and the worker can run from the same codebase using different commands:

```text
cmd/api/main.go
cmd/worker/main.go
```

## 5. Main Modules

```text
auth
customer
product
order
queue
worker
metrics
config
database
logger
middleware
```

## 6. Domain Model

### User

Represents an authenticated system user.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| name | string | Required |
| email | string | Unique |
| password_hash | string | BCrypt hash |
| role | string | ADMIN or OPERATOR |
| active | bool | Default true |
| created_at | timestamp | Required |
| updated_at | timestamp | Required |

### Customer

Represents a buyer.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| name | string | Required |
| document | string | Unique |
| email | string | Optional |
| phone | string | Optional |
| active | bool | Default true |
| created_at | timestamp | Required |
| updated_at | timestamp | Required |

### Product

Represents an item that can be sold.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| name | string | Required |
| sku | string | Unique |
| price | numeric | Required, greater than zero |
| active | bool | Default true |
| created_at | timestamp | Required |
| updated_at | timestamp | Required |

### Order

Represents a customer purchase.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| customer_id | UUID | FK to customers |
| status | string | CREATED, PROCESSING, PAID, SHIPPED, CANCELED, FAILED |
| total_amount | numeric | Calculated from items |
| created_by | UUID | FK to users |
| created_at | timestamp | Required |
| updated_at | timestamp | Required |
| processed_at | timestamp | Nullable |

### Order Item

Represents one product inside an order.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| order_id | UUID | FK to orders |
| product_id | UUID | FK to products |
| quantity | int | Required, greater than zero |
| unit_price | numeric | Product price at order creation time |
| total_price | numeric | quantity * unit_price |

## 7. Order Status Flow

```text
CREATED -> PROCESSING -> PAID -> SHIPPED
       \                 \
        \                 -> FAILED
         -> CANCELED
```

### Status Rules

| Status | Meaning |
|---|---|
| CREATED | Order was created and queued |
| PROCESSING | Worker started processing the order |
| PAID | Order was processed successfully |
| SHIPPED | Order was manually marked as shipped |
| CANCELED | Order was canceled before processing or shipping |
| FAILED | Worker failed to process the order |

### Transition Rules

| From | To | Allowed By |
|---|---|---|
| CREATED | PROCESSING | Worker |
| PROCESSING | PAID | Worker |
| PROCESSING | FAILED | Worker |
| CREATED | CANCELED | ADMIN, OPERATOR |
| PAID | SHIPPED | ADMIN, OPERATOR |

Invalid transitions must return `400 Bad Request`.

## 8. Database Schema

### `users`

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    email VARCHAR(160) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(30) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### `customers`

```sql
CREATE TABLE customers (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    document VARCHAR(30) NOT NULL UNIQUE,
    email VARCHAR(160),
    phone VARCHAR(30),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### `products`

```sql
CREATE TABLE products (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    sku VARCHAR(80) NOT NULL UNIQUE,
    price NUMERIC(15, 2) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT chk_products_price_positive CHECK (price > 0)
);
```

### `orders`

```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id),
    status VARCHAR(30) NOT NULL,
    total_amount NUMERIC(15, 2) NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT chk_orders_total_amount_non_negative CHECK (total_amount >= 0)
);
```

### `order_items`

```sql
CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price NUMERIC(15, 2) NOT NULL,
    total_price NUMERIC(15, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT chk_order_items_quantity_positive CHECK (quantity > 0),
    CONSTRAINT chk_order_items_unit_price_positive CHECK (unit_price > 0),
    CONSTRAINT chk_order_items_total_price_positive CHECK (total_price > 0)
);
```

### Indexes

```sql
CREATE INDEX idx_customers_document ON customers(document);
CREATE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
```

## 9. Queue Design

Use Redis as a simple queue for background processing.

### Queue Name

```text
orders:processing
```

### Message Payload

```json
{
  "order_id": "8fd9ad06-6db5-49ad-8f75-0e1e9a43c76f",
  "event": "ORDER_CREATED",
  "created_at": "2026-01-01T10:00:00Z"
}
```

### Producer Flow

```text
POST /api/orders
  -> validate request
  -> load customer
  -> load products
  -> calculate totals
  -> create order with status CREATED
  -> create order items
  -> commit transaction
  -> push message to Redis queue
  -> return 201 Created
```

### Worker Flow

```text
Worker reads message from Redis
  -> load order
  -> ignore if order is not CREATED
  -> update status to PROCESSING
  -> simulate/process payment
  -> update status to PAID or FAILED
  -> update metrics
  -> log result
```

## 10. Worker Strategy

Use one or more goroutines to process Redis messages.

```text
worker process
  -> starts N goroutines
  -> each goroutine blocks waiting for Redis messages
  -> each message is processed independently
```

Recommended initial worker count:

```text
ORDER_WORKER_COUNT=3
```

### Failure Handling

Initial version:

- If processing fails, mark order as `FAILED`.
- Log the error with order ID.
- Increment failed orders metric.

Optional improvement:

- Add retry count to the Redis payload.
- Retry up to 3 times.
- Move failed messages to `orders:dead-letter`.

## 11. API Endpoints

### Authentication

```text
POST /api/auth/login
POST /api/auth/register
```

### Customers

```text
GET    /api/customers
GET    /api/customers/{id}
POST   /api/customers
PUT    /api/customers/{id}
PATCH  /api/customers/{id}/deactivate
```

### Products

```text
GET    /api/products
GET    /api/products/{id}
POST   /api/products
PUT    /api/products/{id}
PATCH  /api/products/{id}/deactivate
```

### Orders

```text
GET    /api/orders
GET    /api/orders/{id}
POST   /api/orders
PATCH  /api/orders/{id}/cancel
PATCH  /api/orders/{id}/ship
```

### Metrics

```text
GET /metrics
```

### Health

```text
GET /health
```

### Swagger

```text
GET /swagger/index.html
```

## 12. Request and Response Examples

### Login Request

```json
{
  "email": "admin@example.com",
  "password": "123456"
}
```

### Login Response

```json
{
  "access_token": "jwt-token",
  "token_type": "Bearer",
  "expires_in": 7200
}
```

### Create Customer Request

```json
{
  "name": "ACME Ltd",
  "document": "12345678000199",
  "email": "contact@acme.com",
  "phone": "+55 11 99999-9999"
}
```

### Create Product Request

```json
{
  "name": "Wireless Mouse",
  "sku": "MOUSE-WIRELESS-001",
  "price": 89.90
}
```

### Create Order Request

```json
{
  "customer_id": "5e7fd82a-08b5-43ef-8b76-8e919e5d72f4",
  "items": [
    {
      "product_id": "9c3792f2-8602-4c39-b0f2-65c2e6c7a53b",
      "quantity": 2
    },
    {
      "product_id": "e01e7a3c-8800-43f7-99f4-893c7ef3a71e",
      "quantity": 1
    }
  ]
}
```

### Create Order Response

```json
{
  "id": "8fd9ad06-6db5-49ad-8f75-0e1e9a43c76f",
  "customer_id": "5e7fd82a-08b5-43ef-8b76-8e919e5d72f4",
  "status": "CREATED",
  "total_amount": 269.70,
  "items": [
    {
      "product_id": "9c3792f2-8602-4c39-b0f2-65c2e6c7a53b",
      "quantity": 2,
      "unit_price": 89.90,
      "total_price": 179.80
    },
    {
      "product_id": "e01e7a3c-8800-43f7-99f4-893c7ef3a71e",
      "quantity": 1,
      "unit_price": 89.90,
      "total_price": 89.90
    }
  ],
  "created_at": "2026-01-01T10:00:00Z"
}
```

## 13. Filtering and Pagination

### Customers

```text
GET /api/customers?name=acme&document=123&active=true&page=1&page_size=20
```

### Products

```text
GET /api/products?name=mouse&sku=MOUSE&active=true&page=1&page_size=20
```

### Orders

```text
GET /api/orders?customer_id={uuid}&status=PAID&start_date=2026-01-01&end_date=2026-01-31&page=1&page_size=20
```

## 14. Business Rules

### Customers

```text
BR-CUS-001: Customer name is required.
BR-CUS-002: Customer document must be unique.
BR-CUS-003: Inactive customers cannot create new orders.
BR-CUS-004: Customers with orders must not be physically deleted.
```

### Products

```text
BR-PRD-001: Product name is required.
BR-PRD-002: Product SKU must be unique.
BR-PRD-003: Product price must be greater than zero.
BR-PRD-004: Inactive products cannot be used in new orders.
BR-PRD-005: Product price must be copied into the order item at order creation time.
```

### Orders

```text
BR-ORD-001: Order must have a customer.
BR-ORD-002: Order must have at least one item.
BR-ORD-003: Item quantity must be greater than zero.
BR-ORD-004: Order total is the sum of all item totals.
BR-ORD-005: Order is created with status CREATED.
BR-ORD-006: Created order must be sent to the processing queue.
BR-ORD-007: Only CREATED orders can be canceled.
BR-ORD-008: Only PAID orders can be shipped.
BR-ORD-009: Worker can process only CREATED orders.
BR-ORD-010: Failed processing must move the order to FAILED.
```

### Authentication and Authorization

```text
BR-AUTH-001: Only authenticated users can access business endpoints.
BR-AUTH-002: ADMIN can manage users.
BR-AUTH-003: ADMIN and OPERATOR can manage customers, products, and orders.
BR-AUTH-004: Inactive users cannot authenticate.
```

## 15. Authorization Matrix

| Resource | ADMIN | OPERATOR |
|---|---:|---:|
| Create users | Yes | No |
| List users | Yes | No |
| Create customers | Yes | Yes |
| Update customers | Yes | Yes |
| Create products | Yes | Yes |
| Update products | Yes | Yes |
| Create orders | Yes | Yes |
| Cancel orders | Yes | Yes |
| Ship orders | Yes | Yes |
| View metrics | Yes | No |

## 16. Internal Package Structure

```text
order-service-go/
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── jwt.go
│   │   ├── password.go
│   │   ├── dto.go
│   │   └── model.go
│   ├── customer/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── dto.go
│   │   └── model.go
│   ├── product/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── dto.go
│   │   └── model.go
│   ├── order/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── dto.go
│   │   ├── model.go
│   │   └── status.go
│   ├── queue/
│   │   ├── redis_queue.go
│   │   └── message.go
│   ├── worker/
│   │   └── order_worker.go
│   ├── metrics/
│   │   ├── handler.go
│   │   └── collector.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── request_id.go
│   │   ├── logging.go
│   │   └── recovery.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── postgres.go
│   ├── logger/
│   │   └── logger.go
│   └── errors/
│       ├── app_error.go
│       └── handler.go
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_customers_table.up.sql
│   ├── 000002_create_customers_table.down.sql
│   ├── 000003_create_products_table.up.sql
│   ├── 000003_create_products_table.down.sql
│   ├── 000004_create_orders_table.up.sql
│   └── 000004_create_orders_table.down.sql
├── docs/
│   ├── architecture.md
│   ├── api.md
│   ├── business-rules.md
│   └── openapi.yaml
├── scripts/
│   ├── migrate-up.sh
│   ├── migrate-down.sh
│   └── seed.sh
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
├── go.mod
└── README.md
```

## 17. Service Interfaces

### Customer Service

```go
type CustomerService interface {
    Create(ctx context.Context, input CreateCustomerInput) (*CustomerOutput, error)
    Update(ctx context.Context, id uuid.UUID, input UpdateCustomerInput) (*CustomerOutput, error)
    Deactivate(ctx context.Context, id uuid.UUID) error
    FindByID(ctx context.Context, id uuid.UUID) (*CustomerOutput, error)
    List(ctx context.Context, filter CustomerFilter) (*Page[CustomerOutput], error)
}
```

### Product Service

```go
type ProductService interface {
    Create(ctx context.Context, input CreateProductInput) (*ProductOutput, error)
    Update(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*ProductOutput, error)
    Deactivate(ctx context.Context, id uuid.UUID) error
    FindByID(ctx context.Context, id uuid.UUID) (*ProductOutput, error)
    List(ctx context.Context, filter ProductFilter) (*Page[ProductOutput], error)
}
```

### Order Service

```go
type OrderService interface {
    Create(ctx context.Context, input CreateOrderInput, userID uuid.UUID) (*OrderOutput, error)
    Cancel(ctx context.Context, id uuid.UUID) (*OrderOutput, error)
    Ship(ctx context.Context, id uuid.UUID) (*OrderOutput, error)
    Process(ctx context.Context, id uuid.UUID) error
    FindByID(ctx context.Context, id uuid.UUID) (*OrderOutput, error)
    List(ctx context.Context, filter OrderFilter) (*Page[OrderOutput], error)
}
```

## 18. Repository Interfaces

```go
type OrderRepository interface {
    Create(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
    MarkProcessed(ctx context.Context, id uuid.UUID, status OrderStatus, processedAt time.Time) error
    List(ctx context.Context, filter OrderFilter) (*Page[Order], error)
}
```

```go
type ProductRepository interface {
    Create(ctx context.Context, product *Product) error
    FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
    FindManyByID(ctx context.Context, ids []uuid.UUID) ([]Product, error)
    Update(ctx context.Context, product *Product) error
    Deactivate(ctx context.Context, id uuid.UUID) error
}
```

## 19. Order Creation Algorithm

```text
1. Validate request body.
2. Verify authenticated user.
3. Load customer by ID.
4. Reject inactive customer.
5. Load all products by ID.
6. Reject inactive products.
7. Validate all quantities.
8. Copy product prices into order items.
9. Calculate item total: quantity * unit price.
10. Calculate order total: sum of item totals.
11. Create order with status CREATED.
12. Create order items in the same database transaction.
13. Commit transaction.
14. Publish order ID to Redis queue.
15. Return created order.
```

## 20. Transaction Boundary

Order creation must use a database transaction.

```text
BEGIN
  INSERT INTO orders
  INSERT INTO order_items
COMMIT
PUBLISH TO REDIS
```

Do not publish to Redis before the database commit.

If Redis publish fails after commit, return `201 Created` with a warning log, or return `202 Accepted` only after successful queue publishing. For the initial version, prefer this behavior:

```text
Database commit succeeds + Redis publish fails = return 500 and log the inconsistency
```

Recommended improvement:

```text
Use the Outbox Pattern.
```

## 21. Outbox Pattern as Future Improvement

For the first version, direct Redis publishing is acceptable.

For a more robust version:

```text
orders table
outbox_events table
outbox_publisher worker
Redis queue
```

This prevents orders from being created without a corresponding queue message.

## 22. Metrics

Expose simple metrics at `/metrics`.

### Required Metrics

```text
orders_created_total
orders_processed_total
orders_failed_total
orders_processing_duration_seconds_sum
orders_processing_duration_seconds_count
orders_processing_duration_seconds_avg
```

### Example Response

```text
orders_created_total 120
orders_processed_total 115
orders_failed_total 5
orders_processing_duration_seconds_sum 230.50
orders_processing_duration_seconds_count 115
orders_processing_duration_seconds_avg 2.00
```

Initial implementation can store counters in memory.

Future improvement: expose metrics using Prometheus client library.

## 23. Structured Logging

Use structured logs for every important operation.

### Required Fields

```text
request_id
user_id
method
path
status_code
duration_ms
order_id
error
```

### Example

```json
{
  "level": "info",
  "msg": "order processed successfully",
  "order_id": "8fd9ad06-6db5-49ad-8f75-0e1e9a43c76f",
  "duration_ms": 430
}
```

## 24. Error Response Format

Use one consistent error format.

```json
{
  "error": {
    "code": "INVALID_ORDER_STATUS",
    "message": "Only PAID orders can be shipped."
  }
}
```

### Common Error Codes

```text
VALIDATION_ERROR
UNAUTHORIZED
FORBIDDEN
RESOURCE_NOT_FOUND
DUPLICATE_RESOURCE
INVALID_ORDER_STATUS
INACTIVE_CUSTOMER
INACTIVE_PRODUCT
INTERNAL_ERROR
```

## 25. Middleware

Required middleware:

```text
Request ID
Request logging
Panic recovery
JWT authentication
Role authorization
CORS, if needed
```

## 26. Configuration

Use environment variables.

```text
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://orders:orders@localhost:5432/orders?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
JWT_SECRET=change-me
JWT_EXPIRATION_MINUTES=120
ORDER_WORKER_COUNT=3
LOG_LEVEL=info
```

## 27. Docker Compose

```yaml
services:
  postgres:
    image: postgres:16
    container_name: order-service-postgres
    environment:
      POSTGRES_DB: orders
      POSTGRES_USER: orders
      POSTGRES_PASSWORD: orders
    ports:
      - "5432:5432"
    volumes:
      - order_service_postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    container_name: order-service-redis
    ports:
      - "6379:6379"

  api:
    build: .
    container_name: order-service-api
    depends_on:
      - postgres
      - redis
    environment:
      APP_ENV: development
      HTTP_PORT: 8080
      DATABASE_URL: postgres://orders:orders@postgres:5432/orders?sslmode=disable
      REDIS_ADDR: redis:6379
      JWT_SECRET: change-me
      JWT_EXPIRATION_MINUTES: 120
      LOG_LEVEL: info
    ports:
      - "8080:8080"
    command: ["/app/api"]

  worker:
    build: .
    container_name: order-service-worker
    depends_on:
      - postgres
      - redis
    environment:
      APP_ENV: development
      DATABASE_URL: postgres://orders:orders@postgres:5432/orders?sslmode=disable
      REDIS_ADDR: redis:6379
      JWT_SECRET: change-me
      ORDER_WORKER_COUNT: 3
      LOG_LEVEL: info
    command: ["/app/worker"]

volumes:
  order_service_postgres_data:
```

## 28. Dockerfile

```dockerfile
FROM golang:1.23-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/api ./cmd/api
RUN go build -o /bin/worker ./cmd/worker

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/api /app/api
COPY --from=build /bin/worker /app/worker

EXPOSE 8080

CMD ["/app/api"]
```

## 29. Makefile

```makefile
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
```

## 30. Testing Strategy

### Unit Tests

```text
customer service
product service
order service
JWT service
password service
status transition rules
order total calculation
```

### Handler Tests

Use `httptest`.

```text
POST /api/customers
POST /api/products
POST /api/orders
PATCH /api/orders/{id}/cancel
PATCH /api/orders/{id}/ship
```

### Integration Tests

Use test containers or Docker-backed test database.

```text
repository tests with PostgreSQL
queue tests with Redis
authenticated API tests
order creation + Redis publish test
worker processing test
```

## 31. Minimum Test Cases

### Order Service

```text
should create order with valid customer and products
should reject inactive customer
should reject inactive product
should reject empty items
should reject quantity less than one
should calculate total amount correctly
should copy product price into order item
should create order with status CREATED
should publish order to queue after creation
should cancel CREATED order
should not cancel PAID order
should ship PAID order
should not ship CREATED order
```

### Worker

```text
should process CREATED order
should ignore non-CREATED order
should mark order as PROCESSING before processing
should mark order as PAID after success
should mark order as FAILED after processing error
should increment processed metric
should increment failed metric
```

### Auth

```text
should login with valid credentials
should reject invalid password
should reject inactive user
should reject missing token
should reject invalid token
should block OPERATOR from admin-only endpoint
```

## 32. OpenAPI Documentation

Use Swagger to document:

```text
auth endpoints
customer endpoints
product endpoints
order endpoints
request schemas
response schemas
error schemas
JWT authorization header
```

The README must include the Swagger URL:

```text
http://localhost:8080/swagger/index.html
```

## 33. README Structure

```text
# Order Service Go

## About
## Features
## Architecture
## Tech Stack
## How to Run
## Environment Variables
## Database Migrations
## Authentication
## API Documentation
## Main Endpoints
## Order Processing Flow
## Metrics
## Tests
## Project Structure
## Business Rules
## Future Improvements
```

## 34. Initial Seed Data

Create an admin user and sample products.

```text
admin@example.com / 123456
```

Use BCrypt for the stored password hash.

## 35. Recommended Commit Plan

```text
chore: initialize go project
chore: add docker compose with postgres and redis
chore: add database migrations
chore: configure application environment
feat: implement structured logger
feat: implement auth with jwt
feat: implement customer crud
feat: implement product crud
feat: implement order creation
feat: calculate order totals
feat: publish order created event to redis
feat: implement order worker
feat: add order status transitions
feat: add metrics endpoint
feat: add swagger documentation
feat: add filtering and pagination
test: add order service tests
test: add handler tests
test: add worker tests
docs: add architecture documentation
docs: improve readme with examples
```

## 36. MVP Plan

### Version 1

```text
JWT login
Customer CRUD
Product CRUD
Order creation
Order total calculation
Redis queue publish
Worker processing
Status CREATED -> PROCESSING -> PAID
Docker Compose
README
```

### Version 2

```text
Order cancel
Order ship
Filtering and pagination
Structured logs
Metrics endpoint
Swagger documentation
```

### Version 3

```text
Integration tests
Retry strategy
Dead-letter queue
Outbox pattern
Prometheus metrics
More complete README examples
```

## 37. Final Expected Result

The repository should demonstrate:

```text
REST API design
PostgreSQL modeling
JWT authentication
Clean service/repository separation
Order total calculation
Asynchronous processing with Redis
Go concurrency with workers
Structured logging
Basic metrics
Tests
Docker-based local environment
OpenAPI documentation
```
