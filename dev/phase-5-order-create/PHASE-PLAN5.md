# PHASE-PLAN5 — Order Creation, Totals & Queue Producer

## Goal

Create orders: validate, load customer + products, copy prices, compute totals, persist order + items in one DB transaction, then publish the order id to Redis. Status lifecycle types and the `CREATED` state. Standalone module; fake repository, fake queue, and fake clock so unit tests need no DB/Redis.

## Scope (build exactly this)

- `internal/order/status.go` — `OrderStatus` defined type, all states, `ParseOrderStatus`, and the transition rule table (architecture §7) as a pure, tested function `CanTransition(from, to)`.
- `internal/order/model.go` — `Order`, `OrderItem` (architecture §6). Money decimal-safe.
- `internal/order/dto.go` — `CreateOrderInput`, `OrderOutput`, `OrderItemOutput`, `OrderFilter`.
- `internal/order/repository.go` — `OrderRepository` interface (architecture §18) + in-memory impl; a `Tx`/unit-of-work abstraction so create wraps order+items atomically.
- `internal/order/queue.go` — `OrderProducer` interface + fake; publishes the §9 message payload.
- `internal/order/service.go` — `Create` following the architecture §19 algorithm and §20 transaction boundary. Also `FindByID`, `List`.
- `internal/order/handler.go` — `POST /api/orders`, `GET /api/orders`, `GET /api/orders/{id}` (cancel/ship are Phase 6).
- Minimal in-folder scaffolding (errors, router, auth-context, customer/product *ports* it reads from — defined as interfaces with fakes, NOT full re-implementations).

## Entry condition

Foundation + auth + customer + product assumed. Represent customer/product dependencies as **interfaces** (ports: `CustomerLookup`, `ProductLookup`) with in-folder fakes. No dependency on other phase folders.

## Exit condition

`POST /api/orders` creates an order with correct totals and `CREATED` status, persists atomically, publishes to the fake queue after commit, returns 201. Transition rules and total calc unit-tested. All positive/negative tests green with no DB/Redis.

## Must-exist checklist

- [ ] `OrderStatus` type + `ParseOrderStatus` + `CanTransition` covering every §7 transition.
- [ ] Order created with status `CREATED` (BR-ORD-005).
- [ ] Order has a customer (BR-ORD-001) and ≥ 1 item (BR-ORD-002); quantity > 0 (BR-ORD-003).
- [ ] Unit price copied from product at creation (BR-PRD-005); item total = qty × unit_price.
- [ ] Order total = sum of item totals (BR-ORD-004), decimal-safe.
- [ ] Order + items persisted in one transaction; publish to queue **after** commit (BR-ORD-006, architecture §20).
- [ ] Commit ok + publish fail → 500 + logged inconsistency (no silent loss).
- [ ] Inactive customer rejected (`INACTIVE_CUSTOMER`); inactive product rejected (`INACTIVE_PRODUCT`).
- [ ] `OrderRepository`, `OrderProducer`, `CustomerLookup`, `ProductLookup` are interfaces with fakes.

## Must-NOT-exist checklist

- [ ] Publishing to Redis before the DB commit.
- [ ] Money as float64.
- [ ] Order persisted without its items (non-atomic create).
- [ ] Business rules / totals computed in the handler or repository.
- [ ] Live DB/Redis required by default tests.
- [ ] `TODO`/`FIXME`/debug prints.

## Positive tests

- create with valid customer + products → 201, status `CREATED`, correct total.
- unit price copied from product; item totals and order total correct (use the architecture §12 example numbers).
- order + items created atomically (fake repo records a single committed unit).
- message published to fake queue after commit, with the §9 payload shape.
- `CanTransition` allows every valid §7 transition.

## Negative tests

- empty items → `VALIDATION_ERROR` (BR-ORD-002).
- quantity < 1 → `VALIDATION_ERROR` (BR-ORD-003).
- inactive customer → `INACTIVE_CUSTOMER`.
- inactive product → `INACTIVE_PRODUCT`.
- unknown product id → `RESOURCE_NOT_FOUND`.
- `CanTransition` rejects every invalid transition.
- commit succeeds but publish fails → 500, inconsistency logged, order NOT lost.

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
