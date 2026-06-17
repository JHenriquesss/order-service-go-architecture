package order

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrors "order-service-worker/internal/errors"
	"order-service-worker/internal/metrics"
	"order-service-worker/internal/money"
)

type recordingMetrics struct {
	mu        sync.Mutex
	created   int
	processed int
	failed    int
	durations []float64
}

func (m *recordingMetrics) IncOrdersCreated()   { m.mu.Lock(); m.created++; m.mu.Unlock() }
func (m *recordingMetrics) IncOrdersProcessed() { m.mu.Lock(); m.processed++; m.mu.Unlock() }
func (m *recordingMetrics) IncOrdersFailed()    { m.mu.Lock(); m.failed++; m.mu.Unlock() }
func (m *recordingMetrics) RecordProcessingDuration(s float64) {
	m.mu.Lock()
	m.durations = append(m.durations, s)
	m.mu.Unlock()
}

type logRecorder struct {
	mu   sync.Mutex
	logs []map[string]string
}

type logRecorderHandler struct {
	rec *logRecorder
}

func (h logRecorderHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h logRecorderHandler) Handle(_ context.Context, r slog.Record) error {
	entry := map[string]string{"msg": r.Message}
	r.Attrs(func(a slog.Attr) bool {
		entry[a.Key] = a.Value.String()
		return true
	})
	h.rec.mu.Lock()
	h.rec.logs = append(h.rec.logs, entry)
	h.rec.mu.Unlock()
	return nil
}

func (h logRecorderHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h logRecorderHandler) WithGroup(string) slog.Handler      { return h }

func (l *logRecorder) handler() slog.Handler {
	return logRecorderHandler{rec: l}
}

func (l *logRecorder) hasErrorWithOrderID(orderID uuid.UUID) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	want := orderID.String()
	for _, entry := range l.logs {
		if entry["msg"] == "order processing failed" && entry["order_id"] == want {
			return true
		}
	}
	return false
}

func price8990(t *testing.T) money.Money {
	t.Helper()
	p, err := money.ParseString("89.90")
	if err != nil {
		t.Fatalf("parse price: %v", err)
	}
	return p
}

func assertCode(t *testing.T, err error, want apperrors.Code) {
	t.Helper()
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	if appErr.Code != want {
		t.Fatalf("expected code %s, got %s", want, appErr.Code)
	}
}

type testEnv struct {
	repo      *InMemoryRepository
	customers *FakeCustomerLookup
	products  *FakeProductLookup
	producer  *FakeProducer
	metrics   *recordingMetrics
	logs      *logRecorder
	service   *Service
	customer  uuid.UUID
	productA  uuid.UUID
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	customerID := uuid.New()
	productA := uuid.New()
	price := price8990(t)

	repo := NewInMemoryRepository()
	customers := &FakeCustomerLookup{Customers: map[uuid.UUID]Customer{
		customerID: {ID: customerID, Active: true},
	}}
	products := &FakeProductLookup{Products: map[uuid.UUID]Product{
		productA: {ID: productA, Price: price, Active: true},
	}}
	producer := &FakeProducer{}
	m := &recordingMetrics{}
	logs := &logRecorder{}
	logger := slog.New(logs.handler())
	svc := NewService(repo, customers, products, producer, nil, m, logger)
	fixed := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixed }

	return &testEnv{
		repo: repo, customers: customers, products: products,
		producer: producer, metrics: m, logs: logs, service: svc,
		customer: customerID, productA: productA,
	}
}

func seedCreatedOrder(t *testing.T, env *testEnv) uuid.UUID {
	t.Helper()
	out, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return out.ID
}

func TestProcessCreatedOrderToPaidIncrementsProcessedMetric(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)

	if err := env.service.Process(context.Background(), orderID, &FakePaymentProcessor{}); err != nil {
		t.Fatalf("process: %v", err)
	}
	order, _, err := env.repo.FindByID(context.Background(), orderID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if order.Status != StatusPaid {
		t.Fatalf("status %s, want PAID", order.Status)
	}
	env.metrics.mu.Lock()
	defer env.metrics.mu.Unlock()
	if env.metrics.processed != 1 {
		t.Fatalf("processed metric %d, want 1", env.metrics.processed)
	}
	if len(env.metrics.durations) != 1 {
		t.Fatalf("duration samples %d, want 1", len(env.metrics.durations))
	}
}

func TestProcessErrorMarksFailedAndIncrementsFailedMetric(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	proc := &FakePaymentProcessor{}
	proc.SetFailure(orderID, errors.New("card declined"))

	if err := env.service.Process(context.Background(), orderID, proc); err != nil {
		t.Fatalf("process: %v", err)
	}
	order, _, _ := env.repo.FindByID(context.Background(), orderID)
	if order.Status != StatusFailed {
		t.Fatalf("status %s, want FAILED", order.Status)
	}
	env.metrics.mu.Lock()
	if env.metrics.failed != 1 {
		t.Fatalf("failed metric %d, want 1", env.metrics.failed)
	}
	env.metrics.mu.Unlock()
	if !env.logs.hasErrorWithOrderID(orderID) {
		t.Fatal("expected error log with order id")
	}
}

func TestProcessIgnoresNonCreatedOrderBRORD009(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	_ = env.repo.UpdateStatus(context.Background(), orderID, StatusPaid)

	if err := env.service.Process(context.Background(), orderID, &FakePaymentProcessor{}); err != nil {
		t.Fatalf("process: %v", err)
	}
	env.metrics.mu.Lock()
	if env.metrics.processed != 0 {
		t.Fatalf("processed metric %d, want 0", env.metrics.processed)
	}
	env.metrics.mu.Unlock()
}

func TestProcessIgnoresUnknownOrderID(t *testing.T) {
	env := newTestEnv(t)
	if err := env.service.Process(context.Background(), uuid.New(), &FakePaymentProcessor{}); err != nil {
		t.Fatalf("process: %v", err)
	}
}

func TestProcessIgnoresNilOrderID(t *testing.T) {
	env := newTestEnv(t)
	if err := env.service.Process(context.Background(), uuid.Nil, &FakePaymentProcessor{}); err != nil {
		t.Fatalf("process: %v", err)
	}
}

func TestCancelCreatedOrderBRORD007(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)

	out, err := env.service.Cancel(context.Background(), orderID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if out.Status != StatusCanceled {
		t.Fatalf("status %s", out.Status)
	}
}

func TestCancelPaidOrderInvalidBRORD007(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	_ = env.repo.UpdateStatus(context.Background(), orderID, StatusPaid)

	_, err := env.service.Cancel(context.Background(), orderID)
	assertCode(t, err, apperrors.CodeInvalidOrderStatus)
}

func TestShipPaidOrderBRORD008(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	_ = env.repo.UpdateStatus(context.Background(), orderID, StatusPaid)

	out, err := env.service.Ship(context.Background(), orderID)
	if err != nil {
		t.Fatalf("ship: %v", err)
	}
	if out.Status != StatusShipped {
		t.Fatalf("status %s", out.Status)
	}
}

func TestShipCreatedOrderInvalidBRORD008(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)

	_, err := env.service.Ship(context.Background(), orderID)
	assertCode(t, err, apperrors.CodeInvalidOrderStatus)
}

func TestCreateIncrementsCreatedMetric(t *testing.T) {
	env := newTestEnv(t)
	_ = seedCreatedOrder(t, env)
	env.metrics.mu.Lock()
	defer env.metrics.mu.Unlock()
	if env.metrics.created != 1 {
		t.Fatalf("created metric %d", env.metrics.created)
	}
}

func TestMetricsCollectorIntegration(t *testing.T) {
	c := metrics.NewCollector()
	_ = NewService(NewInMemoryRepository(), nil, nil, nil, nil, c, nil)
}

func TestProcessTransitionsThroughProcessing(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	var sawProcessing bool
	proc := &FakePaymentProcessor{
		OnProcess: func(id uuid.UUID) {
			o, _, _ := env.repo.FindByID(context.Background(), id)
			if o.Status == StatusProcessing {
				sawProcessing = true
			}
		},
	}
	if err := env.service.Process(context.Background(), orderID, proc); err != nil {
		t.Fatalf("process: %v", err)
	}
	if !sawProcessing {
		t.Fatal("expected PROCESSING state during payment")
	}
}

func TestProcessFailureBRORD010(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	if err := env.service.Process(context.Background(), orderID, AlwaysFailProcessor{}); err != nil {
		t.Fatalf("process: %v", err)
	}
	order, _, _ := env.repo.FindByID(context.Background(), orderID)
	if order.Status != StatusFailed {
		t.Fatalf("status %s, want FAILED", order.Status)
	}
}

func TestInconsistencyLogger(t *testing.T) {
	env := newTestEnv(t)
	env.producer.FailNext = true
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	if err == nil {
		t.Fatal("expected publish failure")
	}
}
