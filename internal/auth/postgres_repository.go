package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserRepository is a pgx-backed UserRepository (architecture §18).
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository builds a Postgres user repository.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

// FindByEmail returns the user with the given email or ErrUserNotFound.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	const q = `SELECT id, name, email, password_hash, role, active, created_at, updated_at
		FROM users WHERE email = $1`
	var u User
	var role string
	err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &role, &u.Active, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	u.Role = Role(role)
	return &u, nil
}

// Create inserts a new user, mapping a unique-violation to ErrEmailTaken.
func (r *PostgresUserRepository) Create(ctx context.Context, user *User) error {
	const q = `INSERT INTO users (id, name, email, password_hash, role, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, q,
		user.ID, user.Name, user.Email, user.PasswordHash, string(user.Role),
		user.Active, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return err
	}
	return nil
}
