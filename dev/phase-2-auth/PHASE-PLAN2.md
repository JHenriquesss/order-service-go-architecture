# PHASE-PLAN2 — Authentication & Authorization

## Goal

JWT-based authentication and role-based authorization: user model, bcrypt password handling, JWT issue/verify, login + register endpoints, and auth + role middleware. Implemented as a standalone module with in-memory user repository so unit tests run with no database.

## Scope (build exactly this)

- `internal/auth/model.go` — `User` (architecture §6) and `Role` as a defined type with `ParseRole` (`ADMIN`, `OPERATOR`).
- `internal/auth/password.go` — bcrypt hash + compare. Never logs/returns secrets.
- `internal/auth/jwt.go` — issue + verify tokens; secret + expiry from config; validates signing method and `exp`.
- `internal/auth/repository.go` — `UserRepository` interface (`FindByEmail`, `Create`) + an in-memory implementation for tests.
- `internal/auth/service.go` — `Login`, `Register`. Login returns access token; rejects bad creds / inactive users.
- `internal/auth/dto.go` — login/register request + token response (architecture §12).
- `internal/auth/handler.go` — `POST /api/auth/login`, `POST /api/auth/register`.
- `internal/middleware/auth.go` — JWT auth middleware (puts user id + role in context).
- `internal/middleware/authz.go` — role-requirement middleware + the authorization matrix (architecture §15) as explicit, tested rules.
- Minimal in-folder scaffolding (config accessor for `JWT_SECRET`/expiry, error helpers, router wiring) so the module builds and tests alone.

## Entry condition

Foundation primitives (config, logger, errors, router) are assumed available; recreate the **minimal** versions needed here in-folder (clean-room). Do not depend on other phase folders.

## Exit condition

Login/register work over `httptest`, JWT round-trips, middleware enforces auth + role, all positive/negative tests green with no DB.

## Must-exist checklist

- [ ] `Role` defined type with validating parser; invalid role rejected.
- [ ] bcrypt hashing + compare; password/hash never logged or returned in any response.
- [ ] JWT issue + verify; rejects wrong signing method, expired token, tampered token.
- [ ] JWT secret + expiry read from config, never hardcoded.
- [ ] `UserRepository` interface + in-memory impl; service depends on the interface.
- [ ] `POST /api/auth/login` returns `{access_token, token_type, expires_in}` on valid creds.
- [ ] `POST /api/auth/register` creates a user with hashed password.
- [ ] Auth middleware rejects missing/invalid tokens with 401.
- [ ] Role middleware blocks OPERATOR from ADMIN-only routes with 403, encoding architecture §15 matrix.
- [ ] Inactive users cannot authenticate.

## Must-NOT-exist checklist

- [ ] No plaintext password storage; no password/hash in logs or responses.
- [ ] No hardcoded JWT secret.
- [ ] JWT verify must not accept `alg: none` or HMAC/RSA confusion.
- [ ] Login error must not reveal whether email vs password was wrong.
- [ ] No business rules (customer/product/order) here.
- [ ] No live DB required by default tests.
- [ ] No `TODO`/`FIXME`/debug prints.

## Positive tests

- login with valid credentials returns a token whose claims verify.
- registered user can subsequently log in.
- auth middleware admits a valid token and exposes user id + role in context.
- ADMIN passes an ADMIN-only role check.
- token expiry value matches configured `JWT_EXPIRATION_MINUTES`.

## Negative tests

- wrong password → 401, generic message.
- unknown email → 401, same generic message (no user-enumeration).
- inactive user → rejected (BR-AUTH-004).
- missing token → 401; malformed token → 401; expired token → 401; wrong-alg token → 401.
- OPERATOR on ADMIN-only route → 403.

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
