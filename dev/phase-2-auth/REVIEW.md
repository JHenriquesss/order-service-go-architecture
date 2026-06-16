# REVIEW — Phase 2 (Auth)

Caveman review. Reviewer = integrating LLM.

## Score as delivered: 92/100

Honest, matches self-score.

## What good

- All must-exist present, all must-not-exist absent (grep + read verified).
- Gates re-run by me (go1.23.10 + staticcheck installed): build ok, gofmt clean, vet clean, **staticcheck clean**, `go test ./...` all pass. Coverage: middleware 100%, auth jwt/password 100%, service Login 92.3%/Register 93.3%. Meets ≥90% gate on jwt/password/service.
- Hand-rolled HS256 JWT (no lib) is **correct + safe**: fixed alg, `header.Alg != HS256` rejected before verify, signature checked with constant-time `hmac.Equal` before payload trusted, exp + role-claim validated. alg:none / alg-confusion / tamper / expired / malformed all tested.
- bcrypt hash + `CompareHashAndPassword` (constant-time). Hash never serialized (`UserResponse` has no hash field), never logged.
- §15 authorization matrix encoded as explicit map + every cell tested. RequireAction 401-no-role vs 403-denied correct.
- Single generic credential error (unknown email = wrong password = inactive). No enumeration via message.
- Boundaries clean: handler decode→service→repo interface; in-mem repo RWMutex-guarded, copies in/out.

## Gaps found (real)

1. `service.Login`: unknown email returns before `ComparePassword` → **timing side-channel**. Known email pays bcrypt cost, unknown doesn't → enumeration by latency despite identical message/status.
2. `Register`: password >72 bytes makes bcrypt error → `Internal`/500 on user-controlled input. Should be a 400 validation.

## Env-deferred (not code defects)

- `-race`: no cgo/gcc in box. Shared state = RWMutex-guarded in-mem repo + request-context values; low risk.
- gosec/golangci-lint not installed. staticcheck (clean) + govulncheck cover most.

## Fixes I applied → 100/100

- Login: dummy constant-cost bcrypt compare on the user-not-found path to equalize timing.
- Register: reject password >72 bytes (bcrypt limit) as `VALIDATION_ERROR` before hashing.
- Re-ran gates green after fix.

## Integration notes (root)

- Copy `internal/auth/*` + `internal/middleware/{auth,authz}.go` into root.
- Add `Unauthorized()` + `Forbidden()` constructors to root `internal/errors` (codes already exist there).
- Merge root `server/router.go`: keep foundation mw (RequestID/Recovery/Logging) + `/health`, add `/api/auth` + protected group. Update `New` signature.
- Update `cmd/api/main.go`: wire config→TokenManager→repo→service→handler→server.New.
- Root config already has `JWTSecret`/`JWTExpirationMinutes`; keep root config (phase-2 config is a minimal stand-in — discard).
- `go get golang.org/x/crypto`, `go mod tidy`.

## Final score after fix: 100/100 (runnable gates + staticcheck clean). `-race` deferred (gcc).
