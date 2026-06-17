package order

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"order-service-worker/internal/money"
)

// ErrNotFound is returned when a customer, product, or order does not exist.
var ErrNotFound = errors.New("resource not found")

// Customer is the minimal customer view this phase needs (architecture §6).
type Customer struct {
	ID     uuid.UUID
	Active bool
}

// Product is the minimal product view this phase needs (architecture §6).
type Product struct {
	ID     uuid.UUID
	Price  money.Money
	Active bool
}

// CustomerLookup loads customers for order creation (BR-CUS-003).
type CustomerLookup interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)
}

// ProductLookup loads products for order creation (BR-PRD-004, BR-PRD-005).
type ProductLookup interface {
	FindManyByID(ctx context.Context, ids []uuid.UUID) ([]Product, error)
}

// Metrics records operational counters (architecture §22).
type Metrics interface {
	IncOrdersCreated()
	IncOrdersProcessed()
	IncOrdersFailed()
	RecordProcessingDuration(seconds float64)
}

type noopMetrics struct{}

func (noopMetrics) IncOrdersCreated()                {}
func (noopMetrics) IncOrdersProcessed()              {}
func (noopMetrics) IncOrdersFailed()                 {}
func (noopMetrics) RecordProcessingDuration(float64) {}

// FakeCustomerLookup is an in-memory CustomerLookup for unit tests.
type FakeCustomerLookup struct {
	Customers map[uuid.UUID]Customer
}

func (f *FakeCustomerLookup) FindByID(_ context.Context, id uuid.UUID) (*Customer, error) {
	c, ok := f.Customers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &c, nil
}

// FakeProductLookup is an in-memory ProductLookup for unit tests.
type FakeProductLookup struct {
	Products map[uuid.UUID]Product
}

func (f *FakeProductLookup) FindManyByID(_ context.Context, ids []uuid.UUID) ([]Product, error) {
	out := make([]Product, 0, len(ids))
	for _, id := range ids {
		p, ok := f.Products[id]
		if !ok {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}
