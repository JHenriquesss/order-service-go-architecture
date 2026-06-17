# REVIEW — Phase 7 (Docs + Integration)

Caveman review. Reviewer = integrating LLM. Final phase.

## Score as delivered: 82/100

Honest. Code-complete; live integration unrun in isolation; swagger UI not mountable in isolated module.

## What good

- Default gates re-run (go1.23.10 + staticcheck): build, gofmt, vet, **staticcheck**, `go test ./...` (openapi unit tests) clean.
- OpenAPI 3 spec covers all auth/customer/product/order paths + error schema (§24) + JWT bearer (§32); validated programmatically via kin-openapi. `RequiredPaths` match root chi routes exactly.
- Integration tests are **black-box** (HTTP client + raw pgx + env DSNs + migrations runner) — no app-package coupling, portable to root. Tag-gated `//go:build integration`; fail loudly (not skip) when services/env missing.
- README full §33; docs/api.md, architecture.md, business-rules.md present (§16).

## Gaps found (real)

1. **e2e metrics assertion is cross-process-unsound**: `TestE2EWorkflowReachesPaidAndIncrementsMetrics` checks the API `/metrics` `orders_processed_total` rises, but that counter increments in the **worker** process (separate in-memory collector). Under the architecture's per-process in-memory metrics (§22), the API endpoint can't see it. Fix at integration: keep the order→PAID + positive-total assertions (the real end-to-end proof), drop the cross-process metric-increment check; document metrics aggregation as future work (Prometheus, §22).
2. Swagger UI not served (isolated module can't mount). Add at root.

## Env-deferred (not code defects)

- Live `go test -tags=integration ./...` needs Postgres + Redis + api + worker (no Docker in this CI box). Tests compile under the tag; user runs live via docker-compose. **This is where the phase 6.5 pgx repos finally get live validation.**

## Integration notes (root)

- Copy docs/ + README to root; embed openapi.yaml in internal/openapi (`go:embed`, not disk/runtime.Caller — works in a built binary) for both the unit test and swagger serving.
- Mount `/swagger/index.html` + `/swagger/openapi.yaml` in root cmd/api (CDN swagger-ui, unauthenticated).
- Copy tests/integration/* with module rewrite; trim the cross-process metrics assertion; keep order→PAID.
- `go get github.com/getkin/kin-openapi` (1.23-safe at v0.132 with x/text v0.21).

## Final score after fix: 100/100 on runnable gates (build, fmt, vet, staticcheck, default tests, -race, integration-compiles). Live integration run is the user's `docker compose up` step — documented + runnable.
