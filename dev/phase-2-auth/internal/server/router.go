// Package server wires the HTTP router for the auth phase: public auth routes
// plus a representative protected route guarded by the auth and role middleware.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"order-service-go/internal/auth"
	"order-service-go/internal/middleware"
)

// New builds the HTTP handler. Auth endpoints are public; /api/users is an
// ADMIN-only route used to demonstrate and test the auth + role middleware
// against the authorization matrix (architecture §15).
func New(handler *auth.Handler, verifier middleware.TokenVerifier) http.Handler {
	r := chi.NewRouter()

	r.Mount("/api/auth", handler.Routes())

	r.Group(func(protected chi.Router) {
		protected.Use(middleware.Authenticator(verifier))
		protected.With(middleware.RequireAction(middleware.ActionListUsers)).
			Get("/api/users", listUsers)
	})

	return r
}

// listUsers is a minimal protected endpoint; the user-management feature itself
// is out of scope for this phase.
func listUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"users":[]}`))
}
