package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestManager(t *testing.T) *TokenManager {
	t.Helper()
	m, err := NewTokenManager("test-secret", 120*time.Minute)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	return m
}

func TestNewTokenManagerRejectsBadInput(t *testing.T) {
	if _, err := NewTokenManager("", time.Minute); err == nil {
		t.Fatal("empty secret should be rejected")
	}
	if _, err := NewTokenManager("s", 0); err == nil {
		t.Fatal("non-positive expiry should be rejected")
	}
}

func TestIssueAndVerifyRoundTrip(t *testing.T) {
	m := newTestManager(t)
	token, expiresIn, err := m.Issue("user-1", RoleAdmin)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if expiresIn != 7200 {
		t.Fatalf("expiresIn = %d, want 7200 (120 min)", expiresIn)
	}
	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != "user-1" || claims.Role != RoleAdmin {
		t.Fatalf("claims = %+v, want user-1/ADMIN", claims)
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	m := newTestManager(t)
	m.now = func() time.Time { return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) }
	token, _, _ := m.Issue("user-1", RoleOperator)
	m.now = func() time.Time { return time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC) }
	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	m := newTestManager(t)
	token, _, _ := m.Issue("user-1", RoleAdmin)
	parts := strings.Split(token, ".")
	// Re-encode a payload claiming a different user; signature no longer matches.
	parts[1] = encodeSegment(jwtPayload{Sub: "attacker", Role: string(RoleAdmin), Exp: 9999999999, Iat: 1})
	tampered := strings.Join(parts, ".")
	if _, err := m.Verify(tampered); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for tampered token, got %v", err)
	}
}

func TestVerifyRejectsAlgNone(t *testing.T) {
	m := newTestManager(t)
	header := encodeSegment(jwtHeader{Alg: "none", Typ: "JWT"})
	payload := encodeSegment(jwtPayload{Sub: "user-1", Role: string(RoleAdmin), Exp: 9999999999, Iat: 1})
	token := header + "." + payload + "." // empty signature, as alg:none attacks use
	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for alg:none, got %v", err)
	}
}

func TestVerifyRejectsAlgConfusion(t *testing.T) {
	m := newTestManager(t)
	// A token validly HMAC-signed but advertising a different algorithm must be
	// rejected on the algorithm check, not silently trusted.
	payload := encodeSegment(jwtPayload{Sub: "user-1", Role: string(RoleAdmin), Exp: 9999999999, Iat: 1})
	header := encodeSegment(jwtHeader{Alg: "RS256", Typ: "JWT"})
	sig := base64.RawURLEncoding.EncodeToString(m.sign(header + "." + payload))
	token := header + "." + payload + "." + sig
	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for alg confusion, got %v", err)
	}
}

func TestVerifyRejectsBadSignatureEncoding(t *testing.T) {
	m := newTestManager(t)
	header := encodeSegment(jwtHeader{Alg: algHS256, Typ: "JWT"})
	payload := encodeSegment(jwtPayload{Sub: "u", Role: string(RoleAdmin), Exp: 9999999999, Iat: 1})
	token := header + "." + payload + ".!!!notbase64"
	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for bad signature encoding, got %v", err)
	}
}

func TestVerifyRejectsBadPayloadEncoding(t *testing.T) {
	m := newTestManager(t)
	header := encodeSegment(jwtHeader{Alg: algHS256, Typ: "JWT"})
	// "!!" is valid token shape (3 segments) but not valid base64 payload.
	sig := base64.RawURLEncoding.EncodeToString(m.sign(header + ".!!"))
	if _, err := m.Verify(header + ".!!." + sig); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for bad payload encoding, got %v", err)
	}
}

func TestVerifyRejectsInvalidRoleClaim(t *testing.T) {
	m := newTestManager(t)
	header := encodeSegment(jwtHeader{Alg: algHS256, Typ: "JWT"})
	payload := encodeSegment(jwtPayload{Sub: "u", Role: "WIZARD", Exp: 9999999999, Iat: 1})
	sig := base64.RawURLEncoding.EncodeToString(m.sign(header + "." + payload))
	if _, err := m.Verify(header + "." + payload + "." + sig); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for invalid role claim, got %v", err)
	}
}

func TestVerifyRejectsMalformedToken(t *testing.T) {
	m := newTestManager(t)
	for _, tok := range []string{"", "abc", "a.b", "a.b.c.d", "!!!.@@@.###"} {
		if _, err := m.Verify(tok); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken for %q, got %v", tok, err)
		}
	}
}
