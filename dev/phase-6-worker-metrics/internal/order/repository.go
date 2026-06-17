package order

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrOrderNotFound is the domain not-found signal for orders.
var ErrOrderNotFound = errors.New("order not found")

// OrderRepository is the persistence port (architecture §18).
type OrderRepository interface {
	CreateWithItems(ctx context.Context, order *Order, items []OrderItem) error
	FindByID(ctx context.Context, id uuid.UUID) (*Order, []OrderItem, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
	MarkProcessed(ctx context.Context, id uuid.UUID, status OrderStatus, processedAt time.Time) error
	List(ctx context.Context, filter OrderFilter) (*Page[Order], error)
}

// InMemoryRepository is a concurrency-safe in-memory OrderRepository for tests.
type InMemoryRepository struct {
	mu     sync.RWMutex
	orders map[uuid.UUID]Order
	items  map[uuid.UUID][]OrderItem

	CommitCount int
	OnCreate    func(orderID uuid.UUID)
}

// NewInMemoryRepository builds an empty in-memory repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		orders: make(map[uuid.UUID]Order),
		items:  make(map[uuid.UUID][]OrderItem),
	}
}

func (r *InMemoryRepository) CreateWithItems(_ context.Context, order *Order, items []OrderItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if order == nil {
		return errors.New("order is nil")
	}
	if len(items) == 0 {
		return errors.New("order must have items")
	}
	for _, item := range items {
		if item.OrderID != order.ID {
			return errors.New("item order_id mismatch")
		}
	}

	r.orders[order.ID] = *order
	stored := make([]OrderItem, len(items))
	copy(stored, items)
	r.items[order.ID] = stored
	r.CommitCount++

	if r.OnCreate != nil {
		r.OnCreate(order.ID)
	}
	return nil
}

func (r *InMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*Order, []OrderItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return nil, nil, ErrOrderNotFound
	}
	items := r.items[id]
	copied := make([]OrderItem, len(items))
	copy(copied, items)
	return &order, copied, nil
}

func (r *InMemoryRepository) UpdateStatus(_ context.Context, id uuid.UUID, status OrderStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	order, ok := r.orders[id]
	if !ok {
		return ErrOrderNotFound
	}
	order.Status = status
	order.UpdatedAt = time.Now()
	r.orders[id] = order
	return nil
}

func (r *InMemoryRepository) MarkProcessed(_ context.Context, id uuid.UUID, status OrderStatus, processedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	order, ok := r.orders[id]
	if !ok {
		return ErrOrderNotFound
	}
	order.Status = status
	order.ProcessedAt = &processedAt
	order.UpdatedAt = processedAt
	r.orders[id] = order
	return nil
}

func (r *InMemoryRepository) List(_ context.Context, filter OrderFilter) (*Page[Order], error) {
	r.mu.RLock()
	matched := make([]Order, 0, len(r.orders))
	for _, o := range r.orders {
		if matchesOrder(o, filter) {
			matched = append(matched, o)
		}
	}
	r.mu.RUnlock()

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].CreatedAt.Equal(matched[j].CreatedAt) {
			return matched[i].ID.String() < matched[j].ID.String()
		}
		return matched[i].CreatedAt.Before(matched[j].CreatedAt)
	})

	total := len(matched)
	start := (filter.Page - 1) * filter.PageSize
	if start > total {
		start = total
	}
	end := start + filter.PageSize
	if end > total {
		end = total
	}

	totalPages := 0
	if filter.PageSize > 0 {
		totalPages = (total + filter.PageSize - 1) / filter.PageSize
	}

	return &Page[Order]{
		Items:      matched[start:end],
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func matchesOrder(o Order, filter OrderFilter) bool {
	if filter.CustomerID != nil && o.CustomerID != *filter.CustomerID {
		return false
	}
	if filter.Status != nil && o.Status != *filter.Status {
		return false
	}
	if filter.StartDate != nil && o.CreatedAt.Before(*filter.StartDate) {
		return false
	}
	if filter.EndDate != nil && o.CreatedAt.After(*filter.EndDate) {
		return false
	}
	return true
}
