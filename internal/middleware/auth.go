// Package middleware provides the HTTP middleware for this phase: JWT
// authentication (auth.go) and role-based authorization (authz.go).
package middleware

import (
	"context"
	"net/http"
	"strings"

	"order-service-go/internal/auth"
	apperrors "order-service-go/internal/errors"
)

// TokenVerifier verifies a raw token and returns its claims. *auth.TokenManager
// satisfies it; the interface keeps the middleware testable with fakes.
type TokenVerifier interface {
	Verify(token string) (*auth.Claims, error)
}

type authCtxKey int

const (
	userIDKey authCtxKey = iota
	roleKey
)

// Authenticator returns middleware that requires a valid "Authorization: Bearer
// <token>" header. On success it stores the user id and role in the request
// context; otherwise it responds 401 with a generic message.
func Authenticator(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				apperrors.Write(w, apperrors.Unauthorized("Authentication required"))
				return
			}
			claims, err := verifier.Verify(token)
			if err != nil {
				apperrors.Write(w, apperrors.Unauthorized("Invalid or expired token"))
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, roleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

// UserIDFromContext returns the authenticated user id stored by Authenticator.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// ContextWithUserID stores a user id the way Authenticator does. It exists so
// handler tests can exercise routes that read the authenticated user without
// constructing a full token round-trip.
func ContextWithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// RoleFromContext returns the authenticated user's role stored by Authenticator.
func RoleFromContext(ctx context.Context) (auth.Role, bool) {
	role, ok := ctx.Value(roleKey).(auth.Role)
	return role, ok
}
