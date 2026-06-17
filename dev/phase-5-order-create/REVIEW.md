# REVIEW — Phase 5 (Order Creation)

Caveman review. Reviewer = integrating LLM. Most critical phase (money + tx boundary).

## Score as delivered: 94/100

Honest, matches self-score. No code defects.

## What good

- All must-exist present, all must-not-exist absent (read + grep verified).
- Gates re-run (go1.23.10 + staticcheck): build, gofmt, vet, **staticcheck**, tests all clean. `internal/order` 87.5% pkg; service.go + status.go functions 100%.
- **`status.go`**: `CanTransition` matches §7 exactly (CREATED→PROCESSING|CANCELED, PROCESSING→PAID|FAILED, PAID→SHIPPED, else false). `ParseOrderStatus` trims+uppercases+validates. Pure, 100% covered, mutation-ready.
- **Totals**: `money.Money` int64 cents, `Add`/`Multiply` pure no-float64. `ItemTotal`/`OrderTotal` pure. §12 example (2×89.90 + 89.90 = 269.70) tested.
- **Tx boundary (§20)**: `repo.CreateWithItems` (atomic) → publish → on publish-fail: log inconsistency + return 500, order retained. Both paths tested. Publish strictly after commit.
- Unit price copied from product at creation (BR-PRD-005). Inactive customer/product → INACTIVE_*; empty items / qty<1 → VALIDATION; unknown product → NOT_FOUND. All tested.
- Ports (`CustomerLookup`/`ProductLookup`) + `OrderProducer` are interfaces with fakes. Clean boundaries.

## Gaps found (real)

- `List` does N+1 (`FindByID` per listed order to fetch items). In-memory only, acceptable this phase; flag for the SQL repo later. Not blocking.

## Env-deferred (not code defects)

- `-race`: no gcc. Shared state mutex-guarded. **Matters more at phase 6.**
- gremlins mutation: not installed; status/total are pure + fully tested.

## Integration notes (root, order-service-go + chi + real auth)

- Merge `Add`/`Multiply` into root `internal/money` (root lacked them).
- Copy `internal/order/*`; module path `order-service-order` → `order-service-go`; `Page[T]` → shared `internal/pagination`.
- Handler: replace phase `auth.IdentityFromContext` with real `middleware.UserIDFromContext` (parse uuid → createdBy); ServeMux → chi `Routes()` + `chi.URLParam`. Add `middleware.ContextWithUserID` test helper so handler_test can inject identity.
- Wire lookup adapters in main: `customer.Service.FindByID`→`order.Customer`, `product.Service.FindManyByID`→`[]order.Product`.
- **Real Redis producer**: add `internal/queue` LPUSH producer implementing `order.OrderProducer` (§9 payload to `orders:processing`); create redis client in main. Unit tests keep `order.FakeProducer`; real path exercised in phase 7 integration. (Worker consumer arrives phase 6.)
- Wire `/api/orders` behind `Authenticator` + `RequireAnyRole(ADMIN,OPERATOR)` per §15.

## Final score after fix: 100/100 (runnable gates + staticcheck + govulncheck clean). `-race` deferred (gcc).
