package customer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-service-go/internal/pagination"
)

// PostgresRepository is a pgx-backed CustomerRepository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository builds a Postgres customer repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// customerInsertColumns is the column list for INSERT.
const customerInsertColumns = `id, name, document, email, phone, active, created_at, updated_at`

// customerSelectColumns coalesces the nullable email/phone columns to empty
// strings so they scan into the model's string fields even if a row stored NULL.
const customerSelectColumns = `id, name, document, COALESCE(email, ''), COALESCE(phone, ''), active, created_at, updated_at`

func scanCustomer(row pgx.Row) (*Customer, error) {
	var c Customer
	if err := row.Scan(&c.ID, &c.Name, &c.Document, &c.Email, &c.Phone, &c.Active, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) Create(ctx context.Context, c *Customer) error {
	const q = `INSERT INTO customers (` + customerInsertColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, q, c.ID, c.Name, c.Document, c.Email, c.Phone, c.Active, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, c *Customer) error {
	const q = `UPDATE customers SET name=$2, document=$3, email=$4, phone=$5, updated_at=$6 WHERE id=$1`
	tag, err := r.pool.Exec(ctx, q, c.ID, c.Name, c.Document, c.Email, c.Phone, c.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE customers SET active=false, updated_at=now() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	c, err := scanCustomer(r.pool.QueryRow(ctx, `SELECT `+customerSelectColumns+` FROM customers WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *PostgresRepository) FindByDocument(ctx context.Context, document string) (*Customer, error) {
	c, err := scanCustomer(r.pool.QueryRow(ctx, `SELECT `+customerSelectColumns+` FROM customers WHERE document=$1`, document))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter CustomerFilter) (*pagination.Page[Customer], error) {
	conds, args := customerConditions(filter)
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM customers`+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	limit := filter.PageSize
	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, limit, offset)
	q := `SELECT ` + customerSelectColumns + ` FROM customers` + where +
		` ORDER BY created_at, id LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Customer, 0, limit)
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := 0
	if filter.PageSize > 0 {
		totalPages = (total + filter.PageSize - 1) / filter.PageSize
	}
	return &pagination.Page[Customer]{
		Items:      items,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// customerConditions builds parameterized WHERE clauses. Only placeholders are
// interpolated into SQL; user values are always bound as parameters.
func customerConditions(filter CustomerFilter) ([]string, []any) {
	var conds []string
	var args []any
	if filter.Name != "" {
		args = append(args, "%"+filter.Name+"%")
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	if filter.Document != "" {
		args = append(args, "%"+filter.Document+"%")
		conds = append(conds, fmt.Sprintf("document ILIKE $%d", len(args)))
	}
	if filter.Active != nil {
		args = append(args, *filter.Active)
		conds = append(conds, fmt.Sprintf("active = $%d", len(args)))
	}
	return conds, args
}
