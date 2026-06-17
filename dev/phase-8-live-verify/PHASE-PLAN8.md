# PHASE-PLAN8 — Live Verification & Developer Experience

## Goal

Prove the assembled system runs end-to-end against **live PostgreSQL + Redis**, and make that one command. This phase is the first real execution of the `//go:build integration` suite (which validates the pgx repositories and the api→Redis→worker→PAID flow). Any defect the live run surfaces — especially pgx SQL — is fixed here.

> Unlike phases 1-7, this phase operates on the **already-integrated root repository** (module `order-service-go`), not a clean-room slice. It needs Docker (Postgres + Redis) available.

## Scope (build/verify exactly this)

- `Makefile` targets (extend architecture §29): `migrate-up`, `migrate-down`, `seed`, `run-api`, `run-worker`, `compose-up`, `compose-down`, and `test-integration` (`go test -tags=integration ./...` with the documented env vars).
- `docker-compose.yml`: add **healthchecks** — postgres `pg_isready`, redis `redis-cli ping`; make `api`/`worker` `depends_on` with `condition: service_healthy`.
- Migration application: a documented, repeatable way to apply `migrations/` to the running database before api/worker serve (golang-migrate target and/or a one-shot compose step). The integration suite already self-migrates; the app path must also be covered.
- Run the integration suite live and make it green: repo round-trips, redis publish/consume, e2e happy path → PAID, failure path → FAILED, auth 401, services-down.
- Fix any defect the live run reveals (pgx SQL, wiring, env). Keep fixes surgical; do not change business behavior covered by unit tests.
- README "How to Run" + "Tests" sections updated with the exact live procedure.

## Entry condition

Root repo with phases 1-7 + Postgres repos integrated; default `go test ./...` green; Docker available.

## Exit condition

`make test-integration` (or the documented command) passes against live Postgres + Redis with api + worker running: e2e reaches PAID, failure reaches FAILED, round-trips pass. Default `go test ./...` still green. One documented command brings the stack up.

## Must-exist checklist

- [ ] Makefile targets: `migrate-up`, `migrate-down`, `seed`, `compose-up`, `compose-down`, `test-integration`.
- [ ] docker-compose healthchecks for postgres + redis; api/worker wait for healthy.
- [ ] Documented migration-apply step that works against the compose database.
- [ ] `go test -tags=integration ./...` green against live services (evidence captured in PHASE-RESULT).
- [ ] e2e: login → customer → product → order → worker → **PAID** asserted on live data.
- [ ] Failure path → **FAILED** asserted live.
- [ ] Default `go test ./...` (no tag) still green.
- [ ] README run + test procedure accurate.

## Must-NOT-exist checklist

- [ ] No integration test that skips when the tag is set and services are down (must fail loudly).
- [ ] No business-logic change that alters phase-1..7 unit-test behavior.
- [ ] No secrets committed (compose dev creds only).
- [ ] No reliance on a manually mutated DB (the procedure must be reproducible from clean).
- [ ] No `TODO`/`FIXME`/debug prints added.

## Positive verification

- Stack comes up from clean with one documented command; api `/health` 200; `/swagger/index.html` renders.
- Full `-tags=integration` suite passes; e2e order reaches PAID; metrics endpoint serves.
- Re-running the suite is idempotent (unique test data per run).

## Negative verification

- With Postgres or Redis down, integration tests fail loudly (not skip).
- Bad `DATABASE_URL` fails fast with a clear error.

## Known-gap notes (carry forward, not fixed here)

- Cross-process metric aggregation (worker `processed`/`failed`) is **Phase 9** (Prometheus). The e2e asserts `orders_created_total` (API-side), not processed.
- Stdlib govulncheck advisories need a Go toolchain patch bump (separate ops task).

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
