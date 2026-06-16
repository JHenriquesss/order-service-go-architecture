package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	apperrors "order-service-go/internal/errors"
	"order-service-go/internal/logger"
)

// Recovery converts a panic in a downstream handler into a logged 500 JSON
// error response instead of crashing the server. recover is used only here, at
// the framework boundary — never as business control flow.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						logger.FieldError, fmt.Sprint(rec),
						logger.FieldRequestID, RequestIDFromContext(r.Context()),
						logger.FieldPath, r.URL.Path,
					)
					apperrors.Write(w, apperrors.Internal("Internal server error", nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
