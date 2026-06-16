package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "order-service-go/internal/errors"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	tm, err := NewTokenManager("test-secret", 120*time.Minute)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	return NewService(NewInMemoryUserRepository(), tm)
}

func registerUser(t *testing.T, s *Service, email, password, role string) {
	t.Helper()
	_, err := s.Register(context.Background(), RegisterRequest{
		Name: "Test User", Email: email, Password: password, Role: role,
	})
	if err != nil {
		t.Fatalf("Register(%s): %v", email, err)
	}
}

func TestRegisteredUserCanLogin(t *testing.T) {
	s := newTestService(t)
	registerUser(t, s, "admin@example.com", "123456", "ADMIN")

	resp, err := s.Login(context.Background(), LoginRequest{Email: "admin@example.com", Password: "123456"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.TokenType != "Bearer" || resp.ExpiresIn != 7200 || resp.AccessToken == "" {
		t.Fatalf("unexpected token response: %+v", resp)
	}
	if _, err := s.tokens.Verify(resp.AccessToken); err != nil {
		t.Fatalf("issued token should verify: %v", err)
	}
}

func TestRegisterHashesPasswordAndNeverReturnsIt(t *testing.T) {
	s := newTestService(t)
	resp, err := s.Register(context.Background(), RegisterRequest{
		Name: "A", Email: "a@example.com", Password: "secret123", Role: "OPERATOR",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	stored, err := s.repo.FindByEmail(context.Background(), "a@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if stored.PasswordHash == "secret123" {
		t.Fatal("password must be stored hashed, not in plaintext")
	}
	if !ComparePassword(stored.PasswordHash, "secret123") {
		t.Fatal("stored hash must verify against the password")
	}
	// UserResponse has no password field by construction; confirm role/email only.
	if resp.Email != "a@example.com" || resp.Role != "OPERATOR" {
		t.Fatalf("unexpected register response: %+v", resp)
	}
}

func TestRegisterRejectsInvalidRoleAndDuplicate(t *testing.T) {
	s := newTestService(t)
	if _, err := s.Register(context.Background(), RegisterRequest{
		Name: "A", Email: "a@example.com", Password: "secret123", Role: "WIZARD",
	}); err == nil {
		t.Fatal("expected error for invalid role")
	}
	registerUser(t, s, "dup@example.com", "123456", "ADMIN")
	_, err := s.Register(context.Background(), RegisterRequest{
		Name: "B", Email: "dup@example.com", Password: "123456", Role: "ADMIN",
	})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeDuplicate {
		t.Fatalf("expected DUPLICATE_RESOURCE, got %v", err)
	}
}

func TestLoginWrongPasswordAndUnknownEmailReturnSameGenericError(t *testing.T) {
	s := newTestService(t)
	registerUser(t, s, "user@example.com", "correctpass", "OPERATOR")

	_, errWrong := s.Login(context.Background(), LoginRequest{Email: "user@example.com", Password: "bad"})
	_, errUnknown := s.Login(context.Background(), LoginRequest{Email: "nobody@example.com", Password: "bad"})

	for _, err := range []error{errWrong, errUnknown} {
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.HTTPStatus != 401 {
			t.Fatalf("expected 401 AppError, got %v", err)
		}
	}
	if errWrong.Error() != errUnknown.Error() {
		t.Fatalf("wrong-password and unknown-email errors must be identical to prevent enumeration: %q vs %q",
			errWrong.Error(), errUnknown.Error())
	}
}

func TestRegisterValidatesRequiredFields(t *testing.T) {
	s := newTestService(t)
	cases := []RegisterRequest{
		{Name: "", Email: "a@example.com", Password: "123456", Role: "ADMIN"},
		{Name: "A", Email: "  ", Password: "123456", Role: "ADMIN"},
		{Name: "A", Email: "a@example.com", Password: "short", Role: "ADMIN"},
	}
	for i, req := range cases {
		_, err := s.Register(context.Background(), req)
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeValidation {
			t.Fatalf("case %d: expected VALIDATION_ERROR, got %v", i, err)
		}
	}
}

// errRepo forces repository failures to exercise the service's internal-error
// paths.
type errRepo struct{ findErr, createErr error }

func (r errRepo) FindByEmail(context.Context, string) (*User, error) { return nil, r.findErr }
func (r errRepo) Create(context.Context, *User) error                { return r.createErr }

func TestLoginMapsRepositoryErrorToInternal(t *testing.T) {
	tm, _ := NewTokenManager("s", time.Hour)
	s := NewService(errRepo{findErr: errors.New("db down")}, tm)
	_, err := s.Login(context.Background(), LoginRequest{Email: "x@example.com", Password: "p"})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeInternal {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestRegisterMapsRepositoryErrorToInternal(t *testing.T) {
	tm, _ := NewTokenManager("s", time.Hour)
	s := NewService(errRepo{createErr: errors.New("db down")}, tm)
	_, err := s.Register(context.Background(), RegisterRequest{
		Name: "A", Email: "a@example.com", Password: "123456", Role: "ADMIN",
	})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeInternal {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	s := newTestService(t)
	hash, _ := HashPassword("123456")
	_ = s.repo.Create(context.Background(), &User{
		ID: "u1", Name: "Inactive", Email: "inactive@example.com",
		PasswordHash: hash, Role: RoleOperator, Active: false,
	})
	_, err := s.Login(context.Background(), LoginRequest{Email: "inactive@example.com", Password: "123456"})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != 401 {
		t.Fatalf("inactive user must be rejected with 401 (BR-AUTH-004), got %v", err)
	}
}

func TestRegisterRejectsOverlongPasswordAsValidation(t *testing.T) {
	s := newTestService(t)
	longPw := make([]byte, 73)
	for i := range longPw {
		longPw[i] = 'a'
	}
	_, err := s.Register(context.Background(), RegisterRequest{
		Name: "X", Email: "x@example.com", Password: string(longPw), Role: "ADMIN",
	})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR for overlong password, got %v", err)
	}
}

func TestLoginUnknownEmailSpendsDummyCompareAndStaysGeneric(t *testing.T) {
	s := newTestService(t)
	// dummyHash must be a valid bcrypt hash so the timing-equalizing compare
	// runs without error and the path returns the generic credential error.
	if ComparePassword(dummyHash, "anything") {
		t.Fatal("dummyHash must not match an arbitrary password")
	}
	_, err := s.Login(context.Background(), LoginRequest{Email: "nobody@example.com", Password: "whatever"})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeUnauthorized {
		t.Fatalf("want generic UNAUTHORIZED, got %v", err)
	}
}
