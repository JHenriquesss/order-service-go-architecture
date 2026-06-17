package product

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-service-go/internal/money"
	"order-service-go/internal/pagination"
)

// PostgresRepository is a pgx-backed ProductRepository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository builds a Postgres product repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// price is cast to text so the amount is read as a decimal string and parsed
// without ever passing through float64.
const productColumns = `id, name, sku, price::text, active, created_at, updated_at`

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	var priceText string
	if err := row.Scan(&p.ID, &p.Name, &p.SKU, &priceText, &p.Active, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	price, err := money.ParseString(priceText)
	if err != nil {
		return nil, fmt.Errorf("parse product price %q: %w", priceText, err)
	}
	p.Price = price
	return &p, nil
}

func (r *PostgresRepository) Create(ctx context.Context, p *Product) error {
	const q = `INSERT INTO products (id, name, sku, price, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, q, p.ID, p.Name, p.SKU, p.Price.Decimal(), p.Active, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, p *Product) error {
	const q = `UPDATE products SET name=$2, sku=$3, price=$4, updated_at=$5 WHERE id=$1`
	tag, err := r.pool.Exec(ctx, q, p.ID, p.Name, p.SKU, p.Price.Decimal(), p.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE products SET active=false, updated_at=now() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	p, err := scanProduct(r.pool.QueryRow(ctx, `SELECT `+productColumns+` FROM products WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepository) FindBySKU(ctx context.Context, sku string) (*Product, error) {
	p, err := scanProduct(r.pool.QueryRow(ctx, `SELECT `+productColumns+` FROM products WHERE sku=$1`, sku))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepository) FindManyByID(ctx context.Context, ids []uuid.UUID) ([]Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+productColumns+` FROM products WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Product, 0, len(ids))
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) List(ctx context.Context, filter ProductFilter) (*pagination.Page[Product], error) {
	conds, args := productConditions(filter)
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM products`+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	q := `SELECT ` + productColumns + ` FROM products` + where +
		` ORDER BY created_at, id LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Product, 0, filter.PageSize)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := 0
	if filter.PageSize > 0 {
		totalPages = (total + filter.PageSize - 1) / filter.PageSize
	}
	return &pagination.Page[Product]{
		Items:      items,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func productConditions(filter ProductFilter) ([]string, []any) {
	var conds []string
	var args []any
	if filter.Name != "" {
		args = append(args, "%"+filter.Name+"%")
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	if filter.SKU != "" {
		args = append(args, "%"+filter.SKU+"%")
		conds = append(conds, fmt.Sprintf("sku ILIKE $%d", len(args)))
	}
	if filter.Active != nil {
		args = append(args, *filter.Active)
		conds = append(conds, fmt.Sprintf("active = $%d", len(args)))
	}
	return conds, args
}
