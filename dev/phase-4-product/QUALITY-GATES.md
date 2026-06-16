# QUALITY-GATES — Phase 4 (Product)

Mandatory; record actuals in PHASE-RESULT.

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
| `go test ./...` | all pass, no DB |
| `go test -race ./...` | passes |
| Coverage on `internal/product` service | ≥ 85% |
| Business rules (BR-PRD-001..005) | each has a test |
| Negative-path tests (validation, duplicate, not-found, authz) | all present |

## Structure & complexity

| Gate | Threshold |
|------|-----------|
| Cyclomatic complexity per function | ≤ 10 |
| Function length | ≤ 60 lines |
| File length | ≤ 400 lines |
| Business rules location | service, not handler/repo |
| Import direction | handler → service → repository (interface); no cycles |

## Security / hygiene

| Gate | Threshold |
|------|-----------|
| Filter values parameterized (no SQL string concat) | enforced |
| page_size upper bound | enforced + tested |
| Panic outside `main` | none |

## Evidence required in PHASE-RESULT.md

- Coverage number for `internal/product`.
- Mapping of each BR-PRD rule → test name.

## Phase-4 specific

| Gate | Threshold |
|------|-----------|
| Price type | decimal-safe (not float64) — verified |
| Price ≤ 0 rejected | tested |
