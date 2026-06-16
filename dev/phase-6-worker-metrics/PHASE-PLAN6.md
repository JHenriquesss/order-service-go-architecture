# PHASE-PLAN6 — Worker, Order Lifecycle & Metrics

## Goal

Asynchronous order processing with a Redis-fed worker pool, the remaining status transitions (cancel, ship, worker PROCESSING→PAID/FAILED), and operational metrics. Standalone module; fake queue, fake repository, and controllable fake processor so concurrency tests are deterministic and need no DB/Redis.

## Scope (build exactly this)

- `internal/worker/order_worker.go` — starts N goroutines (count from config), each consumes messages from an `OrderQueue` interface, processes via the order service, respects `context.Context` cancellation, and shuts down/drains cleanly.
- `internal/metrics/collector.go` — concurrency-safe counters: `orders_created_total`, `orders_processed_total`, `orders_failed_total`, processing-duration sum/count/avg (architecture §22).
- `internal/metrics/handler.go` — `GET /metrics` rendering the §22 text format.
- Order lifecycle additions: `Process` (CREATED→PROCESSING→PAID|FAILED), `Cancel` (CREATED only), `Ship` (PAID only), reusing the `CanTransition` rules; handlers `PATCH /api/orders/{id}/cancel`, `PATCH /api/orders/{id}/ship`.
- `cmd/worker/main.go` — real composition: config → deps → start worker pool → wait for signal → graceful shutdown.
- Minimal in-folder scaffolding (status rules, order repo port, errors, config) clean-room. No dependency on other phase folders.

## Entry condition

Foundation + order-create assumed. Recreate the **minimal** status rules + order repository port + queue port as interfaces with fakes. No dependency on other phase folders.

## Exit condition

Worker processes a CREATED order to PAID, marks failures FAILED, increments metrics; cancel/ship enforce transition rules; `go test -race ./...` green; cancellation + graceful shutdown tested; no DB/Redis needed by default tests.

## Must-exist checklist

- [ ] Worker starts a bounded pool (count from config), each goroutine context-aware.
- [ ] Worker processes only `CREATED` orders (BR-ORD-009); ignores others.
- [ ] Lifecycle: CREATED→PROCESSING→PAID on success; →FAILED on error (BR-ORD-010).
- [ ] Processing failure → status FAILED + `orders_failed_total`++ + error log with order id.
- [ ] Success → `orders_processed_total`++ + duration recorded.
- [ ] `Cancel` only CREATED (BR-ORD-007); `Ship` only PAID (BR-ORD-008); invalid → `INVALID_ORDER_STATUS`/400.
- [ ] Metrics counters concurrency-safe; `/metrics` renders §22 format.
- [ ] Graceful shutdown drains/stops workers on context cancel / signal.

## Must-NOT-exist checklist

- [ ] Detached fire-and-forget goroutines with no supervision.
- [ ] `time.Sleep` used for synchronization in tests or code.
- [ ] Data races (must pass `-race`).
- [ ] Unbounded goroutine creation per message.
- [ ] Copying lock-bearing structs.
- [ ] Invalid status transition allowed.
- [ ] Live DB/Redis required by default tests.
- [ ] `TODO`/`FIXME`/debug prints.

## Positive tests

- worker processes a CREATED order → PROCESSING then PAID; processed metric ++.
- processing error → FAILED; failed metric ++; error logged with order id.
- worker ignores a non-CREATED order.
- cancel a CREATED order → CANCELED.
- ship a PAID order → SHIPPED.
- `/metrics` renders all §22 counters with avg = sum/count.
- context cancellation stops the pool; shutdown drains in-flight work.

## Negative tests

- cancel a PAID order → `INVALID_ORDER_STATUS`/400.
- ship a CREATED order → `INVALID_ORDER_STATUS`/400.
- worker handles a malformed/unknown order id message without crashing the pool.
- duration count of 0 → avg renders without divide-by-zero.

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
