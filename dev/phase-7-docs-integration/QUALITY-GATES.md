# QUALITY-GATES — Phase 7 (Docs + Integration)

Whole-system verification. Record actuals in PHASE-RESULT.

## Build & static

| Gate | Threshold |
|------|-----------|
| `go build ./...` | passes |
| `gofmt -l .` | prints nothing |
| `go vet ./...` | clean |
| `go mod tidy` | no diff |

## Tests

| Gate | Threshold |
|------|-----------|
| Default `go test ./...` (no tag) | passes, integration tests skipped |
| `go test -tags=integration ./...` (live Postgres + Redis) | passes |
| End-to-end happy path → `PAID` | asserted on real persisted state |
| End-to-end failure path → `FAILED` | asserted |
| Repository (Postgres) + queue (Redis) round-trips | pass under tag |

## Docs / contract

| Gate | Threshold |
|------|-----------|
| OpenAPI covers every auth/customer/product/order endpoint | yes |
| Error schema documented (architecture §24) | yes |
| JWT bearer auth documented | yes |
| Swagger schemas match real request/response shapes | verified |
| README has all §33 sections + Swagger URL | yes |
| API behavior changed by this phase | none |

## Evidence required in PHASE-RESULT.md

- Confirmation default test run skips integration tests.
- Integration run command + services used + result.
- Any documented schema/code mismatch found and how it was resolved.
- Note that the live integration run is executed at root integration time (services required).
