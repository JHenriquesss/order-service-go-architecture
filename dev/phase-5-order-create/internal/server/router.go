// Package server wires the order endpoints behind authentication and
// role-authorization middleware.
package server

import (
	"net/http"

	"order-service-order/internal/auth"
	"order-service-order/internal/order"
)

// New builds the HTTP handler for the order API.
func New(verifier auth.Verifier, handler *order.Handler) http.Handler {
	mux := http.NewServeMux()
	handler.Register(mux)

	protected := auth.Authenticator(verifier)(
		auth.RequireRoles(auth.RoleAdmin, auth.RoleOperator)(mux),
	)
	return protected
}
