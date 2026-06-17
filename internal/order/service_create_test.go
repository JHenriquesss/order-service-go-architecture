package order

import (
	"context"
	"testing"

	"github.com/google/uuid"

	apperrors "order-service-go/internal/errors"
)

func TestCreateValidatesCustomerID(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		Items: []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestCreateCustomerNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: uuid.New(),
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestCreateInactiveCustomer(t *testing.T) {
	env := newTestEnv(t)
	inactive := uuid.New()
	env.customers.Customers[inactive] = Customer{ID: inactive, Active: false}
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: inactive,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeInactiveCustomer)
}

func TestCreateInactiveProduct(t *testing.T) {
	env := newTestEnv(t)
	inactive := uuid.New()
	env.products.Products[inactive] = Product{ID: inactive, Price: price8990(t), Active: false}
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: inactive, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeInactiveProduct)
}

func TestCreateProductNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: uuid.New(), Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestCreateValidatesQuantity(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 0}},
	})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestCancelNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Cancel(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestShipNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Ship(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestListPageSizeValidation(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.List(context.Background(), OrderFilter{Page: 1, PageSize: 101})
	assertCode(t, err, apperrors.CodeValidation)
}
