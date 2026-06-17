package customer

import (
	"time"

	"github.com/google/uuid"
)

// CreateCustomerInput is the decoded body of POST /api/customers (architecture Â§12).
type CreateCustomerInput struct {
	Name     string `json:"name"`
	Document string `json:"document"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// UpdateCustomerInput is the decoded body of PUT /api/customers/{id}. It carries
// the mutable fields; the identifier comes from the path.
type UpdateCustomerInput struct {
	Name     string `json:"name"`
	Document string `json:"document"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// CustomerOutput is the public view returned by the API.
type CustomerOutput struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Document  string    `json:"document"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CustomerFilter carries the list query parameters (architecture Â§13). Active is
// a pointer so "unset" is distinguishable from "false". Page and PageSize are
// validated and bounded by the service.
type CustomerFilter struct {
	Name     string
	Document string
	Active   *bool
	Page     int
	PageSize int
}

func toOutput(c *Customer) CustomerOutput {
	return CustomerOutput{
		ID:        c.ID,
		Name:      c.Name,
		Document:  c.Document,
		Email:     c.Email,
		Phone:     c.Phone,
		Active:    c.Active,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
