# QUALITY-GATES — Phase 6 (Worker, Lifecycle, Metrics)

Concurrency-critical phase. Race + cancellation evidence is mandatory. Record actuals in PHASE-RESULT.

## Build & static

| Gate | Threshold |
|------|-----------|
| `go build ./...` | passes |
| `gofmt -l .` | prints nothing |
| `go vet ./...` | clean |
| `go mod tidy` | no diff |
| `staticcheck` / `golangci-lint` | clean when available; document if not |

## Tests & coverage

| Gate | Threshold |
|------|-----------|
| `go test ./...` | all pass, no DB/Redis |
| `go test -race ./...` | **passes — required** (concurrency) |
| `go test -shuffle=on ./...` | passes (no inter-test coupling) |
| Coverage on `internal/worker` + `internal/metrics` + lifecycle | ≥ 85% |
| Cancellation + graceful-shutdown | tested deterministically (no sleeps) |
| BR-ORD-007..010 + transition rejections | each mapped to a test |

## Concurrency

| Gate | Threshold |
|------|-----------|
| Data races | none (`-race` clean) |
| Goroutine leaks | none; pool stops on cancel (test asserts shutdown) |
| `time.Sleep` for synchronization | none |
| Worker count | bounded, from config |
| Metrics counters | concurrency-safe (atomic/mutex), verified under `-race` |

## Structure & complexity

| Gate | Threshold |
|------|-----------|
| Cyclomatic complexity per function | ≤ 10 |
| Function length | ≤ 60 lines |
| File length | ≤ 400 lines |
| Business rules location | service/domain, not worker plumbing or handler |
| Import direction | no cycles; metrics has no business deps |

## Evidence required in PHASE-RESULT.md

- `-race` run output summary.
- How cancellation + shutdown are tested without sleeps.
- Coverage for worker + metrics.
- `/metrics` sample output matching architecture §22.
