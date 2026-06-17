# GO-QUALITY-GATE.md (distilled, scoped to this phase)

Engineering control document, not a style preference. Code is **not** done because files exist. It is done when it builds, is formatted, passes vet, has meaningful tests, preserves boundaries, models errors explicitly, and has evidence in `PHASE-RESULT.md`. Full reference: `../../GO-CODE-QUALITY-GATE.md` (not in this folder — these are the rules that matter for this slice).

## Mandatory commands (run; record results in PHASE-RESULT.md)

- `go version` and `go env GOVERSION GOOS GOARCH CGO_ENABLED`
- `test -z "$(gofmt -l .)"` — no unformatted files
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `go test -covermode=atomic -coverprofile=coverage.out ./...` then `go tool cover -func=coverage.out`

A command not run is not evidence. A command that failed and was ignored is negative evidence. If a command can't run, document the concrete blocker (never "tool unavailable").

## Toolchain & modules

- Stable Go, explicit `go` directive in `go.mod`. Document the version used.
- `go.mod` + `go.sum` committed and tidy (`go mod tidy` leaves no diff).
- Only `github.com/prometheus/client_golang` is added (plus its transitive deps). No `replace` to local paths.
- Build reproducible from a clean checkout with the documented commands.

## Formatting & naming

- `gofmt` clean, mandatory. `goimports` grouping if used.
- Package names short, lowercase, meaningful. No `utils`/`common`/`helpers`/`misc`. No stutter.
- Initialisms consistent: `ID`, `URL`, `HTTP`, `JSON`, `SQL`, `UUID`.
- Names reveal intent; tests named as behavior (`TestCollectorCountsOrdersCreated`).

## Errors

- Return errors for fallible ops; `panic` only for programmer bugs / impossible invariants.
- Wrap with `fmt.Errorf("context: %w", err)` when callers need `errors.Is`/`errors.As`.
- No `log.Fatal`/`os.Exit` outside `main`/command composition.
- The worker metrics server returning `http.ErrServerClosed` on shutdown is expected, not an error to surface.

## Panic / leftovers

- No `panic("TODO")`, `TODO`, `FIXME`, debug `fmt.Println`, leftover prints.
- No use of `prometheus.MustRegister` against the global default registry, and no `promauto` defaults — register on an owned `*prometheus.Registry` so tests can build collectors freely without panics.

## Architectural boundaries (architecture.md wins)

- Handler → Service → Repository → DB. Dependency direction inward.
- `metrics` stays a leaf adapter: it owns the registry + collectors and an HTTP handler; it must not import `order`/`worker`. Callers depend on the small `Metrics` port (the `IncOrders*` / `RecordProcessingDuration` method set), not on Prometheus types.
- Domain code (`order`, `worker`) must not learn it is talking to Prometheus. No Prometheus types leak past `internal/metrics`.
- The worker's metrics HTTP server is composition wiring (`cmd/worker`), kept thin.

## Types

- Keep the `Collector` method set stable; do not widen the `Metrics` port to expose Prometheus internals.
- Histogram bucket choice: use a small, documented bucket set appropriate for sub-second→few-second processing (e.g. default `prometheus.DefBuckets` or a stated custom set). Don't over-engineer.

## Testing

- TDD: red → green → refactor. Tests encode the phase goal (observable counters), not the implementation.
- Use `prometheus/client_golang/prometheus/testutil` (`ToFloat64`, `CollectAndCompare`, `CollectAndCount`) — do not scrape the HTTP body and string-match in unit tests except where verifying the handler wiring/exposition itself.
- Unit tests run with **no external services**.
- Live Prometheus/worker-scrape assertions gated behind `//go:build integration`.
- Test the zero-observation histogram and the two-collectors-no-panic cases (negative verification).

## Phase-specific emphasis: Prometheus metrics

- The defining win is **cross-process observability**: the worker must expose its own counters. A green build that still hides worker metrics fails the phase regardless of other gates.
- Exactly one observable behavior change is allowed and must be documented: `/metrics` now serves Prometheus exposition and `_avg` becomes a derived (PromQL) value rather than an exposed series.
- Surgical: infra (compose/prometheus.yml), `internal/metrics` internals, the two `cmd/` mains, and any test that string-matched the old format. Do not touch order/worker domain logic.
- Remove the old in-memory text path (`RenderText`/`Snapshot`) only if your change makes it dead; mention any pre-existing dead code you find rather than deleting it.
- Integration tests must fail loudly (not `t.Skip`) when the tag is set and services are down.
