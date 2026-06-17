# PHASE-PLAN10 — Resilience: Retry + Dead-Letter Queue

## Goal

Stop dropping messages the worker fails to process. Add bounded **retry** (retry count carried in the Redis payload, capped at 3 attempts) and a **dead-letter queue** (`orders:dead-letter`) for messages that exhaust retries — architecture §10 "Optional improvement". Today, when `order.Service.Process` returns an error, the worker logs it and the message is gone forever ([order_worker.go] `runLoop`). This phase makes transient processing failures survivable.

> Like phases 8-9, this operates on the **already-integrated root repository** (module `order-service-go`), not a clean-room slice. Standalone-module rules from AGENTS.md still apply to any in-folder spike; the real target is the root repo. Needs Docker (Postgres + Redis) for the live gate.

## The distinction that governs the whole phase (read carefully)

There are **two different kinds of "failure"** and they must not be conflated:

1. **Payment decline (business outcome).** `processor.ProcessPayment` returns an error → `Service.Process` marks the order **FAILED** and returns **`nil`**. This is a *terminal, correct* result. It MUST NOT be retried and MUST NOT be dead-lettered. An order that legitimately failed payment is done.
2. **Transient / infrastructure failure (message not processed).** `Service.Process` returns a **non-nil error** (DB load failed, `UpdateStatus`/`MarkProcessed` repo error, etc.). The order's outcome is still undetermined. *This* is what gets retried, and dead-lettered after the cap.

The retry trigger is therefore exactly: **`Service.Process(...) != nil`**. Do not add retry logic inside the payment path, and do not change the rule that a declined payment is a `nil`-return FAILED. If you find yourself retrying FAILED orders, you have conflated the two — stop.

## Scope (build/verify exactly this)

- **Message format**: add `RetryCount int` (`json:"retry_count"`) to `order.OrderCreatedMessage`. Absent in old payloads → 0 (backward compatible). The producer's initial publish leaves it 0; do not change the create→publish path otherwise.
- **Retry decision (worker)**: when `Service.Process` returns a non-nil error:
  - if `msg.RetryCount < maxRetries` → re-publish the same message with `RetryCount+1` onto the **processing** queue;
  - else → publish to the **dead-letter** queue `orders:dead-letter` and log at error with `order_id` + final attempt count.
  - On `Process == nil` (PAID, FAILED, ignored/non-CREATED) → done, no re-publish.
- **`maxRetries`**: config `ORDER_MAX_RETRIES`, default **3** (meaning up to 3 *re-tries* after the first attempt; state the exact semantics you choose in PHASE-RESULT and keep code + docs consistent). Validate ≥ 0.
- **Ports**: introduce a minimal interface the worker depends on for re-publish + dead-letter (e.g. `RetryQueue { Requeue(ctx, msg) error; DeadLetter(ctx, msg) error }`), with an in-memory fake for unit tests. Wire a Redis implementation in `internal/queue` (LPUSH onto `orders:processing` / `orders:dead-letter`). Keep dependency direction inward: domain defines the port, `queue` implements it.
- **Worker wiring** (`cmd/worker`): construct and inject the Redis retry/dead-letter queue.
- **Docs**: README "Resilience" subsection — retry semantics, the cap, `orders:dead-letter`, how to inspect it (`redis-cli LRANGE orders:dead-letter 0 -1`), and the explicit note that payment-declined (FAILED) orders are **not** retried.

## Optional (stretch) — Outbox pattern

Architecture §20/§6 list a transactional **outbox** as an optional improvement (closes the tiny window where the order commits but the post-commit publish fails, leaving an order stuck CREATED with no queue message). It is **OPTIONAL** in this phase and must not jeopardize the required retry/dead-letter work.

- If attempted: `outbox_events` table written **in the same transaction** as the order insert; a publisher (poll loop) reads unpublished rows, publishes to Redis, marks them published; publishing is **idempotent** (at-least-once; the worker already no-ops non-CREATED orders, so duplicate delivery is safe). Add migration + tests. Document it as its own slice.
- If not attempted: say so in PHASE-RESULT and carry it forward. Do **not** half-build it.

## Entry condition

Root repo through phase 9; default `go test ./...` green; worker currently drops messages on `Process` error.

## Exit condition

- A transient `Process` failure causes re-enqueue with incremented `retry_count`; after the cap the message lands in `orders:dead-letter` and is not reprocessed.
- A payment-declined order is FAILED exactly once, never retried, never dead-lettered.
- Default `go test ./...` green; live `go test -tags=integration ./...` green, including a dead-letter assertion.
- One documented command brings the stack up; dead-letter contents inspectable.

## Must-exist checklist

- [ ] `OrderCreatedMessage.RetryCount` (`retry_count`), backward compatible (missing → 0).
- [ ] Worker retries on `Process` error up to `maxRetries`, then dead-letters; never retries on `Process == nil`.
- [ ] `orders:dead-letter` constant + Redis Requeue/DeadLetter implementation in `internal/queue`.
- [ ] Worker depends on a domain-defined retry/dead-letter port with an in-memory fake.
- [ ] `ORDER_MAX_RETRIES` config (default 3), validated.
- [ ] Unit tests: retry increments count; cap routes to dead-letter; payment decline is NOT retried; `Process==nil` never re-publishes.
- [ ] Live integration: a transient failure exhausts retries → message in `orders:dead-letter`.
- [ ] README Resilience subsection (incl. "declines are not retried").
- [ ] Default `go test ./...` green; live `-tags=integration` green.

## Must-NOT-exist checklist

- [ ] No retry of payment-declined (FAILED) orders. No dead-lettering of terminal business outcomes.
- [ ] No unbounded retry / hot-loop with no cap.
- [ ] No change to the create→commit→publish ordering (BR-ORD-006 / §20) — the producer still publishes only after commit.
- [ ] No new external dependency (use the existing go-redis client).
- [ ] No half-finished outbox left in the tree (all-or-clearly-deferred).
- [ ] No business-logic change to status transitions, totals, or the payment rule.
- [ ] No `TODO`/`FIXME`/debug prints; no secrets.

## Positive verification

- Unit: fake processor/repo that returns a transient error on attempt 1 and succeeds on a later attempt → order eventually PAID, `retry_count` incremented across attempts, nothing dead-lettered.
- Unit: always-transient-error → after `maxRetries`, exactly one dead-letter publish, no further reprocessing.
- Unit: payment decline → FAILED once, zero Requeue, zero DeadLetter calls.
- Unit: old payload without `retry_count` decodes to `RetryCount == 0`.
- Live: inject a transient failure (documented mechanism), observe the message reach `orders:dead-letter` (`LRANGE`), with the order not left mid-flight inconsistently.

## Negative verification

- `maxRetries = 0` → first transient failure dead-letters immediately (no retry).
- A message at the cap is not requeued again (no off-by-one re-enqueue).
- Worker shutdown mid-retry does not lose the in-flight message beyond at-most-once semantics already documented (state the guarantee).

## Known-gap notes (carry forward, not fixed here)

- If outbox is deferred: the post-commit publish window remains (commit-then-publish can drop the initial event on a Redis outage) — note it.
- No retry backoff/delay (immediate re-enqueue) unless you add one; if added, keep it simple and documented.
- Stdlib govulncheck advisories need a Go toolchain patch bump (separate ops task).

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
