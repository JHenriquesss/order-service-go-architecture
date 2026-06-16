// Package auth implements authentication and role-based authorization: the user
// model, bcrypt password handling, JWT issue/verify, and the login/register
// service and handlers. It depends on a UserRepository interface so unit tests
// run with an in-memory repository and no database.
package auth

import (
	"fmt"
	"time"
)

// Role is the defined type for a user's authorization role (architecture §6).
// Only ADMIN and OPERATOR are valid; raw strings are never used as roles.
type Role string

const (
	RoleAdmin    Role = "ADMIN"
	RoleOperator Role = "OPERATOR"
)

// ParseRole validates a raw string and returns the corresponding Role. An
// unknown value is rejected rather than silently coerced.
func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleAdmin:
		return RoleAdmin, nil
	case RoleOperator:
		return RoleOperator, nil
	default:
		return "", fmt.Errorf("invalid role %q: must be ADMIN or OPERATOR", s)
	}
}

// User represents an authenticated system user (architecture §6). PasswordHash
// holds a bcrypt hash and is never serialized to clients.
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
