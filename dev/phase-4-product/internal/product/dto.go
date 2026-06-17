package product

import (
	"time"

	"github.com/google/uuid"

	"order-service-product/internal/money"
)

// CreateProductInput is the decoded body of POST /api/products (architecture §12).
type CreateProductInput struct {
	Name  string      `json:"name"`
	SKU   string      `json:"sku"`
	Price money.Money `json:"price"`
}

// UpdateProductInput is the decoded body of PUT /api/products/{id}. It carries
// the mutable fields; the identifier comes from the path.
type UpdateProductInput struct {
	Name  string      `json:"name"`
	SKU   string      `json:"sku"`
	Price money.Money `json:"price"`
}

// ProductOutput is the public view returned by the API.
type ProductOutput struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	SKU       string      `json:"sku"`
	Price     money.Money `json:"price"`
	Active    bool        `json:"active"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// ProductFilter carries the list query parameters (architecture §13). Active is
// a pointer so "unset" is distinguishable from "false". Page and PageSize are
// validated and bounded by the service.
type ProductFilter struct {
	Name     string
	SKU      string
	Active   *bool
	Page     int
	PageSize int
}

// Page is the generic paginated result envelope (architecture §17).
type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func toOutput(p *Product) ProductOutput {
	return ProductOutput{
		ID:        p.ID,
		Name:      p.Name,
		SKU:       p.SKU,
		Price:     p.Price,
		Active:    p.Active,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
