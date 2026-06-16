# PHASE-PLAN7 — Swagger, README & End-to-End Integration Tests

## Goal

Document the API (Swagger/OpenAPI), write the README, and add real end-to-end integration tests that exercise the full workflow against live PostgreSQL + Redis. This phase validates the system as a whole; it must not change API behavior, only document and verify it.

> Note for the reviewing LLM: unlike phases 1-6, this phase's integration tests genuinely require live services and run only under the `integration` build tag. It is verified at root **after** phases 1-6 are integrated, since it needs the real assembled code. In its isolated folder it ships the test bodies, build tags, docs, and a documented run procedure; the implementer documents in PHASE-RESULT that the live run happens at integration time.

## Scope (build exactly this)

- `docs/openapi.yaml` (or swag annotations) covering auth, customer, product, order endpoints + request/response/error schemas + JWT bearer auth header (architecture §32).
- Swagger UI mounted at `GET /swagger/index.html` (wiring documented for integration).
- `README.md` following architecture §33 structure, including the Swagger URL and run instructions.
- `tests/integration/` — end-to-end test (`//go:build integration`): spin up / connect to Postgres + Redis (testcontainers or compose-backed), run migrations, then:
  login → create customer → create product → create order → worker processes it → assert order reaches PAID and metrics moved.
- Repository integration tests (Postgres) and queue integration test (Redis) under the same tag.
- `docs/architecture.md`, `docs/api.md`, `docs/business-rules.md` stubs/content as per architecture §16 tree.

## Entry condition

Phases 1-6 assumed integrated at root. In isolation, write tag-gated tests + docs against the documented public API/contracts. No dependency on other phase folders for compilation of the default build.

## Exit condition

`go build ./...` and default `go test ./...` pass (integration tests skipped without the tag). With `-tags=integration` and live services, the end-to-end test passes. Swagger renders and matches the implemented schemas. README complete.

## Must-exist checklist

- [ ] OpenAPI/Swagger covering all auth/customer/product/order endpoints + error schema + JWT header.
- [ ] Swagger UI route documented/wired.
- [ ] README with all architecture §33 sections + Swagger URL.
- [ ] End-to-end integration test covering the full happy-path workflow ending in `PAID`.
- [ ] Repository (Postgres) + queue (Redis) integration tests under `//go:build integration`.
- [ ] Documented procedure to run integration tests (services + commands).

## Must-NOT-exist checklist

- [ ] Integration tests that run by default (must be tag-gated).
- [ ] Swagger schemas that contradict actual request/response/error shapes.
- [ ] README claims not backed by the code.
- [ ] API behavior changes introduced by this phase.
- [ ] Secrets committed in docs or test fixtures.
- [ ] `TODO`/`FIXME`/debug prints.

## Positive tests

- end-to-end: full workflow login→customer→product→order→worker→PAID, asserting final persisted state + metrics.
- repository round-trip (insert + read back) against Postgres.
- queue publish + consume against Redis.
- default `go test ./...` (no tag) is green and skips integration tests.

## Negative tests

- integration test fails loudly (not silently skipped) if a required service is down when the tag is set.
- order processing failure path reaches FAILED end-to-end.
- unauthenticated end-to-end request rejected with 401.

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
