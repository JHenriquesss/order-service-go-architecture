# QUALITY-GATES — Phase 9 (Prometheus Metrics)

Mandatory; record actuals in PHASE-RESULT. This phase's defining gate is that **the worker's `orders_processed_total` is observable** — proven via a Prometheus-format scrape, not just incremented in memory.

## Build & static (root repo)

| Gate | Threshold |
|------|-----------|
| `go build ./...` | passes |
| `gofmt -l .` | prints nothing |
| `go vet ./...` | clean |
| `go vet -tags=integration ./tests/integration/` | clean |
| `staticcheck ./...` | clean when available |
| `go mod tidy` | no diff; only `prometheus/client_golang` (+ its transitive deps) added |

## Default test suite (no services)

| Gate | Threshold |
|------|-----------|
| `go test ./...` | all pass, integration excluded |
| `go test -race ./...` | passes when CGO/gcc available; else document concrete blocker |
| `internal/metrics` coverage | ≥ 85% (small package, mostly testable) |

## Metrics correctness (unit, via prometheus/testutil)

| Gate | Threshold |
|------|-----------|
| Counters increment | `ToFloat64(created/processed/failed)` equals call count |
| Histogram | `_count` == #observations, `_sum` == Σ samples |
| Series names present | `orders_created_total`, `orders_processed_total`, `orders_failed_total`, `orders_processing_duration_seconds_{sum,count,bucket}` |
| No global registry | building 2 collectors in one process does not panic |
| Handler body | Prometheus exposition (`# TYPE` lines), 200 |

## Cross-process / live (the point of this phase)

| Gate | Threshold |
|------|-----------|
| Worker serves `/metrics` + `/health` | 200; Prometheus body on metrics |
| Worker graceful shutdown | metrics server stops with the pool; no leaked port/goroutine |
| Stack up incl. Prometheus | one documented command |
| `go test -tags=integration ./...` (live) | passes |
| Live: order processed | worker job shows `orders_processed_total >= 1` |
| Live: failure path | worker job shows `orders_failed_total >= 1` |
| Prometheus targets | api + worker both `up` |

## DX / docs

| Gate | Threshold |
|------|-----------|
| `docker-compose.yml` prometheus service + `prometheus.yml` | present, scrapes api + worker |
| README "Metrics" section | accurate; includes `_avg` PromQL |
| API/worker domain behavior vs phases 1-8 | unchanged (only metrics output format changes, by design) |

## Evidence required in PHASE-RESULT.md

- The exposition bodies (or representative excerpts) from `api` and `worker` `/metrics`.
- Live evidence the worker job shows non-zero `orders_processed_total` (scrape output or Prometheus query).
- Which old code was removed (`RenderText`/`Snapshot`) and why it was safe.
- Confirmation default `go test ./...` (+ `-race` if runnable) and live integration still green.
- The one intentional behavior change (`_avg` now derived) stated explicitly.
