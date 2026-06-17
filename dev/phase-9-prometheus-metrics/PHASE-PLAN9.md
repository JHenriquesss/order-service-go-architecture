# PHASE-PLAN9 — Prometheus Metrics (cross-process observability)

## Goal

Replace the per-process in-memory metrics counters with the **Prometheus client library** (architecture §22 "Future improvement"), so the `worker`'s `orders_processed_total` / `orders_failed_total` become observable instead of dying inside a process with no HTTP endpoint. After this phase, **both** `api` and `worker` expose Prometheus exposition at `/metrics`, and aggregation across processes is done by Prometheus at scrape/query time.

> Like phase 8, this phase operates on the **already-integrated root repository** (module `order-service-go`), not a clean-room slice. Standalone-module rules from AGENTS.md still apply to any in-folder spike, but the real target is the root repo.

## Why this is needed (the actual gap)

- `api` increments `orders_created_total` and exposes `/metrics`; `worker` increments `processed`/`failed` into its **own** `metrics.Collector` that nothing ever serves. Those counters are invisible. Phase 8 e2e had to assert `orders_created_total` (api side) for this reason.
- Two processes cannot share an in-memory counter. Correct fix is pull-based: each process exposes its own `/metrics`; Prometheus scrapes both jobs and sums them.

## Scope (build/verify exactly this)

- Add dependency `github.com/prometheus/client_golang/prometheus` (+ `.../promhttp`, `.../prometheus/testutil` for tests). No other new deps.
- Rework `internal/metrics`:
  - `Collector` keeps its current **method set** unchanged so callers are untouched: `IncOrdersCreated()`, `IncOrdersProcessed()`, `IncOrdersFailed()`, `RecordProcessingDuration(seconds float64)`.
  - Internals become Prometheus collectors registered on a **private `*prometheus.Registry`** (no global default registry — avoids double-register panics and keeps construction testable):
    - `orders_created_total`, `orders_processed_total`, `orders_failed_total` → `prometheus.Counter`.
    - `orders_processing_duration_seconds` → `prometheus.Histogram` (yields `_sum`, `_count`, `_bucket`).
  - Expose `Handler() http.Handler` via `promhttp.HandlerFor(reg, …)`.
  - `metrics.Handler.Register(mux)` keeps the same signature and still mounts `GET /metrics`, now serving Prometheus exposition.
- `cmd/api`: serve metrics via the Prometheus handler (router wiring stays; only the bytes change). Keep a single shared `Collector` passed into the order service (as today).
- `cmd/worker`: give the worker a **minimal HTTP server** serving `GET /metrics` (Prometheus) and `GET /health`, started alongside the pool and gracefully shut down with it. The worker's order service uses the **same** `Collector` instance that backs that endpoint (fix the current `metrics.NewCollector()` throwaway). Add a `METRICS_PORT` (or reuse an existing config field) with a sane default; document it.
- `docker-compose.yml`: add a dev `prometheus` service with a committed `prometheus.yml` scraping the `api` and `worker` jobs. Dev-only; no Grafana required.
- README: short "Metrics" section — what each process exposes, the scrape setup, and the PromQL for `_avg` (since average is now derived, not exposed).

## Deliberate behavior change (must be flagged, not hidden)

- The custom plain-text `RenderText` format (incl. `orders_processing_duration_seconds_avg`) is **replaced** by standard Prometheus exposition. `_avg` is **not** a native Prometheus series; it is computed downstream:
  `rate(orders_processing_duration_seconds_sum[5m]) / rate(orders_processing_duration_seconds_count[5m])`.
- `_sum`, `_count`, and the three `_total` counters remain present (Histogram + Counters), so §22's required series — except the derived `_avg` — are still exposed by name.
- This is the one place phase-1..8 observable output changes on purpose. Any unit/integration test that string-matched the old format must be updated to parse Prometheus exposition. Do **not** change order/worker domain behavior to accommodate it.

## Entry condition

Root repo through phase 8; default `go test ./...` green; `/metrics` currently serves the in-memory text format.

## Exit condition

- `api` and `worker` both expose `/metrics` in Prometheus format.
- Creating + processing an order increments the right series in each process; a live scrape of the worker job shows `orders_processed_total > 0`.
- Default `go test ./...` green; live `go test -tags=integration ./...` green.
- One documented command brings the stack up with Prometheus scraping both jobs.

## Must-exist checklist

- [ ] `internal/metrics` backed by `prometheus/client_golang`; `Collector` method set unchanged; private registry.
- [ ] Series exposed by name: `orders_created_total`, `orders_processed_total`, `orders_failed_total`, `orders_processing_duration_seconds_{sum,count,bucket}`.
- [ ] `cmd/worker` serves `GET /metrics` + `GET /health`, sharing the order service's `Collector`; graceful shutdown.
- [ ] `cmd/api` `/metrics` serves Prometheus exposition (same shared collector).
- [ ] `docker-compose.yml` `prometheus` service + committed `prometheus.yml` scraping api + worker.
- [ ] Unit tests assert counter/histogram increments via `prometheus/testutil` (`ToFloat64` / `CollectAndCompare`).
- [ ] README "Metrics" section incl. the `_avg` PromQL.
- [ ] Default `go test ./...` green; live `-tags=integration` green.

## Must-NOT-exist checklist

- [ ] No use of the global default Prometheus registry / `promauto` default (avoid cross-test double-register panics).
- [ ] No second, parallel metrics path left behind (old `RenderText`/`Snapshot` removed if unused — remove only the metrics code your change orphans).
- [ ] No domain-logic change in `order`/`worker` to suit metrics.
- [ ] No new dependency beyond `prometheus/client_golang`.
- [ ] No secrets committed; Prometheus dev config only.
- [ ] No `TODO`/`FIXME`/debug prints.

## Positive verification

- `metrics_test.go`: after N `IncOrdersCreated`/`Processed`/`Failed` and K `RecordProcessingDuration`, `testutil.ToFloat64` returns N for each counter and the histogram `_count` == K, `_sum` == sum of samples.
- `Handler()` output contains `# TYPE orders_created_total counter` and the three counter names + histogram series.
- Worker HTTP: `GET /metrics` 200 Prometheus body; `GET /health` 200.
- Live (integration): create + process an order; scrape worker `/metrics` (or assert via the worker job through Prometheus) shows `orders_processed_total >= 1` and `orders_failed_total` for the failure path.
- `docker compose up` brings Prometheus up; both targets are `up` on its `/targets`.

## Negative verification

- Building two `Collector`s in one process does **not** panic (proves private, non-global registries).
- Histogram with zero observations: `_count` 0, `_sum` 0; no divide-by-zero anywhere (avg is downstream now).
- Worker metrics server shuts down cleanly on SIGINT/SIGTERM (no leaked goroutine / port).

## Known-gap notes (carry forward, not fixed here)

- Retry / dead-letter / outbox resilience is **Phase 10**.
- Stdlib govulncheck advisories need a Go toolchain patch bump (separate ops task).
- No alerting rules / Grafana dashboards — out of scope (dev scrape only).

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
