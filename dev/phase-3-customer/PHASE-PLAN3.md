# PHASE-PLAN3 — Customer CRUD

## Goal

Customer management: create, update, deactivate, find-by-id, and list with filtering + pagination. Standalone module; in-memory repository so unit tests need no database.

## Scope (build exactly this)

- `internal/customer/model.go` — `Customer` (architecture §6).
- `internal/customer/dto.go` — create/update inputs, output, `CustomerFilter`, generic `Page[T]`.
- `internal/customer/repository.go` — `CustomerRepository` interface (`Create`, `Update`, `Deactivate`, `FindByID`, `List`) + in-memory impl for tests.
- `internal/customer/service.go` — `CustomerService` (architecture §17) with business rules.
- `internal/customer/handler.go` — the 5 customer endpoints (architecture §11) + filtering/pagination query parsing (§13).
- Minimal in-folder scaffolding (errors, router, auth-context accessor) so the module builds and tests alone.

## Entry condition

Foundation + auth assumed; recreate the **minimal** clean-room pieces needed (error helpers, `Page[T]`, a way to read the authenticated role/user from context). No dependency on other phase folders.

## Exit condition

All 5 endpoints work over `httptest`, filtering + pagination correct, business rules enforced, positive/negative tests green with no DB.

## Must-exist checklist

- [ ] `Customer` model + DTOs + generic `Page[T]`.
- [ ] `CustomerRepository` interface + in-memory impl; service depends on interface.
- [ ] Create: requires name (BR-CUS-001), unique document (BR-CUS-002).
- [ ] Update existing customer.
- [ ] Deactivate (soft) — never physical delete (BR-CUS-004).
- [ ] FindByID returns customer or not-found.
- [ ] List with filters (`name`, `document`, `active`) + pagination (`page`, `page_size`) returning `Page`.
- [ ] Endpoints protected by auth (ADMIN + OPERATOR per §15).

## Must-NOT-exist checklist

- [ ] No physical delete of customers.
- [ ] No business rules inside the handler (handler thin).
- [ ] No unbounded list (page_size has a max).
- [ ] No raw SQL string concatenation of filter values (parameterized only).
- [ ] No live DB required by default tests.
- [ ] No `TODO`/`FIXME`/debug prints.

## Positive tests

- create with valid data succeeds and returns the customer.
- update changes mutable fields.
- deactivate sets `active=false`, record still retrievable.
- list filters by name/document/active and paginates correctly.
- find-by-id returns the right customer.

## Negative tests

- create without name → `VALIDATION_ERROR`.
- create with duplicate document → `DUPLICATE_RESOURCE`.
- find-by-id unknown → `RESOURCE_NOT_FOUND`.
- list with invalid page/page_size → bounded/validated (define behavior, test it).
- unauthenticated request → 401.

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
