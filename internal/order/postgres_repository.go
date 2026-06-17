package order

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-service-go/internal/money"
	"order-service-go/internal/pagination"
)

// PostgresRepository is a pgx-backed OrderRepository (architecture §18, §20).
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository builds a Postgres order repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// CreateWithItems inserts the order and its items in one transaction
// (architecture §20). The caller publishes to the queue only after this commits.
func (r *PostgresRepository) CreateWithItems(ctx context.Context, order *Order, items []OrderItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertOrder = `INSERT INTO orders (id, customer_id, status, total_amount, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if _, err := tx.Exec(ctx, insertOrder,
		order.ID, order.CustomerID, string(order.Status), order.TotalAmount.Decimal(),
		order.CreatedBy, order.CreatedAt, order.UpdatedAt,
	); err != nil {
		return err
	}

	const insertItem = `INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, total_price)
		VALUES ($1, $2, $3, $4, $5, $6)`
	for _, item := range items {
		if _, err := tx.Exec(ctx, insertItem,
			item.ID, item.OrderID, item.ProductID, item.Quantity,
			item.UnitPrice.Decimal(), item.TotalPrice.Decimal(),
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Order, []OrderItem, error) {
	const q = `SELECT id, customer_id, status, total_amount::text, created_by, created_at, updated_at, processed_at
		FROM orders WHERE id=$1`
	var o Order
	var status, totalText string
	if err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.CustomerID, &status, &totalText, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt, &o.ProcessedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrOrderNotFound
		}
		return nil, nil, err
	}
	o.Status = OrderStatus(status)
	total, err := money.ParseString(totalText)
	if err != nil {
		return nil, nil, fmt.Errorf("parse order total %q: %w", totalText, err)
	}
	o.TotalAmount = total

	items, err := r.findItems(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return &o, items, nil
}

func (r *PostgresRepository) findItems(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	const q = `SELECT id, order_id, product_id, quantity, unit_price::text, total_price::text
		FROM order_items WHERE order_id=$1 ORDER BY id`
	rows, err := r.pool.Query(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []OrderItem
	for rows.Next() {
		var it OrderItem
		var unitText, totalText string
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.Quantity, &unitText, &totalText); err != nil {
			return nil, err
		}
		if it.UnitPrice, err = money.ParseString(unitText); err != nil {
			return nil, fmt.Errorf("parse unit_price %q: %w", unitText, err)
		}
		if it.TotalPrice, err = money.ParseString(totalText); err != nil {
			return nil, fmt.Errorf("parse total_price %q: %w", totalText, err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error {
	tag, err := r.pool.Exec(ctx, `UPDATE orders SET status=$2, updated_at=now() WHERE id=$1`, id, string(status))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *PostgresRepository) MarkProcessed(ctx context.Context, id uuid.UUID, status OrderStatus, processedAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE orders SET status=$2, processed_at=$3, updated_at=$3 WHERE id=$1`,
		id, string(status), processedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, filter OrderFilter) (*pagination.Page[Order], error) {
	conds, args := orderConditions(filter)
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM orders`+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	q := `SELECT id, customer_id, status, total_amount::text, created_by, created_at, updated_at, processed_at
		FROM orders` + where +
		` ORDER BY created_at, id LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Order, 0, filter.PageSize)
	for rows.Next() {
		var o Order
		var status, totalText string
		if err := rows.Scan(&o.ID, &o.CustomerID, &status, &totalText, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt, &o.ProcessedAt); err != nil {
			return nil, err
		}
		o.Status = OrderStatus(status)
		if o.TotalAmount, err = money.ParseString(totalText); err != nil {
			return nil, fmt.Errorf("parse order total %q: %w", totalText, err)
		}
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := 0
	if filter.PageSize > 0 {
		totalPages = (total + filter.PageSize - 1) / filter.PageSize
	}
	return &pagination.Page[Order]{
		Items:      items,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func orderConditions(filter OrderFilter) ([]string, []any) {
	var conds []string
	var args []any
	if filter.CustomerID != nil {
		args = append(args, *filter.CustomerID)
		conds = append(conds, fmt.Sprintf("customer_id = $%d", len(args)))
	}
	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.StartDate != nil {
		args = append(args, *filter.StartDate)
		conds = append(conds, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.EndDate != nil {
		args = append(args, *filter.EndDate)
		conds = append(conds, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	return conds, args
}
