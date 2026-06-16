package auth

import (
	"context"
	"errors"
	"sync"
)

// ErrUserNotFound is returned by FindByEmail when no user matches.
var ErrUserNotFound = errors.New("user not found")

// ErrEmailTaken is returned by Create when the email already exists.
var ErrEmailTaken = errors.New("email already registered")

// UserRepository is the persistence port for users. The service depends on this
// interface, not a concrete store, so tests use an in-memory implementation.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
}

// InMemoryUserRepository is a goroutine-safe UserRepository for unit tests and
// local runs. Emails are matched case-sensitively as provided by the service.
type InMemoryUserRepository struct {
	mu      sync.RWMutex
	byEmail map[string]*User
}

// NewInMemoryUserRepository returns an empty in-memory repository.
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{byEmail: make(map[string]*User)}
}

// FindByEmail returns the stored user or ErrUserNotFound.
func (r *InMemoryUserRepository) FindByEmail(_ context.Context, email string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	copy := *u
	return &copy, nil
}

// Create stores a new user, rejecting a duplicate email with ErrEmailTaken.
func (r *InMemoryUserRepository) Create(_ context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byEmail[user.Email]; exists {
		return ErrEmailTaken
	}
	stored := *user
	r.byEmail[user.Email] = &stored
	return nil
}
