package worker

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"order-service-go/internal/metrics"
	"order-service-go/internal/money"
	"order-service-go/internal/order"
)

type failingRepo struct {
	*order.InMemoryRepository
	failMark bool
}

func (r *failingRepo) MarkProcessed(ctx context.Context, id uuid.UUID, status order.OrderStatus, processedAt time.Time) error {
	if r.failMark {
		return errors.New("mark failed")
	}
	return r.InMemoryRepository.MarkProcessed(ctx, id, status, processedAt)
}

func TestWorkerLogsServiceProcessError(t *testing.T) {
	repo := &failingRepo{InMemoryRepository: order.NewInMemoryRepository(), failMark: true}
	customerID := uuid.New()
	productID := uuid.New()
	orderID := uuid.New()
	price, _ := money.ParseString("10.00")
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	o := &order.Order{
		ID: orderID, CustomerID: customerID, Status: order.StatusCreated,
		TotalAmount: price, CreatedBy: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	items := []order.OrderItem{{ID: uuid.New(), OrderID: orderID, ProductID: productID, Quantity: 1, UnitPrice: price, TotalPrice: price}}
	_ = repo.CreateWithItems(context.Background(), o, items)

	collector := metrics.NewCollector()
	svc := order.NewService(repo, nil, nil, &order.FakeProducer{}, nil, collector, slog.Default())
	queue := order.NewFakeQueue()
	retryQ := order.NewFakeRetryQueue(queue)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		New(queue, retryQ, svc, &order.FakePaymentProcessor{}, 3, 1, slog.Default()).Run(ctx)
		close(done)
	}()

	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	waitProcessed(t, repo.InMemoryRepository, orderID, order.StatusProcessing)
	cancel()
	<-done
}
