package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const defaultAdminEmail = "admin@example.com"

// SeedDefaultAdmin creates the default admin user (architecture §34) if it
// does not already exist. It is idempotent.
func SeedDefaultAdmin(ctx context.Context, repo *PostgresUserRepository, log *slog.Logger) error {
	if _, err := repo.FindByEmail(ctx, defaultAdminEmail); err == nil {
		return nil
	} else if !errors.Is(err, ErrUserNotFound) {
		return err
	}
	hash, err := HashPassword("123456")
	if err != nil {
		return err
	}
	now := time.Now()
	user := &User{
		ID:           uuid.NewString(),
		Name:         "Admin",
		Email:        defaultAdminEmail,
		PasswordHash: hash,
		Role:         RoleAdmin,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Create(ctx, user); err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return nil
		}
		return err
	}
	if log != nil {
		log.Info("seeded default admin user", "email", defaultAdminEmail)
	}
	return nil
}
