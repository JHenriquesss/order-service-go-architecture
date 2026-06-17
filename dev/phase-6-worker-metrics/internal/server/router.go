// Package server wires the order endpoints behind authentication and
// role-authorization middleware, plus the metrics endpoint.
package server

import (
	"net/http"

	"order-service-worker/internal/auth"
	"order-service-worker/internal/metrics"
	"order-service-worker/internal/order"
)

// New builds the HTTP handler for the order API (protected) and public metrics.
func New(verifier auth.Verifier, orderHandler *order.Handler, metricsHandler *metrics.Handler) http.Handler {
	return NewWithPublicMetrics(verifier, orderHandler, metricsHandler)
}

// NewWithPublicMetrics exposes /metrics without authentication.
func NewWithPublicMetrics(verifier auth.Verifier, orderHandler *order.Handler, metricsHandler *metrics.Handler) http.Handler {
	mux := http.NewServeMux()
	metricsHandler.Register(mux)

	orderMux := http.NewServeMux()
	orderHandler.Register(orderMux)
	protectedOrders := auth.Authenticator(verifier)(
		auth.RequireRoles(auth.RoleAdmin, auth.RoleOperator)(orderMux),
	)
	mux.Handle("/api/", protectedOrders)

	return mux
}
