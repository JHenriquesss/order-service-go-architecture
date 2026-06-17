package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"order-service-go/internal/money"
	"order-service-go/internal/order"
)

// transientFailureRepo simulates infrastructure failures for orders whose total
// matches the configured sentinel amount. Used when ORDER_TRANSIENT_FAILURE_TOTAL
// is set (documented integration mechanism for dead-letter verification).
type transientFailureRepo struct {
	order.OrderRepository
	sentinel money.Money
}

func newTransientFailureRepo(inner order.OrderRepository, sentinel money.Money) *transientFailureRepo {
	return &transientFailureRepo{OrderRepository: inner, sentinel: sentinel}
}

func (r *transientFailureRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status order.OrderStatus) error {
	if status == order.StatusProcessing {
		o, _, err := r.OrderRepository.FindByID(ctx, id)
		if err == nil && o.TotalAmount.Cents() == r.sentinel.Cents() {
			return errors.New("simulated transient infrastructure failure")
		}
	}
	return r.OrderRepository.UpdateStatus(ctx, id, status)
}
