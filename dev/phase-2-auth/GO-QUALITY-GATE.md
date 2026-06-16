# GO-QUALITY-GATE.md (distilled, scoped to this phase)

Engineering control document, not a style preference. Code is **not** done because files exist. It is done when it builds, is formatted, passes vet, has meaningful tests, preserves boundaries, models errors explicitly, and has evidence in `PHASE-RESULT.md`. Full reference: `../../GO-CODE-QUALITY-GATE.md` (not in this folder — these are the rules that matter for this slice).

## Mandatory commands (run; record results in PHASE-RESULT.md)

- `go version` and `go env GOVERSION GOOS GOARCH CGO_ENABLED`
- `test -z "$(gofmt -l .)"` — no unformatted files
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `go test -covermode=atomic -coverprofile=coverage.out ./...` then `go tool cover -func=coverage.out`

A command not run is not evidence. A command that failed and was ignored is negative evidence. If a command can't run, document the concrete blocker (never "tool unavailable").

## Toolchain & modules

- Stable Go, explicit `go` directive in `go.mod`. Document the version used.
- `go.mod` + `go.sum` committed and tidy (`go mod tidy` leaves no diff).
- No `replace` to local paths. No dependency added for a trivial helper.
- Build reproducible from a clean checkout with the documented commands.

## Formatting & naming

- `gofmt` clean, mandatory. `goimports` grouping if used.
- Package names short, lowercase, meaningful. No `utils`/`common`/`helpers`/`misc` dumping grounds. No stutter (`order.Order`, not `order.OrderOrder`).
- Initialisms consistent: `ID`, `URL`, `HTTP`, `JSON`, `SQL`, `UUID`.
- Names reveal intent; tests named as behavior (`TestXValidatesY`).

## Errors

- Return errors for fallible ops; `panic` only for programmer bugs / impossible invariants.
- Wrap with `fmt.Errorf("context: %w", err)` when callers need `errors.Is`/`errors.As`.
- No `errors.New("failed")`-style vagueness. No string-matching on errors in logic.
- No `log.Fatal`/`os.Exit` outside `main`/command composition.
- Don't return `nil, nil` where absence has meaning. Test failure paths, not only the happy path.
- One consistent API error format (see architecture §24): `{ "error": { "code", "message" } }`.

## Panic / leftovers

- No `panic("TODO")`, `panic("not implemented")`, `TODO`, `FIXME`, `fmt.Println` debug, leftover prints.
- `recover` only at process/framework boundaries, never as business control flow.

## Architectural boundaries (architecture.md wins)

- Handler → Service → Repository → DB. Dependency direction inward.
- Business rules live in service/domain, **never** in handlers, SQL, or DTO mappers.
- Handlers stay thin: parse, validate shape, call service, map result, respond.
- Repositories focus on persistence; map rows to domain types explicitly.
- Use interfaces (ports) for repository / clock / queue so unit tests use fakes.

## Types

- Defined types for domain identifiers/states (e.g. `OrderStatus string` with a validating parser), not raw strings/maps.
- Enforce invariants at construction; keep fields unexported when they protect invariants.
- No `map[string]any` / `any` as a domain model.

## Testing

- TDD: red → green → refactor. Tests encode the phase goal and exercise real behavior.
- Table tests where natural; readable after `gofmt`.
- Unit tests run with **no external services** (use in-memory/fake adapters).
- Live PostgreSQL/Redis tests gated behind `//go:build integration`.
- Test nil/zero/empty/invalid inputs where the contract distinguishes them.

## Phase-specific emphasis: Auth

- Passwords: bcrypt only; never log or return password/hash. Constant-time compare via bcrypt API.
- JWT secret comes from config, never hardcoded. Validate `exp`, signing method (reject `none`/alg confusion).
- Login must not reveal whether email or password was wrong (generic "invalid credentials").
- Inactive users cannot authenticate (BR-AUTH-004).
- `role` is a defined type with a validating parser (`ADMIN`, `OPERATOR`), not a raw string.
- Middleware: auth + role checks are middleware, but the authorization *matrix* decision is explicit and tested.
