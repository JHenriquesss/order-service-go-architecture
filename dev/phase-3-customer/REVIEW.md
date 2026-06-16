# REVIEW — Phase 3 (Customer)

Caveman review. Reviewer = integrating LLM.

## Score as delivered: 95/100

Honest, matches self-score.

## What good

- All must-exist present, all must-not-exist absent (read + grep verified).
- Gates re-run by me (go1.23.10 + staticcheck): build ok, gofmt clean, vet clean, **staticcheck clean**, `go test ./...` pass. Coverage `internal/customer` 86.9% (>85% gate).
- Business rules in service (name required, document unique incl. update-exclude-self, soft deactivate, bounded pagination Default 20/Max 100). Handler thin: decode/parse/call/map. Repo is a port + RWMutex in-mem fake, filtering in Go (no SQL concat).
- Clean not-found path: repo `ErrNotFound` sentinel → `mapRepoErr` → `RESOURCE_NOT_FOUND`. Duplicate → 409. Validation → 400. Unauthenticated → 401.
- `CustomerFilter.Active *bool` distinguishes unset from false. `Page[T]` generic per §17.

## Gaps found (real)

- None blocking. BR-CUS-003 (inactive customer can't order) correctly deferred to phase 5 (no order subsystem here) — documented.

## Env-deferred (not code defects)

- `-race`: no gcc/cgo in box. Shared state = RWMutex-guarded in-mem repo only.
- govulncheck/golangci-lint: I run govulncheck at root.

## Integration notes (root, order-service-go + chi)

- Module rename: `order-service-customer` → `order-service-go` in all copied files.
- Copy `internal/customer/*`. DROP phase-3 clean-room `internal/auth`, `internal/errors`, `internal/server/router.go` (root has real auth middleware + richer errors).
- Handler uses stdlib ServeMux (`mux.HandleFunc("GET /api/customers/{id}")`, `r.PathValue`). Rewrite to chi `Routes() http.Handler` + `chi.URLParam` to match root.
- Wire `/api/customers` in root router behind `middleware.Authenticator`; POST→`RequireAction(ActionCreateCustomers)`, PUT + PATCH-deactivate→`RequireAction(ActionUpdateCustomers)`, GET list/id→authenticated (both roles read; §15 grants create/update to both).
- errors used (Validation/Duplicate/NotFound/Internal) all exist in root.
- `Page[T]` lives in customer pkg; dedup with product at phase-4 integration.

## Final score after fix: 100/100 (runnable gates + staticcheck + govulncheck clean). `-race` deferred (gcc).
