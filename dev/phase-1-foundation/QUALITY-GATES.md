# QUALITY-GATES — Phase 1 (Foundation)

Measurable, mandatory acceptance criteria. The phase is **not** complete unless these pass or a failure is explicitly justified in `PHASE-RESULT.md`. Record actual numbers in the PHASE-RESULT table.

## Build & static

| Gate | Threshold |
|------|-----------|
| `go build ./...` | passes |
| `gofmt -l .` | prints nothing |
| `go vet ./...` | clean (or justified, documented exception) |
| `go mod tidy` | no resulting diff |
| `staticcheck ./...` / `golangci-lint run` | clean when available; document if not installed |

## Tests & coverage

| Gate | Threshold |
|------|-----------|
| `go test ./...` | all pass, no external services |
| `go test -race ./...` | passes (recovery/middleware) |
| Coverage (`go tool cover -func`) on `internal/config`, `internal/errors` | ≥ 80% |
| Overall coverage | ≥ 60% (foundation has thin glue code) |

## Structure & complexity

| Gate | Threshold |
|------|-----------|
| Cyclomatic complexity per function (gocyclo / gocognit) | ≤ 10 |
| Function length | ≤ 60 lines |
| Go file length | ≤ 400 lines |
| Package import direction | no cycles; `internal/*` does not import `cmd/*` |
| Dependency count added | minimal; each justified in PHASE-RESULT |

## Security / hygiene

| Gate | Threshold |
|------|-----------|
| Secrets in source | none (config from env only) |
| `govulncheck ./...` | clean when available; else documented |
| Panic outside `main` | none |

## Evidence required in PHASE-RESULT.md

- Go version + `go env` snapshot.
- Each command above: run / passed / failed (+reason/impact/fix).
- Coverage numbers for the named packages.
- Justification for every added dependency.
