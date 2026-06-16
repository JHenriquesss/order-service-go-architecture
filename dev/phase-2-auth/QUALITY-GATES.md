# QUALITY-GATES — Phase 2 (Auth)

Security-sensitive phase — stricter test and coverage requirements. Mandatory; record actuals in PHASE-RESULT.

## Build & static

| Gate | Threshold |
|------|-----------|
| `go build ./...` | passes |
| `gofmt -l .` | prints nothing |
| `go vet ./...` | clean |
| `go mod tidy` | no diff |
| `staticcheck` / `golangci-lint` | clean when available; document if not |
| `gosec ./...` | clean when available (security phase) |
| `govulncheck ./...` | clean when available |

## Tests & coverage

| Gate | Threshold |
|------|-----------|
| `go test ./...` | all pass, no DB |
| `go test -race ./...` | passes |
| Coverage on `internal/auth` (jwt, password, service) | ≥ 90% |
| Failure-path tests (bad creds, expired/tampered/wrong-alg token, inactive user) | all present |
| Authorization matrix (architecture §15) | every cell tested |

## Structure & complexity

| Gate | Threshold |
|------|-----------|
| Cyclomatic complexity per function | ≤ 10 |
| Function length | ≤ 60 lines |
| File length | ≤ 400 lines |
| Business rules location | in service/middleware, not handlers |
| Import direction | handler → service → repository (interface); no cycles |

## Security

| Gate | Threshold |
|------|-----------|
| Password/hash in logs or responses | none |
| Hardcoded secrets | none |
| JWT `alg: none` / alg-confusion accepted | must be rejected (tested) |
| User-enumeration via login error | none (generic message, tested) |

## Evidence required in PHASE-RESULT.md

- Coverage numbers for `internal/auth`.
- Proof each negative auth test exists and passes.
- Confirmation secrets come from config; no secret literals.
