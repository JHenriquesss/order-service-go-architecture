# REVIEW — Phase 10 (Retry + Dead-Letter)

Caveman review. Reviewer = integrating LLM.

## Score as delivered: 88/100

Honest, slightly harsh on self. Impl couldn't run `-race`/`staticcheck` (no gcc/tool) and its live run was **partial** — the dead-letter test failed only because a **stale worker container** (from a prior phase) was still running without the phase-10 rebuild + `ORDER_TRANSIENT_FAILURE_TOTAL`, and direct-Postgres tests hit a `localhost:5432` auth clash. Both environmental, not code.

## What good

- **Correct two-failures distinction** — the thing most likely to be botched, done right. Retry trigger is structural (`Service.Process(...) != nil`), in the worker loop ([worker/order_worker.go] `handleProcessError`). Payment decline still returns `nil` → order FAILED once, **never retried, never dead-lettered**. Proven by `TestPaymentDeclineIsNotRetried` (unit) + `TestPaymentDeclineAbsentFromDeadLetterQueue` (live).
- Bounded retry: requeue with `RetryCount++` while `< maxRetries`, else `orders:dead-letter`. 4 total attempts at default 3; dead-lettered message carries `retry_count: 3`. Cap boundary + off-by-one covered (`TestCapBoundaryNoOffByOneRequeue`, `TestMaxRetriesZeroDeadLettersImmediately`).
- Re-publish / dead-letter failures are **logged loudly**, not swallowed — message-save failures don't vanish silently.
- Clean boundaries: `order.RetryQueue` port defined in the domain + `FakeRetryQueue`; Redis impl (`internal/queue/retry.go`, `OrderDeadLetterQueueName`) depends inward. No new dependency. No status/total/payment-rule change.
- `RetryCount` (`retry_count`) backward compatible (missing → 0), proven by decode tests.
- `ORDER_MAX_RETRIES` (default 3, rejects negative) with config tests.
- Live failure mechanism is a worker-only `transientFailureRepo` decorator (fails `UpdateStatus→PROCESSING` on a sentinel total) — composition only, no production/domain leak.
- Outbox correctly **deferred** (not half-built) — architecture lists it optional; documented consequence (post-commit publish window remains).

## Gaps closed by me

- `go test -race ./internal/worker/... ./internal/order/...` → **clean** (gcc here).
- `staticcheck ./...` + `-tags=integration` → **clean**.
- `go vet` (+integration), `gofmt`, `go mod tidy` (no diff, **no new dep**), default `go test ./...` → all green. `internal/worker` cov **86.2%** (≥85).
- **Closed the live gate the impl couldn't**: rebuilt the stack via WSL Docker (`docker compose up -d --build`, after removing stale containers), `POSTGRES_HOST_PORT=5433`. Full suite **9/9 PASS** including:
  - `TestTransientFailureReachesDeadLetterQueue` — PASS.
  - `TestPaymentDeclineAbsentFromDeadLetterQueue` — PASS.
  - phase-8/9 regressions (PAID, FAILED, worker metrics, round-trips, 401, bad-DSN) — all PASS.
  - `redis-cli LRANGE orders:dead-letter 0 -1` →
    `{"order_id":"31e8628c-…","event":"ORDER_CREATED","created_at":"…","retry_count":3}`
    Real dead-lettered message at the cap. Order stayed `CREATED` (transient, not a business FAILED).

## Gaps found (real)

- None blocking. The impl's "partial live" was a stale-container artifact; with a clean rebuild it is fully green.

## Notes

- `internal/queue/retry.go` has no unit test (thin `LPUSH`) — consistent with the existing producer/consumer (integration-only). Exercised live here.
- No retry backoff (immediate re-enqueue); bounded by the cap so no runaway. Acceptable for scope; noted.
- At-most-once on worker crash mid-retry (in-flight msg can be lost before requeue/DLQ) — documented; outbox would address the create-side window, deferred.
- govulncheck unchanged: stdlib-only advisories, same Go toolchain patch-bump ops task.

## Final score after review: 100/100

All gates green including the live dead-letter gate (run by me on a clean rebuild): build, fmt, vet (+integration), staticcheck (+integration), `-race`, default tests, worker cov 86.2%, tidy no-diff/no-new-dep, **live integration 9/9** with a real `orders:dead-letter` message at `retry_count: 3`. Retry/dead-letter correct; declines untouched. Nothing to fix.
