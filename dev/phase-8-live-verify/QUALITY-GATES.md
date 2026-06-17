# QUALITY-GATES — Phase 8 (Live Verification & DX)

Mandatory; record actuals in PHASE-RESULT. This phase's defining gate is the **live integration run** — it must actually execute, not be documented-as-skipped.

## Build & static (root repo)

| Gate | Threshold |
|------|-----------|
| `go build ./...` | passes |
| `gofmt -l .` | prints nothing |
| `go vet ./...` | clean |
| `go vet -tags=integration ./tests/integration/` | clean |
| `staticcheck ./...` | clean when available |
| `go mod tidy` | no diff |

## Default test suite (no services)

| Gate | Threshold |
|------|-----------|
| `go test ./...` | all pass, integration excluded |
| `go test -race ./...` | passes |

## Live integration (the point of this phase)

| Gate | Threshold |
|------|-----------|
| Stack up from clean (compose + migrate) | one documented command |
| postgres/redis healthchecks | present; api/worker wait for healthy |
| `go test -tags=integration ./...` (live) | **passes** |
| e2e → PAID | asserted on live data |
| failure path → FAILED | asserted on live data |
| repo + queue round-trips | pass |
| services-down / bad-DSN | fail loudly (not skip) |

## DX / docs

| Gate | Threshold |
|------|-----------|
| Makefile targets (migrate/seed/compose/test-integration) | present + working |
| README run + test procedure | accurate, reproducible |
| API behavior changed vs phases 1-7 | none (unit suite unchanged) |

## Evidence required in PHASE-RESULT.md

- `docker`/`compose` versions; the exact up + migrate + test commands.
- Live integration run output summary (pass counts, services).
- Any defect found live + the surgical fix applied.
- Confirmation default `go test ./...` + `-race` still green.
