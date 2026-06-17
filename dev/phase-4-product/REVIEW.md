# REVIEW — Phase 4 (Product)

Caveman review. Reviewer = integrating LLM.

## Score as delivered: 95/100

Honest, matches self-score. No code defects.

## What good

- All must-exist present, all must-not-exist absent (read + grep verified).
- Gates re-run by me (go1.23.10 + staticcheck): build, gofmt, vet, **staticcheck**, tests all clean. Coverage `internal/product` 87.0% (>85% gate).
- **`internal/money`**: int64-cents `Money`, no float64. `ParseString` rejects >2 frac digits / empty / multi-dot; strips sign correctly; JSON marshals as a bare number (matches §12). Overflow safe for numeric(15,2).
- Product service rules: name required (BR-PRD-001), unique SKU incl. update-exclude-self (BR-PRD-002), price>0 (BR-PRD-003). `FindManyByID` + price exposure ready for order copy (BR-PRD-005). Soft deactivate. Bounded pagination.
- Handler thin; repo port + RWMutex in-mem fake, in-Go filtering (no SQL concat).
- BR-PRD-004/005 correctly deferred to phase 5 (no order subsystem here) — documented.

## Gaps found (real)

- None blocking.

## Env-deferred (not code defects)

- `-race`: no gcc/cgo. Shared state = RWMutex-guarded in-mem repo.
- govulncheck/golangci-lint: I run govulncheck at root.

## Integration notes (root, order-service-go + chi)

- Module rename `order-service-product` → `order-service-go`.
- **Extract shared `internal/pagination.Page[T]`**: customer + product (+ orders soon) all define identical `Page[T]`. Move to one package, point both at it. Debt-free per myrules.
- Copy `internal/money` to root (no internal imports — clean). Reused by orders phase 5.
- Copy `internal/product/*`; rewrite money import path; use `pagination.Page`.
- Handler ServeMux → chi `Routes()` + `chi.URLParam` (same as customer integration).
- Wire `/api/products` behind `Authenticator` + `RequireAnyRole(ADMIN,OPERATOR)` per §15.
- Drop phase clean-room auth/errors/server stand-ins.
- Port handler-level tests to `internal/product/handler_test.go` (coverage phase drove via its server tests).

## Final score after fix: 100/100 (runnable gates + staticcheck + govulncheck clean). `-race` deferred (gcc).
