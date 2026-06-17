// Package server wires the HTTP router, mounts core middleware, and exposes
// the service endpoints. Business routes are added per phase.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"order-service-go/internal/auth"
	"order-service-go/internal/customer"
	"order-service-go/internal/metrics"
	"order-service-go/internal/middleware"
	"order-service-go/internal/order"
	"order-service-go/internal/product"
)

// New builds the HTTP handler: request id, recovery and logging middleware
// wrap all routes. Auth endpoints are public; /api/users is an ADMIN-only route
// demonstrating the auth + role middleware against the authorization matrix
// (architecture §15).
func New(log *slog.Logger, authHandler *auth.Handler, customerHandler *customer.Handler, productHandler *product.Handler, orderHandler *order.Handler, metricsCollector *metrics.Collector, verifier middleware.TokenVerifier) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logging(log))

	r.Get("/health", health)
	// /metrics is an infrastructure endpoint (architecture §11), unauthenticated
	// like /health for scrape compatibility.
	r.Handle("/metrics", metrics.NewHandler(metricsCollector))
	r.Mount("/api/auth", authHandler.Routes())

	r.Group(func(protected chi.Router) {
		protected.Use(middleware.Authenticator(verifier))

		protected.With(middleware.RequireAction(middleware.ActionListUsers)).
			Get("/api/users", listUsers)

		// Customers + products: ADMIN or OPERATOR per architecture §15.
		protected.With(middleware.RequireAnyRole(auth.RoleAdmin, auth.RoleOperator)).
			Mount("/api/customers", customerHandler.Routes())
		protected.With(middleware.RequireAnyRole(auth.RoleAdmin, auth.RoleOperator)).
			Mount("/api/products", productHandler.Routes())
		protected.With(middleware.RequireAnyRole(auth.RoleAdmin, auth.RoleOperator)).
			Mount("/api/orders", orderHandler.Routes())
	})

	return r
}

// listUsers is a minimal protected endpoint; user management itself is out of
// scope for this phase.
func listUsers(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"users":[]}`))
}
