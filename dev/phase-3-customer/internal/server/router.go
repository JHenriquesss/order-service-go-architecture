// Package server wires the customer endpoints behind the authentication and
// role-authorization middleware. It is the minimal router scaffold this phase
// needs to exercise the endpoints over httptest.
package server

import (
	"net/http"

	"order-service-customer/internal/auth"
	"order-service-customer/internal/customer"
)

// New builds the HTTP handler for the customer API. Every customer endpoint is
// protected: a valid bearer token is required (Authenticator) and the role must
// be ADMIN or OPERATOR per the authorization matrix (architecture §15).
func New(verifier auth.Verifier, handler *customer.Handler) http.Handler {
	mux := http.NewServeMux()
	handler.Register(mux)

	protected := auth.Authenticator(verifier)(
		auth.RequireRoles(auth.RoleAdmin, auth.RoleOperator)(mux),
	)
	return protected
}
