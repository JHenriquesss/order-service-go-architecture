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
- **No new dependency** — retry/dead-letter uses the existing `go-redis` client and stdlib. No `replace` to local paths.
- Build reproducible from a clean checkout with the documented commands.

## Formatting & naming

- `gofmt` clean, mandatory. `goimports` grouping if used.
- Package names short, lowercase, meaningful. No `utils`/`common`/`helpers`/`misc`. No stutter.
- Initialisms consistent: `ID`, `URL`, `HTTP`, `JSON`, `SQL`, `UUID`, `TTL`.
- Names reveal intent; tests named as behavior (`TestWorkerDeadLettersAfterMaxRetries`, `TestPaymentDeclineIsNotRetried`).

## Errors

- Return errors for fallible ops; `panic` only for programmer bugs / impossible invariants.
- Wrap with `fmt.Errorf("context: %w", err)` when callers need `errors.Is`/`errors.As`.
- No string-matching on errors to decide retry. The retry trigger is structural: `Service.Process` returned non-nil. Do not sniff error text.
- No `log.Fatal`/`os.Exit` outside `main`/command composition.
- A failure to **re-publish or dead-letter** is itself an error to log loudly — do not silently swallow it (that re-drops the message you were trying to save).

## Panic / leftovers

- No `panic("TODO")`, `TODO`, `FIXME`, debug `fmt.Println`, leftover prints.
- `recover` only at process/framework boundaries, never as business control flow.

## Architectural boundaries (architecture.md wins)

- Handler → Service → Repository → DB. Dependency direction inward.
- The **retry/dead-letter port is defined in the `order` domain**; `internal/queue` implements it against Redis. The worker depends on the interface, not on Redis types. Unit tests use an in-memory fake.
- Retry orchestration lives in the **worker** (the consumer loop), not inside `Service.Process`. `Process` keeps its current contract: nil for terminal outcomes (PAID/FAILED/ignored), non-nil for transient/infra failure. Do **not** move retry counting into the service or the repository.
- No business rule (status machine, totals, payment decision) changes. The payment-decline → FAILED → `nil` return is load-bearing for the retry distinction; preserve it exactly.

## Types

- `RetryCount` is a plain `int` field on the message with a json tag; missing → 0. Don't introduce a new wrapper type for a single counter.
- Queue/list names are named constants (`OrderQueueName`, new `OrderDeadLetterQueueName = "orders:dead-letter"`), not string literals scattered around.
- Defined types for domain identifiers/states stay as they are.

## Testing

- TDD: red → green → refactor. Tests encode the phase goal (bounded retry + dead-letter + no-retry-on-decline), not the implementation.
- Drive retries through a **fake processor/repo** that fails transiently a controlled number of times; assert attempts, final status, requeue count, dead-letter count. Use a fake retry queue that records calls.
- Table tests for the cap boundary (`maxRetries` = 0, 1, 3).
- Unit tests run with **no external services**.
- Live Redis assertions (dead-letter contents) gated behind `//go:build integration`; fail loudly (not `t.Skip`) when the tag is set and services are down.
- Test the payment-decline-not-retried case explicitly — it is the easiest thing to get wrong.

## Phase-specific emphasis: resilience

- The defining correctness property is the **two-failures distinction** (transient → retry/dead-letter; decline → terminal FAILED, untouched). A green build that retries declines fails the phase regardless of other gates.
- Bounded always: every retry path terminates at the cap in `orders:dead-letter`. No infinite requeue.
- Surgical: message struct + worker loop + a queue impl + config + docs + migration *only if* outbox attempted. Do not touch order create/total/status logic.
- A re-publish/dead-letter failure must be surfaced (logged), never silently dropped.
- If outbox is attempted it must be transactional and idempotent or not present at all — no partial outbox.
