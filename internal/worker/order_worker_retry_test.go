package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"order-service-go/internal/metrics"
	"order-service-go/internal/money"
	"order-service-go/internal/order"
)

type transientRepo struct {
	*order.InMemoryRepository
	mu        sync.Mutex
	failUntil int
	attempts  map[uuid.UUID]int
}

func newTransientRepo(inner *order.InMemoryRepository, failUntil int) *transientRepo {
	return &transientRepo{
		InMemoryRepository: inner,
		failUntil:          failUntil,
		attempts:           make(map[uuid.UUID]int),
	}
}

func (r *transientRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status order.OrderStatus) error {
	if status == order.StatusProcessing {
		r.mu.Lock()
		r.attempts[id]++
		attempt := r.attempts[id]
		r.mu.Unlock()
		if attempt <= r.failUntil {
			return errors.New("transient infrastructure failure")
		}
	}
	return r.InMemoryRepository.UpdateStatus(ctx, id, status)
}

func startWorker(t *testing.T, queue *order.FakeQueue, retryQ *order.FakeRetryQueue, svc *order.Service, proc order.PaymentProcessor, maxRetries int) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		newTestWorker(queue, retryQ, svc, proc, maxRetries, 1).Run(ctx)
		close(done)
	}()
	return cancel, done
}

func TestWorkerRetriesTransientFailureThenSucceeds(t *testing.T) {
	repo := order.NewInMemoryRepository()
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

	transient := newTransientRepo(repo, 1)
	collector := metrics.NewCollector()
	svc := order.NewService(transient, nil, nil, &order.FakeProducer{}, nil, collector, slog.Default())
	queue := order.NewFakeQueue()
	retryQ := order.NewFakeRetryQueue(queue)

	cancel, done := startWorker(t, queue, retryQ, svc, &order.FakePaymentProcessor{}, 3)
	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	waitProcessed(t, repo, orderID, order.StatusPaid)
	cancel()
	<-done

	if retryQ.RequeueCount() != 1 {
		t.Fatalf("requeue count=%d, want 1", retryQ.RequeueCount())
	}
	if retryQ.DeadLetterCount() != 0 {
		t.Fatalf("dead-letter count=%d, want 0", retryQ.DeadLetterCount())
	}
	if retryQ.Requeued[0].RetryCount != 1 {
		t.Fatalf("requeued retry_count=%d, want 1", retryQ.Requeued[0].RetryCount)
	}
}

func TestWorkerDeadLettersAfterMaxRetries(t *testing.T) {
	repo := order.NewInMemoryRepository()
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

	transient := newTransientRepo(repo, 100)
	collector := metrics.NewCollector()
	svc := order.NewService(transient, nil, nil, &order.FakeProducer{}, nil, collector, slog.Default())
	queue := order.NewFakeQueue()
	retryQ := order.NewFakeRetryQueue(queue)
	maxRetries := 3

	cancel, done := startWorker(t, queue, retryQ, svc, &order.FakePaymentProcessor{}, maxRetries)
	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if retryQ.DeadLetterCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if retryQ.DeadLetterCount() != 1 {
		t.Fatalf("dead-letter count=%d, want 1", retryQ.DeadLetterCount())
	}
	if retryQ.RequeueCount() != maxRetries {
		t.Fatalf("requeue count=%d, want %d", retryQ.RequeueCount(), maxRetries)
	}
	if retryQ.DeadLettered[0].RetryCount != maxRetries {
		t.Fatalf("dead-letter retry_count=%d, want %d", retryQ.DeadLettered[0].RetryCount, maxRetries)
	}
	o2, _, _ := repo.FindByID(context.Background(), orderID)
	if o2.Status != order.StatusCreated {
		t.Fatalf("order status=%s, want CREATED (never processed)", o2.Status)
	}
}

func TestPaymentDeclineIsNotRetried(t *testing.T) {
	repo, queue, retryQ, _, svc, orderID := setupWorkerTest(t, 1)
	proc := &order.FakePaymentProcessor{}
	proc.SetFailure(orderID, errors.New("payment declined"))

	cancel, done := startWorker(t, queue, retryQ, svc, proc, 3)
	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	waitProcessed(t, repo, orderID, order.StatusFailed)
	cancel()
	<-done

	if retryQ.RequeueCount() != 0 {
		t.Fatalf("requeue count=%d, want 0", retryQ.RequeueCount())
	}
	if retryQ.DeadLetterCount() != 0 {
		t.Fatalf("dead-letter count=%d, want 0", retryQ.DeadLetterCount())
	}
}

func TestProcessNilNeverRepublishes(t *testing.T) {
	repo, queue, retryQ, _, svc, orderID := setupWorkerTest(t, 1)
	_ = repo.UpdateStatus(context.Background(), orderID, order.StatusPaid)

	cancel, done := startWorker(t, queue, retryQ, svc, &order.FakePaymentProcessor{}, 3)
	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})
	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	if retryQ.RequeueCount() != 0 || retryQ.DeadLetterCount() != 0 {
		t.Fatalf("requeue=%d dead-letter=%d, want 0/0", retryQ.RequeueCount(), retryQ.DeadLetterCount())
	}
}

func TestOrderCreatedMessageRetryCountDefaultsToZero(t *testing.T) {
	raw := `{"order_id":"550e8400-e29b-41d4-a716-446655440000","event":"ORDER_CREATED","created_at":"2026-06-01T12:00:00Z"}`
	var msg order.OrderCreatedMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.RetryCount != 0 {
		t.Fatalf("retry_count=%d, want 0", msg.RetryCount)
	}
}

func TestMaxRetriesZeroDeadLettersImmediately(t *testing.T) {
	repo := order.NewInMemoryRepository()
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

	transient := newTransientRepo(repo, 100)
	svc := order.NewService(transient, nil, nil, &order.FakeProducer{}, nil, metrics.NewCollector(), slog.Default())
	queue := order.NewFakeQueue()
	retryQ := order.NewFakeRetryQueue(queue)

	cancel, done := startWorker(t, queue, retryQ, svc, &order.FakePaymentProcessor{}, 0)
	queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if retryQ.DeadLetterCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if retryQ.RequeueCount() != 0 {
		t.Fatalf("requeue count=%d, want 0", retryQ.RequeueCount())
	}
	if retryQ.DeadLetterCount() != 1 {
		t.Fatalf("dead-letter count=%d, want 1", retryQ.DeadLetterCount())
	}
}

func TestCapBoundaryNoOffByOneRequeue(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		wantReq    int
		wantDL     int
		finalCount int
	}{
		{name: "maxRetries=1", maxRetries: 1, wantReq: 1, wantDL: 1, finalCount: 1},
		{name: "maxRetries=3", maxRetries: 3, wantReq: 3, wantDL: 1, finalCount: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := order.NewInMemoryRepository()
			orderID := uuid.New()
			price, _ := money.ParseString("10.00")
			now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
			o := &order.Order{
				ID: orderID, CustomerID: uuid.New(), Status: order.StatusCreated,
				TotalAmount: price, CreatedBy: uuid.New(), CreatedAt: now, UpdatedAt: now,
			}
			items := []order.OrderItem{{ID: uuid.New(), OrderID: orderID, ProductID: uuid.New(), Quantity: 1, UnitPrice: price, TotalPrice: price}}
			_ = repo.CreateWithItems(context.Background(), o, items)

			transient := newTransientRepo(repo, 100)
			svc := order.NewService(transient, nil, nil, &order.FakeProducer{}, nil, metrics.NewCollector(), slog.Default())
			queue := order.NewFakeQueue()
			retryQ := order.NewFakeRetryQueue(queue)

			cancel, done := startWorker(t, queue, retryQ, svc, &order.FakePaymentProcessor{}, tc.maxRetries)
			queue.Enqueue(order.OrderCreatedMessage{OrderID: orderID, Event: order.OrderCreatedEvent})

			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if retryQ.DeadLetterCount() == tc.wantDL {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			cancel()
			<-done

			if retryQ.RequeueCount() != tc.wantReq {
				t.Fatalf("requeue count=%d, want %d", retryQ.RequeueCount(), tc.wantReq)
			}
			if retryQ.DeadLetterCount() != tc.wantDL {
				t.Fatalf("dead-letter count=%d, want %d", retryQ.DeadLetterCount(), tc.wantDL)
			}
			if tc.wantDL == 1 && retryQ.DeadLettered[0].RetryCount != tc.finalCount {
				t.Fatalf("dead-letter retry_count=%d, want %d", retryQ.DeadLettered[0].RetryCount, tc.finalCount)
			}
		})
	}
}
