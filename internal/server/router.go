// Package server wires the HTTP router, mounts core middleware, and exposes
// the foundation endpoints. Business routes arrive in later phases.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"order-service-go/internal/middleware"
)

// New builds the HTTP handler: request id, logging and recovery middleware
// wrapping the mounted routes.
func New(log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logging(log))

	r.Get("/health", health)

	return r
}
