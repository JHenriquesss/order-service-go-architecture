# QUALITY-GATES — Phase 3 (Customer)

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
| Coverage on `internal/customer` service | ≥ 85% |
| Business rules (BR-CUS-001..004) | each has a test |
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

- Coverage number for `internal/customer`.
- Mapping of each BR-CUS rule → test name.
