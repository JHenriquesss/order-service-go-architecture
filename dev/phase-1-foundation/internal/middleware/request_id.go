// Package middleware provides the foundational HTTP middleware: request id
// propagation, structured request logging, and panic recovery.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// HeaderRequestID is the HTTP header carrying the request correlation id.
const HeaderRequestID = "X-Request-Id"

type ctxKey int

const requestIDKey ctxKey = iota

// RequestID ensures every request has a correlation id: it reuses an inbound
// X-Request-Id or generates one, stores it in the context, and echoes it back.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(HeaderRequestID, id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the correlation id stored by RequestID, or "".
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
