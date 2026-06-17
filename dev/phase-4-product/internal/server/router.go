// Package server wires the product endpoints behind the authentication and
// role-authorization middleware. It is the minimal router scaffold this phase
// needs to exercise the endpoints over httptest.
package server

import (
	"net/http"

	"order-service-product/internal/auth"
	"order-service-product/internal/product"
)

// New builds the HTTP handler for the product API. Every product endpoint is
// protected: a valid bearer token is required (Authenticator) and the role must
// be ADMIN or OPERATOR per the authorization matrix (architecture §15).
func New(verifier auth.Verifier, handler *product.Handler) http.Handler {
	mux := http.NewServeMux()
	handler.Register(mux)

	protected := auth.Authenticator(verifier)(
		auth.RequireRoles(auth.RoleAdmin, auth.RoleOperator)(mux),
	)
	return protected
}
