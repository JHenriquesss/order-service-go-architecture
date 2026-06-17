package order

import (
	"time"

	"github.com/google/uuid"

	"order-service-go/internal/money"
)

// CreateOrderInput is the decoded body of POST /api/orders (architecture Ã‚Â§12).
type CreateOrderInput struct {
	CustomerID uuid.UUID              `json:"customer_id"`
	Items      []CreateOrderItemInput `json:"items"`
}

// CreateOrderItemInput is one line on a create-order request.
type CreateOrderItemInput struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// OrderOutput is the public view returned by the API (architecture Ã‚Â§12).
type OrderOutput struct {
	ID          uuid.UUID         `json:"id"`
	CustomerID  uuid.UUID         `json:"customer_id"`
	Status      OrderStatus       `json:"status"`
	TotalAmount money.Money       `json:"total_amount"`
	Items       []OrderItemOutput `json:"items"`
	CreatedAt   time.Time         `json:"created_at"`
}

// OrderItemOutput is the public view of an order line.
type OrderItemOutput struct {
	ProductID  uuid.UUID   `json:"product_id"`
	Quantity   int         `json:"quantity"`
	UnitPrice  money.Money `json:"unit_price"`
	TotalPrice money.Money `json:"total_price"`
}

// OrderFilter carries list query parameters (architecture Ã‚Â§13).
type OrderFilter struct {
	CustomerID *uuid.UUID
	Status     *OrderStatus
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}

func toOutput(order *Order, items []OrderItem) OrderOutput {
	itemOut := make([]OrderItemOutput, 0, len(items))
	for _, item := range items {
		itemOut = append(itemOut, OrderItemOutput{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			UnitPrice:  item.UnitPrice,
			TotalPrice: item.TotalPrice,
		})
	}
	return OrderOutput{
		ID:          order.ID,
		CustomerID:  order.CustomerID,
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
		Items:       itemOut,
		CreatedAt:   order.CreatedAt,
	}
}
