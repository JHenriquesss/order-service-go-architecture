// Package order implements order creation (architecture §6, §7, §9, §19, §20):
// domain model, status rules, repository/queue ports, service, and HTTP handler.
package order

import (
	"time"

	"github.com/google/uuid"

	"order-service-order/internal/money"
)

// Order represents a customer purchase (architecture §6).
type Order struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	Status      OrderStatus
	TotalAmount money.Money
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProcessedAt *time.Time
}

// OrderItem represents one product inside an order (architecture §6).
type OrderItem struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	ProductID  uuid.UUID
	Quantity   int
	UnitPrice  money.Money
	TotalPrice money.Money
}

// ItemTotal computes quantity * unit_price without float64 (BR-ORD-004).
func ItemTotal(unitPrice money.Money, quantity int) money.Money {
	return unitPrice.Multiply(quantity)
}

// OrderTotal sums item totals (BR-ORD-004).
func OrderTotal(itemTotals []money.Money) money.Money {
	var total money.Money
	for _, item := range itemTotals {
		total = total.Add(item)
	}
	return total
}
