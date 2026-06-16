package middleware

import (
	"net/http"

	"order-service-go/internal/auth"
	apperrors "order-service-go/internal/errors"
)

// Action identifies an authorization-controlled operation from the
// authorization matrix (architecture §15).
type Action string

const (
	ActionCreateUsers     Action = "create_users"
	ActionListUsers       Action = "list_users"
	ActionCreateCustomers Action = "create_customers"
	ActionUpdateCustomers Action = "update_customers"
	ActionCreateProducts  Action = "create_products"
	ActionUpdateProducts  Action = "update_products"
	ActionCreateOrders    Action = "create_orders"
	ActionCancelOrders    Action = "cancel_orders"
	ActionShipOrders      Action = "ship_orders"
	ActionViewMetrics     Action = "view_metrics"
)

// matrix encodes architecture §15 exactly: which roles may perform each action.
// A missing (action, role) entry means "denied".
var matrix = map[Action]map[auth.Role]bool{
	ActionCreateUsers:     {auth.RoleAdmin: true, auth.RoleOperator: false},
	ActionListUsers:       {auth.RoleAdmin: true, auth.RoleOperator: false},
	ActionCreateCustomers: {auth.RoleAdmin: true, auth.RoleOperator: true},
	ActionUpdateCustomers: {auth.RoleAdmin: true, auth.RoleOperator: true},
	ActionCreateProducts:  {auth.RoleAdmin: true, auth.RoleOperator: true},
	ActionUpdateProducts:  {auth.RoleAdmin: true, auth.RoleOperator: true},
	ActionCreateOrders:    {auth.RoleAdmin: true, auth.RoleOperator: true},
	ActionCancelOrders:    {auth.RoleAdmin: true, auth.RoleOperator: true},
	ActionShipOrders:      {auth.RoleAdmin: true, auth.RoleOperator: true},
	ActionViewMetrics:     {auth.RoleAdmin: true, auth.RoleOperator: false},
}

// Can reports whether role is permitted to perform action under the matrix.
func Can(role auth.Role, action Action) bool {
	return matrix[action][role]
}

// RequireAnyRole returns middleware that allows the request only if the
// authenticated role (set by Authenticator) is one of roles. It responds 401 if
// no role is present and 403 if the role is not permitted. Use it to guard a
// group of routes that all share the same role set (e.g. ADMIN or OPERATOR).
func RequireAnyRole(roles ...auth.Role) func(http.Handler) http.Handler {
	allowed := make(map[auth.Role]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok {
				apperrors.Write(w, apperrors.Unauthorized("Authentication required"))
				return
			}
			if !allowed[role] {
				apperrors.Write(w, apperrors.Forbidden("You do not have permission to perform this action"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAction returns middleware that allows the request only if the
// authenticated role (set by Authenticator) may perform action. It responds 401
// if no role is present and 403 if the role is not permitted.
func RequireAction(action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok {
				apperrors.Write(w, apperrors.Unauthorized("Authentication required"))
				return
			}
			if !Can(role, action) {
				apperrors.Write(w, apperrors.Forbidden("You do not have permission to perform this action"))
				return
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}
