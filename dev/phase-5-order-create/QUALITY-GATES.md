# QUALITY-GATES — Phase 5 (Order Creation)

Critical business-rule phase (money + transaction boundary). Strictest correctness evidence. Mandatory; record actuals in PHASE-RESULT.

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
| `go test -race ./...` | passes |
| Coverage on `internal/order` service + status | ≥ 90% |
| `status.go` (`CanTransition`, `ParseOrderStatus`) coverage | 100% |
| Total-calculation + price-copy | tested with concrete numbers (architecture §12) |
| Transaction boundary (publish-after-commit, commit-ok/publish-fail) | tested |
| BR-ORD-001..006 + BR-PRD-005 | each mapped to a test |

## Mutation-readiness (critical rules)

| Gate | Threshold |
|------|-----------|
| `CanTransition` and total calc structured for mutation testing | pure functions, no I/O |
| `gremlins` (or documented equivalent) on `status.go` + totals | run when available; else document why not |

## Structure & complexity

| Gate | Threshold |
|------|-----------|
| Cyclomatic complexity per function | ≤ 10 |
| Function length | ≤ 60 lines |
| File length | ≤ 400 lines |
| Money type | decimal-safe (not float64) |
| Business rules location | service/domain, not handler/repo/queue |
| Import direction | handler → service → repository/producer (interfaces); no cycles |

## Evidence required in PHASE-RESULT.md

- Coverage for `internal/order` and `status.go`.
- Worked example: input → computed item totals + order total, matching architecture §12.
- Explicit description of how publish-after-commit and the commit-ok/publish-fail path are tested.
