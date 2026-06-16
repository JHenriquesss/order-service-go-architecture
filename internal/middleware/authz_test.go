package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-service-go/internal/auth"
)

// TestAuthorizationMatrixEveryCell asserts every cell of architecture §15.
func TestAuthorizationMatrixEveryCell(t *testing.T) {
	type cell struct {
		action      Action
		admin, oper bool
	}
	cells := []cell{
		{ActionCreateUsers, true, false},
		{ActionListUsers, true, false},
		{ActionCreateCustomers, true, true},
		{ActionUpdateCustomers, true, true},
		{ActionCreateProducts, true, true},
		{ActionUpdateProducts, true, true},
		{ActionCreateOrders, true, true},
		{ActionCancelOrders, true, true},
		{ActionShipOrders, true, true},
		{ActionViewMetrics, true, false},
	}
	for _, c := range cells {
		if got := Can(auth.RoleAdmin, c.action); got != c.admin {
			t.Errorf("Can(ADMIN, %s) = %v, want %v", c.action, got, c.admin)
		}
		if got := Can(auth.RoleOperator, c.action); got != c.oper {
			t.Errorf("Can(OPERATOR, %s) = %v, want %v", c.action, got, c.oper)
		}
	}
}

func TestCanRejectsUnknownRole(t *testing.T) {
	if Can(auth.Role("GHOST"), ActionCreateCustomers) {
		t.Fatal("unknown role must not be permitted")
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestRequireActionForbidsOperatorOnAdminOnlyRoute(t *testing.T) {
	h := RequireAction(ActionListUsers)(okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req = req.WithContext(contextWithRole(req, auth.RoleOperator))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("OPERATOR on ADMIN-only route: status = %d, want 403", rec.Code)
	}
}

func TestRequireActionAllowsAdmin(t *testing.T) {
	h := RequireAction(ActionListUsers)(okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req = req.WithContext(contextWithRole(req, auth.RoleAdmin))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ADMIN on ADMIN-only route: status = %d, want 200", rec.Code)
	}
}

func TestRequireActionRejectsMissingRole(t *testing.T) {
	h := RequireAction(ActionListUsers)(okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing role: status = %d, want 401", rec.Code)
	}
}
