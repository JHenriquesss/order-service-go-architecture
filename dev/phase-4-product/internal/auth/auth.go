// Package auth is the minimal clean-room authentication/authorization scaffold
// this phase needs: a Role type, the authenticated identity stored in the
// request context, and middleware that enforces a valid bearer token and the
// role authorization matrix (architecture §15). It depends on a Verifier
// interface so tests inject a fake and run without real JWT infrastructure.
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	apperrors "order-service-product/internal/errors"
)

// Role is the defined type for a user's authorization role (architecture §6).
// Only ADMIN and OPERATOR are valid; raw strings are never used as roles.
type Role string

const (
	RoleAdmin    Role = "ADMIN"
	RoleOperator Role = "OPERATOR"
)

// Identity is the authenticated principal extracted from a token.
type Identity struct {
	UserID uuid.UUID
	Role   Role
}

// Verifier verifies a raw bearer token and returns its identity. A real
// implementation would parse a JWT; tests inject a fake. The seam keeps this
// phase free of any concrete token dependency.
type Verifier interface {
	Verify(token string) (Identity, error)
}

type ctxKey int

const identityKey ctxKey = iota

// Authenticator returns middleware that requires a valid
// "Authorization: Bearer <token>" header. On success it stores the Identity in
// the request context; otherwise it responds 401 with a generic message.
func Authenticator(verifier Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				apperrors.Write(w, apperrors.Unauthorized("Authentication required"))
				return
			}
			identity, err := verifier.Verify(token)
			if err != nil {
				apperrors.Write(w, apperrors.Unauthorized("Invalid or expired token"))
				return
			}
			ctx := context.WithValue(r.Context(), identityKey, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles returns middleware that allows the request only if the
// authenticated role (set by Authenticator) is one of roles. It responds 401 if
// no identity is present and 403 if the role is not permitted.
func RequireRoles(roles ...Role) func(http.Handler) http.Handler {
	allowed := make(map[Role]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				apperrors.Write(w, apperrors.Unauthorized("Authentication required"))
				return
			}
			if !allowed[identity.Role] {
				apperrors.Write(w, apperrors.Forbidden("You do not have permission to perform this action"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IdentityFromContext returns the authenticated identity stored by Authenticator.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	return identity, ok
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
