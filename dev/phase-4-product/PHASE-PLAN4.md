# PHASE-PLAN4 — Product CRUD

## Goal

Product management: create, update, deactivate, find-by-id, list with filtering + pagination, plus `FindManyByID` (used by order creation later). Standalone module; in-memory repository so unit tests need no database.

## Scope (build exactly this)

- `internal/product/model.go` — `Product` (architecture §6). Price is decimal-safe (not float64).
- `internal/product/dto.go` — create/update inputs, output, `ProductFilter`, reuse generic `Page[T]`.
- `internal/product/repository.go` — `ProductRepository` interface (`Create`, `FindByID`, `FindManyByID`, `Update`, `Deactivate`) + in-memory impl.
- `internal/product/service.go` — `ProductService` (architecture §17) with business rules.
- `internal/product/handler.go` — the 5 product endpoints (architecture §11) + filtering/pagination (§13).
- Minimal in-folder scaffolding (errors, router, auth-context accessor, decimal/money type) so the module builds and tests alone.

## Entry condition

Foundation + auth assumed; recreate **minimal** clean-room pieces needed. No dependency on other phase folders.

## Exit condition

All 5 endpoints work over `httptest`, price validation correct, filtering + pagination correct, `FindManyByID` works, positive/negative tests green with no DB.

## Must-exist checklist

- [ ] `Product` model with decimal-safe price.
- [ ] `ProductRepository` interface (incl. `FindManyByID`) + in-memory impl.
- [ ] Create: requires name (BR-PRD-001), unique sku (BR-PRD-002), price > 0 (BR-PRD-003).
- [ ] Update, Deactivate (soft).
- [ ] FindByID + FindManyByID.
- [ ] List with filters (`name`, `sku`, `active`) + pagination → `Page`.
- [ ] Endpoints protected by auth (ADMIN + OPERATOR per §15).

## Must-NOT-exist checklist

- [ ] Price stored/handled as float64.
- [ ] Physical delete of products.
- [ ] Business rules inside the handler.
- [ ] Unbounded list (page_size has a max).
- [ ] Raw SQL string concatenation of filter values.
- [ ] Live DB required by default tests.
- [ ] `TODO`/`FIXME`/debug prints.

## Positive tests

- create with valid data and price > 0 succeeds.
- update changes mutable fields.
- deactivate sets `active=false`.
- list filters by name/sku/active and paginates.
- `FindManyByID` returns the matching subset for a set of ids.

## Negative tests

- create without name → `VALIDATION_ERROR`.
- create with price ≤ 0 → `VALIDATION_ERROR`.
- create with duplicate sku → `DUPLICATE_RESOURCE`.
- find-by-id unknown → `RESOURCE_NOT_FOUND`.
- unauthenticated request → 401.

## Session log

Append concise entries to `SESSION-LOG.md` after each meaningful step.
