# QUALITY-GATES — Phase 10 (Retry + Dead-Letter)

Mandatory; record actuals in PHASE-RESULT. This phase's defining gate: a **transient** processing failure is retried up to the cap and then lands in `orders:dead-letter`, while a **payment-declined** order is FAILED once and never retried.

## Build & static (root repo)

| Gate | Threshold |
|------|-----------|
| `go build ./...` | passes |
| `gofmt -l .` | prints nothing |
| `go vet ./...` | clean |
| `go vet -tags=integration ./tests/integration/` | clean |
| `staticcheck ./...` | clean when available |
| `go mod tidy` | no diff; no new dependency |

## Default test suite (no services)

| Gate | Threshold |
|------|-----------|
| `go test ./...` | all pass, integration excluded |
| `go test -race ./...` | passes when CGO/gcc available; else document concrete blocker |
| `internal/worker` coverage | ≥ 85% |

## Retry / dead-letter correctness (unit — the heart of the phase)

| Gate | Threshold |
|------|-----------|
| Transient error, then success | order reaches PAID; `retry_count` incremented; zero dead-letter |
| Always-transient error | exactly 1 dead-letter publish after cap; no further reprocessing |
| Payment decline | FAILED once; **0** Requeue, **0** DeadLetter |
| `Process == nil` (PAID/FAILED/non-CREATED) | never re-publishes |
| Old payload (no `retry_count`) | decodes to `RetryCount == 0` |
| `maxRetries = 0` | first transient failure dead-letters immediately |
| Cap boundary | no off-by-one extra requeue |

## Live integration (the point of this phase)

| Gate | Threshold |
|------|-----------|
| Stack up incl. dead-letter path | one documented command |
| Transient failure exhausts retries | message present in `orders:dead-letter` (`LRANGE`) |
| Declined order | FAILED, absent from `orders:dead-letter` |
| `go test -tags=integration ./...` (live) | passes |
| Default `go test ./...` (no tag) | still green |

## DX / docs

| Gate | Threshold |
|------|-----------|
| `ORDER_MAX_RETRIES` config (default 3) | present, validated, documented |
| README Resilience subsection | retry cap, `orders:dead-letter`, inspect cmd, "declines not retried" |
| Domain behavior (status/totals/payment rule) | unchanged |

## Optional outbox (only if attempted)

| Gate | Threshold |
|------|-----------|
| `outbox_events` written in the order tx | atomic with the insert |
| Publisher idempotent / at-least-once | duplicate delivery safe (worker no-ops non-CREATED) |
| Migration + tests | present, green |
| If deferred | explicitly stated in PHASE-RESULT; nothing half-built in tree |

## Evidence required in PHASE-RESULT.md

- The exact retry semantics chosen (does "3" mean 3 total attempts or 1 + 3 retries) — code and docs consistent.
- Unit evidence for each row in the retry/dead-letter table (test names).
- Live evidence: `LRANGE orders:dead-letter 0 -1` output showing the dead-lettered message with its final `retry_count`.
- Explicit statement that payment declines are not retried (with the test that proves it).
- Whether outbox was done or deferred, and the consequence either way.
- Confirmation default `go test ./...` (+ `-race` if runnable) and live integration still green.
