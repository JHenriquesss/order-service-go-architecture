package product

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"

	"order-service-go/internal/pagination"
)

// ErrNotFound is the domain not-found signal returned by the repository when a
// product does not exist. The service maps it to RESOURCE_NOT_FOUND.
var ErrNotFound = errors.New("product not found")

// ProductRepository is the persistence port. The service depends on this
// interface, never on a concrete store, so unit tests use the in-memory adapter
// and need no database.
type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	Update(ctx context.Context, p *Product) error
	Deactivate(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
	FindBySKU(ctx context.Context, sku string) (*Product, error)
	FindManyByID(ctx context.Context, ids []uuid.UUID) ([]Product, error)
	List(ctx context.Context, filter ProductFilter) (*pagination.Page[Product], error)
}

// InMemoryRepository is an in-memory ProductRepository for unit tests. It is
// safe for concurrent use. Filtering is done in Go over stored values, so no
// SQL string concatenation exists anywhere in this phase.
type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[uuid.UUID]Product
}

// NewInMemoryRepository builds an empty in-memory repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{items: make(map[uuid.UUID]Product)}
}

func (r *InMemoryRepository) Create(_ context.Context, p *Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[p.ID] = *p
	return nil
}

func (r *InMemoryRepository) Update(_ context.Context, p *Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[p.ID]; !ok {
		return ErrNotFound
	}
	r.items[p.ID] = *p
	return nil
}

func (r *InMemoryRepository) Deactivate(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	p.Active = false
	r.items[id] = p
	return nil
}

func (r *InMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (r *InMemoryRepository) FindBySKU(_ context.Context, sku string) (*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.items {
		if p.SKU == sku {
			found := p
			return &found, nil
		}
	}
	return nil, ErrNotFound
}

func (r *InMemoryRepository) FindManyByID(_ context.Context, ids []uuid.UUID) ([]Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Product, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.items[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *InMemoryRepository) List(_ context.Context, filter ProductFilter) (*pagination.Page[Product], error) {
	r.mu.RLock()
	matched := make([]Product, 0, len(r.items))
	for _, p := range r.items {
		if matches(p, filter) {
			matched = append(matched, p)
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

	totalPages := (total + filter.PageSize - 1) / filter.PageSize

	return &pagination.Page[Product]{
		Items:      matched[start:end],
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func matches(p Product, filter ProductFilter) bool {
	if filter.Name != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(filter.Name)) {
		return false
	}
	if filter.SKU != "" && !strings.Contains(strings.ToLower(p.SKU), strings.ToLower(filter.SKU)) {
		return false
	}
	if filter.Active != nil && p.Active != *filter.Active {
		return false
	}
	return true
}
