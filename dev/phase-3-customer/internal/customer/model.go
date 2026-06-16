// Package customer implements customer management (architecture §6, §11, §17):
// the domain model, DTOs, a repository port with an in-memory adapter, the
// service holding the business rules (BR-CUS-001..004), and a thin HTTP handler.
package customer

import (
	"time"

	"github.com/google/uuid"
)

// Customer represents a buyer (architecture §6). Soft-deletion is modelled by
// Active; a customer is never physically removed (BR-CUS-004).
type Customer struct {
	ID        uuid.UUID
	Name      string
	Document  string
	Email     string
	Phone     string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
