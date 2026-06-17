# REVIEW — Phase 6 (Worker, Lifecycle, Metrics)

Caveman review. Reviewer = integrating LLM. Concurrency-critical.

## Score as delivered: 88/100

Honest. No code defects; one staticcheck nit (their env lacked it).

## What good

- All must-exist present, must-not-exist absent.
- Gates re-run by me (gcc now available): build, gofmt, vet pass; **`go test -race -shuffle=on ./...` RACE-CLEAN** across all packages — the gate that matters most here, now actually executed.
- Worker: bounded pool (count from config), context-aware `Dequeue`, drains in-flight on shutdown (`Process` uses `context.Background` so cancel stops dequeue only; `wg.Wait` joins). No detached goroutines, no sleep-sync.
- Metrics: `atomic.Int64` counters + mutex-guarded duration; `RenderText` §22 format; avg guards divide-by-zero. Concurrency-safe under `-race`.
- Lifecycle: `Process` CREATED→PROCESSING→PAID|FAILED (ignores non-CREATED, BR-ORD-009; failure→FAILED+metric+log, BR-ORD-010); `Cancel` CREATED-only (BR-ORD-007); `Ship` PAID-only (BR-ORD-008); invalid→INVALID_ORDER_STATUS/400. Uses `CanTransition`.
- Repository superset: adds `UpdateStatus`/`MarkProcessed`. Order pkg is a clean superset of phase 5's.

## Gaps found

- **staticcheck**: `internal/order/handler_test.go` has unused `fakeVerifier`/`Verify` (U1000). Dead code — remove at integration (handler_test rewritten anyway).
- `List` N+1 (carried from phase 5). In-memory, acceptable.

## Architectural gap (NOT phase-6's fault — my phase design) — needs a decision before phase 7

No phase built **SQL (pgx) repositories**; all repos are in-memory. `cmd/api` and `cmd/worker` are separate processes — with in-memory repos they cannot share order state, so a real api→Redis→worker→PAID round-trip can't work cross-process. Phase 7 (integration tests) assumes end-to-end works. Resolution options raised to user separately.

## Integration notes (root)

- Replace root `internal/order` with phase-6 superset (module rename, Page→pagination, money path). Rewrite handler → chi + `middleware.UserIDFromContext` (+ cancel/ship); rewrite handler_test (drop dead `fakeVerifier`).
- Copy `internal/metrics`, `internal/worker` (module rename).
- `order.NewService` now 7 args (+ `Metrics`, processLog) — update api main + server_test.
- Add real Redis consumer (`internal/queue`, BRPOP) implementing `order.OrderQueue`; wire `cmd/worker` real (redis client + consumer + pool + simulated processor + shared metrics).
- Mount `GET /metrics` (infra endpoint, unauthenticated like /health; note §15 lists ADMIN — documented deviation). Cancel/ship ride existing `/api/orders` mount.

## Final score after fix: 100/100 (runnable gates + staticcheck + govulncheck + **-race** clean).
