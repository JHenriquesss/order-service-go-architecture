package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	apperrors "order-service-go/internal/errors"

	"github.com/google/uuid"
)

// errInvalidCredentials is the single generic authentication failure. Using one
// message for unknown email, wrong password and inactive user prevents
// user-enumeration (BR-AUTH-004 covers the inactive case).
func errInvalidCredentials() error {
	return apperrors.Unauthorized("Invalid credentials")
}

// dummyHash is a precomputed bcrypt hash used to spend the same comparison cost
// when no user is found, so login latency cannot distinguish a known email from
// an unknown one (timing-based enumeration). Cost matches bcrypt.DefaultCost.
var dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// maxPasswordBytes is bcrypt's input limit; longer inputs are rejected as a
// validation error rather than surfacing as an internal hashing failure.
const maxPasswordBytes = 72

// Service holds the authentication use cases. It depends on the UserRepository
// interface and a TokenManager, keeping business rules out of the handlers.
type Service struct {
	repo   UserRepository
	tokens *TokenManager
	now    func() time.Time
}

// NewService wires the auth service.
func NewService(repo UserRepository, tokens *TokenManager) *Service {
	return &Service{repo: repo, tokens: tokens, now: time.Now}
}

// Login verifies credentials and returns an access token. It rejects unknown
// emails, wrong passwords and inactive users with one generic 401 so the client
// cannot tell them apart.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	user, err := s.repo.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Spend an equivalent bcrypt comparison so response latency does
			// not reveal whether the email exists (timing enumeration).
			ComparePassword(dummyHash, req.Password)
			return nil, errInvalidCredentials()
		}
		return nil, apperrors.Internal("could not load user", err)
	}

	if !ComparePassword(user.PasswordHash, req.Password) {
		return nil, errInvalidCredentials()
	}
	if !user.Active {
		return nil, errInvalidCredentials()
	}

	token, expiresIn, err := s.tokens.Issue(user.ID, user.Role)
	if err != nil {
		return nil, apperrors.Internal("could not issue token", err)
	}
	return &TokenResponse{AccessToken: token, TokenType: "Bearer", ExpiresIn: expiresIn}, nil
}

// Register validates the request, hashes the password and stores a new active
// user. The stored password is hashed; the plaintext is never persisted.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	if err := validateRegister(req); err != nil {
		return nil, err
	}
	role, err := ParseRole(req.Role)
	if err != nil {
		return nil, apperrors.Validation(err.Error())
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, apperrors.Internal("could not hash password", err)
	}

	now := s.now()
	user := &User{
		ID:           uuid.NewString(),
		Name:         req.Name,
		Email:        normalizeEmail(req.Email),
		PasswordHash: hash,
		Role:         role,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return nil, apperrors.Duplicate("Email already registered")
		}
		return nil, apperrors.Internal("could not create user", err)
	}

	return &UserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: string(user.Role)}, nil
}

func validateRegister(req RegisterRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return apperrors.Validation("name is required")
	}
	if strings.TrimSpace(req.Email) == "" {
		return apperrors.Validation("email is required")
	}
	if len(req.Password) < 6 {
		return apperrors.Validation("password must be at least 6 characters")
	}
	if len(req.Password) > maxPasswordBytes {
		return apperrors.Validation("password must be at most 72 bytes")
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
