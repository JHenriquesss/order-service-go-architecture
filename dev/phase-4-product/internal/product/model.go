// Package product implements product management (architecture §6, §11, §17):
// the domain model, DTOs, a repository port with an in-memory adapter, the
// service holding the business rules (BR-PRD-001..003), and a thin HTTP handler.
package product

import (
	"time"

	"github.com/google/uuid"

	"order-service-product/internal/money"
)

// Product represents a sellable item (architecture §6). Soft-deletion is modelled
// by Active; a product is never physically removed.
type Product struct {
	ID        uuid.UUID
	Name      string
	SKU       string
	Price     money.Money
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
