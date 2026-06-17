package product

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "order-service-go/internal/errors"
	"order-service-go/internal/pagination"
)

const (
	// DefaultPageSize is used when the client omits page_size.
	DefaultPageSize = 20
	// MaxPageSize bounds the list so no caller can request an unbounded page.
	MaxPageSize = 100
)

// Service holds the product business rules (architecture Ã‚Â§17). It depends only
// on the ProductRepository port, so it is unit-tested with the in-memory
// adapter and no database.
type Service struct {
	repo ProductRepository
	now  func() time.Time
}

// NewService builds the product service over a repository.
func NewService(repo ProductRepository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Create registers a product. It enforces BR-PRD-001 (name required),
// BR-PRD-002 (sku unique), and BR-PRD-003 (price > 0).
func (s *Service) Create(ctx context.Context, input CreateProductInput) (*ProductOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperrors.Validation("product name is required")
	}
	sku := strings.TrimSpace(input.SKU)
	if sku == "" {
		return nil, apperrors.Validation("product sku is required")
	}
	if !input.Price.IsPositive() {
		return nil, apperrors.Validation("product price must be greater than zero")
	}
	if err := s.ensureSKUUnique(ctx, sku, uuid.Nil); err != nil {
		return nil, err
	}

	now := s.now()
	p := &Product{
		ID:        uuid.New(),
		Name:      name,
		SKU:       sku,
		Price:     input.Price,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, apperrors.Internal("could not create product", err)
	}
	out := toOutput(p)
	return &out, nil
}

// Update changes a product's mutable fields. It enforces BR-PRD-001, BR-PRD-002
// (uniqueness, excluding the product itself), and BR-PRD-003.
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*ProductOutput, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperrors.Validation("product name is required")
	}
	sku := strings.TrimSpace(input.SKU)
	if sku == "" {
		return nil, apperrors.Validation("product sku is required")
	}
	if !input.Price.IsPositive() {
		return nil, apperrors.Validation("product price must be greater than zero")
	}
	if sku != existing.SKU {
		if err := s.ensureSKUUnique(ctx, sku, id); err != nil {
			return nil, err
		}
	}

	existing.Name = name
	existing.SKU = sku
	existing.Price = input.Price
	existing.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, mapRepoErr(err)
	}
	out := toOutput(existing)
	return &out, nil
}

// Deactivate soft-deletes a product. The record remains retrievable with
// active=false.
func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Deactivate(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	return nil
}

// FindByID returns a single product or RESOURCE_NOT_FOUND.
func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*ProductOutput, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	out := toOutput(p)
	return &out, nil
}

// FindManyByID returns the products that exist for the given ids. Missing ids
// are omitted; order follows the repository implementation.
func (s *Service) FindManyByID(ctx context.Context, ids []uuid.UUID) ([]ProductOutput, error) {
	products, err := s.repo.FindManyByID(ctx, ids)
	if err != nil {
		return nil, apperrors.Internal("could not load products", err)
	}
	out := make([]ProductOutput, 0, len(products))
	for i := range products {
		out = append(out, toOutput(&products[i]))
	}
	return out, nil
}

// List returns a validated, bounded page of products matching the filter.
func (s *Service) List(ctx context.Context, filter ProductFilter) (*pagination.Page[ProductOutput], error) {
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = DefaultPageSize
	}
	if filter.Page < 1 {
		return nil, apperrors.Validation("page must be greater than zero")
	}
	if filter.PageSize < 1 {
		return nil, apperrors.Validation("page_size must be greater than zero")
	}
	if filter.PageSize > MaxPageSize {
		return nil, apperrors.Validation("page_size exceeds the maximum of 100")
	}

	page, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.Internal("could not list products", err)
	}

	items := make([]ProductOutput, 0, len(page.Items))
	for i := range page.Items {
		items = append(items, toOutput(&page.Items[i]))
	}
	return &pagination.Page[ProductOutput]{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		Total:      page.Total,
		TotalPages: page.TotalPages,
	}, nil
}

func (s *Service) ensureSKUUnique(ctx context.Context, sku string, excludeID uuid.UUID) error {
	existing, err := s.repo.FindBySKU(ctx, sku)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return apperrors.Internal("could not verify sku uniqueness", err)
	}
	if existing.ID != excludeID {
		return apperrors.Duplicate("a product with this sku already exists")
	}
	return nil
}

func mapRepoErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperrors.NotFound("product not found")
	}
	return apperrors.Internal("repository error", err)
}
