package customer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "order-service-customer/internal/errors"
)

const (
	// DefaultPageSize is used when the client omits page_size.
	DefaultPageSize = 20
	// MaxPageSize bounds the list so no caller can request an unbounded page.
	MaxPageSize = 100
)

// Service holds the customer business rules (architecture §17). It depends only
// on the CustomerRepository port, so it is unit-tested with the in-memory
// adapter and no database.
type Service struct {
	repo CustomerRepository
	now  func() time.Time
}

// NewService builds the customer service over a repository.
func NewService(repo CustomerRepository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Create registers a customer. It enforces BR-CUS-001 (name required) and
// BR-CUS-002 (document unique).
func (s *Service) Create(ctx context.Context, input CreateCustomerInput) (*CustomerOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperrors.Validation("customer name is required")
	}
	document := strings.TrimSpace(input.Document)
	if document == "" {
		return nil, apperrors.Validation("customer document is required")
	}
	if err := s.ensureDocumentUnique(ctx, document, uuid.Nil); err != nil {
		return nil, err
	}

	now := s.now()
	c := &Customer{
		ID:        uuid.New(),
		Name:      name,
		Document:  document,
		Email:     strings.TrimSpace(input.Email),
		Phone:     strings.TrimSpace(input.Phone),
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, apperrors.Internal("could not create customer", err)
	}
	out := toOutput(c)
	return &out, nil
}

// Update changes a customer's mutable fields. It enforces BR-CUS-001 and
// BR-CUS-002 (uniqueness, excluding the customer itself).
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateCustomerInput) (*CustomerOutput, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperrors.Validation("customer name is required")
	}
	document := strings.TrimSpace(input.Document)
	if document == "" {
		return nil, apperrors.Validation("customer document is required")
	}
	if document != existing.Document {
		if err := s.ensureDocumentUnique(ctx, document, id); err != nil {
			return nil, err
		}
	}

	existing.Name = name
	existing.Document = document
	existing.Email = strings.TrimSpace(input.Email)
	existing.Phone = strings.TrimSpace(input.Phone)
	existing.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, mapRepoErr(err)
	}
	out := toOutput(existing)
	return &out, nil
}

// Deactivate soft-deletes a customer (BR-CUS-004: never a physical delete). The
// record remains retrievable with active=false.
func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Deactivate(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	return nil
}

// FindByID returns a single customer or RESOURCE_NOT_FOUND.
func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*CustomerOutput, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	out := toOutput(c)
	return &out, nil
}

// List returns a validated, bounded page of customers matching the filter.
func (s *Service) List(ctx context.Context, filter CustomerFilter) (*Page[CustomerOutput], error) {
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
		return nil, apperrors.Internal("could not list customers", err)
	}

	items := make([]CustomerOutput, 0, len(page.Items))
	for i := range page.Items {
		items = append(items, toOutput(&page.Items[i]))
	}
	return &Page[CustomerOutput]{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		Total:      page.Total,
		TotalPages: page.TotalPages,
	}, nil
}

func (s *Service) ensureDocumentUnique(ctx context.Context, document string, excludeID uuid.UUID) error {
	existing, err := s.repo.FindByDocument(ctx, document)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return apperrors.Internal("could not verify document uniqueness", err)
	}
	if existing.ID != excludeID {
		return apperrors.Duplicate("a customer with this document already exists")
	}
	return nil
}

func mapRepoErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperrors.NotFound("customer not found")
	}
	return apperrors.Internal("repository error", err)
}
