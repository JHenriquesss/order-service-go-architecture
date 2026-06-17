# dev/ — Isolated Implementation Environments

This directory splits the **order-service-go** build (see `../order-service-go-architecture.md`) into ordered, isolated phases. Each phase folder is a **standalone Go module** implemented clean-room by an implementation LLM that can see **only its own folder**.

## The one prompt

Each implementation LLM receives exactly one prompt (also in `IMPLEMENTATION-PROMPT.md`):

> Implement the PHASE-PLAN(number).md following the rules of your AGENTS.md until the end and send me a message: (I finished the implementation) at the end.

## Phases

| # | Folder | Slice | Depends on (at integration) |
|---|--------|-------|-----------------------------|
| 1 | `phase-1-foundation` | go.mod, config, logger, errors, postgres+redis conn, migrations, health, server bootstrap, docker-compose, Makefile | — |
| 2 | `phase-2-auth` | user model, bcrypt, JWT, login/register, auth + role middleware | 1 |
| 3 | `phase-3-customer` | customer model/dto/repo/service/handler + filter/pagination | 1, 2 |
| 4 | `phase-4-product` | product model/dto/repo/service/handler + filter/pagination | 1, 2 |
| 5 | `phase-5-order-create` | order + item model, status rules, total calc, repo, POST handler, Redis producer (tx boundary) | 1–4 |
| 6 | `phase-6-worker-metrics` | worker goroutines, process, cancel/ship, metrics collector + endpoint | 1, 5 |
| 7 | `phase-7-docs-integration` | swagger/OpenAPI, README, end-to-end integration tests | 1–6 |
| 8 | `phase-8-live-verify` | live PG+Redis run, Makefile/compose DX, execute integration suite, fix surfaced defects | 1–7 (root repo) |
| 9 | `phase-9-prometheus-metrics` | swap in-memory metrics for Prometheus client; worker exposes own `/metrics`; compose Prometheus scrape | 1–8 (root repo) |
| 10 | `phase-10-resilience` | retry count in payload, bounded retry (≤3), `orders:dead-letter` queue; optional transactional outbox | 1–9 (root repo) |

"Standalone" means: each folder compiles and tests **alone** using its own `go.mod` and in-folder fakes/stubs (in-memory repos, fake clock, fake queue). Live PostgreSQL/Redis tests are gated behind the `integration` build tag so the default `go test ./...` runs with no external services. The "depends on" column describes only how slices merge at integration time — it is **not** visible to the implementation LLM.

## Files in each phase folder

- `AGENTS.md` — behavioral rules + operating protocol (identical across phases).
- `PHASE-PLAN<n>.md` — scope, entry/exit conditions, must-exist / must-not-exist checklists, positive + negative test lists.
- `QUALITY-GATES.md` — measurable acceptance criteria (coverage, complexity, size, dependency direction, commands).
- `GO-QUALITY-GATE.md` — Go coding-quality rules trimmed to the code this phase produces.
- `PHASE-RESULT.md` — evidence report, filled by the implementation LLM before it finishes.

## Integration & verification protocol (performed by the reviewing LLM, not the implementer)

For each phase, in order 1 → 7:

1. **Score** the delivered folder 0-100 in a short Caveman markdown note (`REVIEW.md`), using the evidence model below.
2. **Fix** gaps myself, raising quality to 100/100.
3. **Integrate** by copying the slice into the project root, reconciling it into the single root module.
4. **Test**: run `go build ./...`, `gofmt -l .`, `go vet ./...`, `go test ./...` (and `-race`, coverage) at the root. Act as CI.
5. **Correct + refactor** until green.
6. Commit the integrated, tested phase (git is initialized at root; local only until final push).
7. Move to the next phase; re-run the full root test suite (regression).

## Evidence-based scoring model

- 0-40: code exists but is not safely verified
- 41-60: basic implementation, weak tests or unclear structure
- 61-75: working implementation, meaningful tests, acceptable structure
- 76-90: strong implementation, good tests, low complexity, clean boundaries
- 91-100: production-grade, strong automated evidence, clear boundaries, strong error handling, no known quality gaps
