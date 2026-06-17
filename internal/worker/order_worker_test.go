package worker

import (
	"context"
	"log/slog"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"

	"order-service-go/internal/metrics"
	"order-service-go/internal/money"
	"order-service-go/internal/order"
)

func setupWorkerTest(t *testing.T, workers int) (*order.InMemoryRepository, *order.FakeQueue, *metrics.Collector, *order.Service, uuid.UUID) {
	t.Helper()
	customerID := uuid.New()
	productID := uuid.New()
	orderID := uuid.New()
	price, _ := money.ParseString("10.00")
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	repo := order.NewInMemoryRepository()
	o := &order.Order{
		ID: orderID, CustomerID: customerID, Status: order.StatusCreated,
		TotalAmount: price, CreatedBy: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	items := []order.OrderItem{{
		ID: uuid.New(), OrderID: orderID, ProductID: productID,
		Quantity: 1, UnitPrice: price, TotalPrice: price,
	}}
	if err := repo.CreateWithItems(context.Background(), o, items); err != nil {
		t.Fatalf("seed order: %v", err)
	}

	collector := metrics.NewCollector()
	svc := order.NewService(repo, nil, nil, &order.FakeProducer{}, nil, collector, slog.Default())
	queue := order.NewFakeQueue()
	_ = workers
	return repo, queue, collector, svc, orderID
}

func TestWorkerProcessesCreatedOrderToPaid(t *testing.T) {
	repo, queue, collector, svc, orderID := setupWorkerTest(t, 1)
	proc := &order.FakePaymentProcessor{}
	w := New(queue, svc, proc, 1, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	waitProcessed(t, repo, orderID, order.StatusPaid)
	cancel()
	<-done

	if _, processed, _, _, count := collector.Snapshot(); processed != 1 || count != 1 {
		t.Fatalf("processed=%d count=%d", processed, count)
	}
}

func TestWorkerProcessingErrorIncrementsFailedMetric(t *testing.T) {
	repo, queue, collector, svc, orderID := setupWorkerTest(t, 1)
	proc := &order.FakePaymentProcessor{}
	proc.SetFailure(orderID, context.Canceled)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w := New(queue, svc, proc, 1, nil); w.Run(ctx); close(done) }()

	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	waitProcessed(t, repo, orderID, order.StatusFailed)
	cancel()
	<-done

	if _, _, failed, _, _ := collector.Snapshot(); failed != 1 {
		t.Fatalf("failed metric %d", failed)
	}
}

func TestWorkerIgnoresNonCreatedOrder(t *testing.T) {
	repo, queue, collector, svc, orderID := setupWorkerTest(t, 1)
	_ = repo.UpdateStatus(context.Background(), orderID, order.StatusPaid)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { New(queue, svc, &order.FakePaymentProcessor{}, 1, nil).Run(ctx); close(done) }()

	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	for i := 0; i < 1000; i++ {
		if _, processed, _, _, _ := collector.Snapshot(); processed > 0 {
			t.Fatal("processed non-CREATED order")
		}
		runtime.Gosched()
	}
	cancel()
	<-done

	if _, processed, _, _, _ := collector.Snapshot(); processed != 0 {
		t.Fatalf("processed %d, want 0", processed)
	}
}

func TestWorkerHandlesUnknownOrderIDWithoutCrash(t *testing.T) {
	_, queue, collector, svc, _ := setupWorkerTest(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { New(queue, svc, &order.FakePaymentProcessor{}, 1, nil).Run(ctx); close(done) }()

	queue.Enqueue(order.OrderCreatedMessage{OrderID: uuid.New(), Event: order.OrderCreatedEvent})
	queue.Enqueue(order.OrderCreatedMessage{OrderID: uuid.Nil, Event: order.OrderCreatedEvent})
	cancel()
	<-done

	if _, processed, _, _, _ := collector.Snapshot(); processed != 0 {
		t.Fatalf("processed %d", processed)
	}
}

func TestWorkerShutdownDrainsInFlightWork(t *testing.T) {
	repo, queue, collector, svc, orderID := setupWorkerTest(t, 1)
	started := make(chan struct{}, 1)
	proceed := make(chan struct{})
	proc := &order.FakePaymentProcessor{
		OnProcess:  func(uuid.UUID) { started <- struct{}{} },
		BlockUntil: proceed,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { New(queue, svc, proc, 1, nil).Run(ctx); close(done) }()

	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	<-started
	cancel()
	close(proceed)
	<-done

	waitProcessed(t, repo, orderID, order.StatusPaid)
	if _, processed, _, _, _ := collector.Snapshot(); processed != 1 {
		t.Fatalf("processed %d", processed)
	}
}

func TestWorkerPoolStartsMultipleWorkers(t *testing.T) {
	repo, queue, _, svc, orderID := setupWorkerTest(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { New(queue, svc, &order.FakePaymentProcessor{}, 2, nil).Run(ctx); close(done) }()

	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	waitProcessed(t, repo, orderID, order.StatusPaid)
	cancel()
	<-done
}

func waitProcessed(t *testing.T, repo *order.InMemoryRepository, orderID uuid.UUID, want order.OrderStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		o, _, err := repo.FindByID(context.Background(), orderID)
		if err == nil && o.Status == want {
			return
		}
		runtime.Gosched()
	}
	o, _, _ := repo.FindByID(context.Background(), orderID)
	t.Fatalf("order status %s, want %s", o.Status, want)
}
