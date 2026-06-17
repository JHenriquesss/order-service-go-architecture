package customer

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
// customer does not exist. The service maps it to RESOURCE_NOT_FOUND.
var ErrNotFound = errors.New("customer not found")

// CustomerRepository is the persistence port. The service depends on this
// interface, never on a concrete store, so unit tests use the in-memory adapter
// and need no database.
type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	Update(ctx context.Context, c *Customer) error
	Deactivate(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	FindByDocument(ctx context.Context, document string) (*Customer, error)
	List(ctx context.Context, filter CustomerFilter) (*pagination.Page[Customer], error)
}

// InMemoryRepository is an in-memory CustomerRepository for unit tests. It is
// safe for concurrent use. Filtering is done in Go over stored values, so no
// SQL string concatenation exists anywhere in this phase.
type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[uuid.UUID]Customer
}

// NewInMemoryRepository builds an empty in-memory repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{items: make(map[uuid.UUID]Customer)}
}

func (r *InMemoryRepository) Create(_ context.Context, c *Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[c.ID] = *c
	return nil
}

func (r *InMemoryRepository) Update(_ context.Context, c *Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[c.ID]; !ok {
		return ErrNotFound
	}
	r.items[c.ID] = *c
	return nil
}

func (r *InMemoryRepository) Deactivate(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	c.Active = false
	r.items[id] = c
	return nil
}

func (r *InMemoryRepository) FindByID(_ context.Context, id uuid.UUID) (*Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &c, nil
}

func (r *InMemoryRepository) FindByDocument(_ context.Context, document string) (*Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.items {
		if c.Document == document {
			found := c
			return &found, nil
		}
	}
	return nil, ErrNotFound
}

func (r *InMemoryRepository) List(_ context.Context, filter CustomerFilter) (*pagination.Page[Customer], error) {
	r.mu.RLock()
	matched := make([]Customer, 0, len(r.items))
	for _, c := range r.items {
		if matches(c, filter) {
			matched = append(matched, c)
		}
	}
	r.mu.RUnlock()

	// Stable order so pagination is deterministic.
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

	return &pagination.Page[Customer]{
		Items:      matched[start:end],
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func matches(c Customer, filter CustomerFilter) bool {
	if filter.Name != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(filter.Name)) {
		return false
	}
	if filter.Document != "" && !strings.Contains(strings.ToLower(c.Document), strings.ToLower(filter.Document)) {
		return false
	}
	if filter.Active != nil && c.Active != *filter.Active {
		return false
	}
	return true
}
