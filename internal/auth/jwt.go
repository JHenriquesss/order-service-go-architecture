package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TokenManager issues and verifies HS256 JWTs. It accepts exactly one signing
// algorithm (HMAC-SHA256), which by construction rejects "alg: none" and any
// HMAC/RSA confusion: a token signed with another algorithm cannot produce a
// matching HMAC under our secret.
type TokenManager struct {
	secret []byte
	expiry time.Duration
	now    func() time.Time
}

// Claims are the verified contents of a token relevant to authorization.
type Claims struct {
	UserID    string
	Role      Role
	ExpiresAt time.Time
}

// jwtHeader is the fixed JWT header for the tokens we issue.
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// jwtPayload is the on-the-wire claim set. exp/iat are seconds since the epoch.
type jwtPayload struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
	Iat  int64  `json:"iat"`
}

const algHS256 = "HS256"

// ErrInvalidToken is returned for any malformed, tampered, wrong-algorithm or
// expired token. The cause is wrapped for diagnostics but callers map it to a
// generic 401 so no detail leaks to clients.
var ErrInvalidToken = errors.New("invalid token")

// NewTokenManager builds a TokenManager from a non-empty secret and a positive
// expiry. An empty secret is a programmer error in wiring and is rejected.
func NewTokenManager(secret string, expiry time.Duration) (*TokenManager, error) {
	if secret == "" {
		return nil, errors.New("jwt: secret must not be empty")
	}
	if expiry <= 0 {
		return nil, errors.New("jwt: expiry must be positive")
	}
	return &TokenManager{secret: []byte(secret), expiry: expiry, now: time.Now}, nil
}

// Issue signs a token for the user and role and returns it with the expiry in
// seconds (matching the configured JWT_EXPIRATION_MINUTES).
func (m *TokenManager) Issue(userID string, role Role) (token string, expiresInSeconds int, err error) {
	now := m.now()
	header := encodeSegment(jwtHeader{Alg: algHS256, Typ: "JWT"})
	payload := encodeSegment(jwtPayload{
		Sub:  userID,
		Role: string(role),
		Exp:  now.Add(m.expiry).Unix(),
		Iat:  now.Unix(),
	})
	signing := header + "." + payload
	sig := base64.RawURLEncoding.EncodeToString(m.sign(signing))
	return signing + "." + sig, int(m.expiry.Seconds()), nil
}

// Verify validates the signing algorithm, signature and expiry, returning the
// claims only when all checks pass.
func (m *TokenManager) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected 3 segments", ErrInvalidToken)
	}

	var header jwtHeader
	if err := decodeSegment(parts[0], &header); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrInvalidToken, err)
	}
	if header.Alg != algHS256 {
		return nil, fmt.Errorf("%w: unexpected alg %q", ErrInvalidToken, header.Alg)
	}

	expected := m.sign(parts[0] + "." + parts[1])
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: signature encoding", ErrInvalidToken)
	}
	if !hmac.Equal(expected, actual) {
		return nil, fmt.Errorf("%w: signature mismatch", ErrInvalidToken)
	}

	var payload jwtPayload
	if err := decodeSegment(parts[1], &payload); err != nil {
		return nil, fmt.Errorf("%w: payload: %v", ErrInvalidToken, err)
	}
	if !m.now().Before(time.Unix(payload.Exp, 0)) {
		return nil, fmt.Errorf("%w: expired", ErrInvalidToken)
	}
	role, err := ParseRole(payload.Role)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	return &Claims{UserID: payload.Sub, Role: role, ExpiresAt: time.Unix(payload.Exp, 0)}, nil
}

func (m *TokenManager) sign(signing string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(signing))
	return mac.Sum(nil)
}

func encodeSegment(v any) string {
	b, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeSegment(segment string, v any) error {
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}
