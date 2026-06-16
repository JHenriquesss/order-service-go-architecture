package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"order-service-go/internal/auth"
)

// contextWithRole injects an authenticated role, mirroring what Authenticator
// stores, so authz middleware can be tested in isolation.
func contextWithRole(r *http.Request, role auth.Role) context.Context {
	return context.WithValue(r.Context(), roleKey, role)
}

// fakeVerifier lets the auth-middleware tests force success or failure without
// crafting tokens for every case.
type fakeVerifier struct {
	claims *auth.Claims
	err    error
}

func (f fakeVerifier) Verify(string) (*auth.Claims, error) { return f.claims, f.err }

func TestAuthenticatorAdmitsValidTokenAndExposesContext(t *testing.T) {
	tm, _ := auth.NewTokenManager("secret", time.Hour)
	token, _, _ := tm.Issue("user-42", auth.RoleAdmin)

	var gotID string
	var gotRole auth.Role
	var gotOK bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = UserIDFromContext(r.Context())
		gotRole, gotOK = RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	Authenticator(tm)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotID != "user-42" || !gotOK || gotRole != auth.RoleAdmin {
		t.Fatalf("context: id=%q role=%q ok=%v, want user-42/ADMIN/true", gotID, gotRole, gotOK)
	}
}

func TestAuthenticatorRejectsMissingAndMalformedToken(t *testing.T) {
	verifier := fakeVerifier{err: errors.New("should not be called")}
	cases := map[string]string{
		"missing header": "",
		"no bearer":      "Token abc",
		"empty bearer":   "Bearer ",
	}
	for name, header := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		called := false
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
		Authenticator(verifier)(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized || called {
			t.Fatalf("%s: status=%d called=%v, want 401 and not called", name, rec.Code, called)
		}
	}
}

func TestAuthenticatorRejectsInvalidToken(t *testing.T) {
	verifier := fakeVerifier{err: auth.ErrInvalidToken}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer some.bad.token")
	Authenticator(verifier)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
