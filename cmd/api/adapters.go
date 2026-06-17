package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"order-service-go/internal/customer"
	apperrors "order-service-go/internal/errors"
	"order-service-go/internal/order"
	"order-service-go/internal/product"
)

// customerLookup adapts the customer service to the order package's
// CustomerLookup port, translating a not-found AppError into order.ErrNotFound
// so the order service maps a missing customer to 404 rather than 500.
type customerLookup struct{ svc *customer.Service }

func (a customerLookup) FindByID(ctx context.Context, id uuid.UUID) (*order.Customer, error) {
	out, err := a.svc.FindByID(ctx, id)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) && appErr.Code == apperrors.CodeNotFound {
			return nil, order.ErrNotFound
		}
		return nil, err
	}
	return &order.Customer{ID: out.ID, Active: out.Active}, nil
}

// productLookup adapts the product service to the order package's ProductLookup
// port. Missing ids are omitted by the underlying service.
type productLookup struct{ svc *product.Service }

func (a productLookup) FindManyByID(ctx context.Context, ids []uuid.UUID) ([]order.Product, error) {
	loaded, err := a.svc.FindManyByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]order.Product, 0, len(loaded))
	for _, p := range loaded {
		out = append(out, order.Product{ID: p.ID, Price: p.Price, Active: p.Active})
	}
	return out, nil
}
