package order

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrors "order-service-go/internal/errors"
	"order-service-go/internal/money"
	"order-service-go/internal/pagination"
)

type recordingLogger struct {
	mu    sync.Mutex
	calls []uuid.UUID
}

func (l *recordingLogger) LogPublishFailureAfterCommit(orderID uuid.UUID, _ error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, orderID)
}

func (l *recordingLogger) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.calls)
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
	logger    *recordingLogger
	service   *Service
	customer  uuid.UUID
	productA  uuid.UUID
	productB  uuid.UUID
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	customerID := uuid.New()
	productA := uuid.New()
	productB := uuid.New()
	price := price8990(t)

	repo := NewInMemoryRepository()
	customers := &FakeCustomerLookup{Customers: map[uuid.UUID]Customer{
		customerID: {ID: customerID, Active: true},
	}}
	products := &FakeProductLookup{Products: map[uuid.UUID]Product{
		productA: {ID: productA, Price: price, Active: true},
		productB: {ID: productB, Price: price, Active: true},
	}}
	producer := &FakeProducer{}
	logger := &recordingLogger{}
	svc := NewService(repo, customers, products, producer, logger)
	svc.now = func() time.Time { return time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC) }

	return &testEnv{
		repo: repo, customers: customers, products: products,
		producer: producer, logger: logger, service: svc,
		customer: customerID, productA: productA, productB: productB,
	}
}

func validInput(env *testEnv) CreateOrderInput {
	return CreateOrderInput{
		CustomerID: env.customer,
		Items: []CreateOrderItemInput{
			{ProductID: env.productA, Quantity: 2},
			{ProductID: env.productB, Quantity: 1},
		},
	}
}

// BR-ORD-005 + architecture Ã‚Â§12 example totals.
func TestCreateValidOrderReturnsCreatedWithCorrectTotal(t *testing.T) {
	env := newTestEnv(t)
	out, err := env.service.Create(context.Background(), uuid.New(), validInput(env))
	if err != nil {
		t.Fatalf("Create unexpected error: %v", err)
	}
	if out.Status != StatusCreated {
		t.Fatalf("expected CREATED, got %s", out.Status)
	}
	if out.TotalAmount.Cents() != 26970 {
		t.Fatalf("expected total 269.70, got %d cents", out.TotalAmount.Cents())
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out.Items))
	}
	if out.Items[0].UnitPrice.Cents() != 8990 || out.Items[0].TotalPrice.Cents() != 17980 {
		t.Fatalf("unexpected first item: %+v", out.Items[0])
	}
	if out.Items[1].TotalPrice.Cents() != 8990 {
		t.Fatalf("unexpected second item: %+v", out.Items[1])
	}
}

// BR-PRD-005: unit price copied from product at creation.
func TestCreateCopiesUnitPriceFromProduct(t *testing.T) {
	env := newTestEnv(t)
	out, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.Items[0].UnitPrice.Cents() != 8990 {
		t.Fatalf("unit price not copied: %+v", out.Items[0])
	}
}

// BR-ORD-004: pure total helpers match service output.
func TestOrderTotalPureFunctionMatchesArchitectureExample(t *testing.T) {
	unit := price8990(t)
	totals := []money.Money{ItemTotal(unit, 2), ItemTotal(unit, 1)}
	if OrderTotal(totals).Cents() != 26970 {
		t.Fatal("OrderTotal mismatch")
	}
}

// BR-ORD-006 + Ã‚Â§20: publish after commit.
func TestCreatePublishesAfterCommit(t *testing.T) {
	env := newTestEnv(t)
	var events []string
	var mu sync.Mutex
	record := func(e string) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}
	env.repo.OnCreate = func(uuid.UUID) { record("commit") }
	env.producer.OnPublish = func(OrderCreatedMessage) { record("publish") }

	out, err := env.service.Create(context.Background(), uuid.New(), validInput(env))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if env.repo.CommitCount != 1 {
		t.Fatalf("expected 1 commit, got %d", env.repo.CommitCount)
	}
	msg, ok := env.producer.LastMessage()
	if !ok {
		t.Fatal("expected a published message")
	}
	if msg.OrderID != out.ID || msg.Event != OrderCreatedEvent {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if msg.CreatedAt != out.CreatedAt {
		t.Fatalf("created_at mismatch: %+v vs %+v", msg.CreatedAt, out.CreatedAt)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 || events[0] != "commit" || events[1] != "publish" {
		t.Fatalf("expected commit then publish, got %v", events)
	}
}

// BR-ORD-002
func TestCreateRejectsEmptyItems(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      nil,
	})
	assertCode(t, err, apperrors.CodeValidation)
}

// BR-ORD-003
func TestCreateRejectsZeroQuantity(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 0}},
	})
	assertCode(t, err, apperrors.CodeValidation)
}

// BR-ORD-001
func TestCreateRejectsMissingCustomer(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: uuid.Nil,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeValidation)
}

// BR-CUS-003
func TestCreateRejectsInactiveCustomer(t *testing.T) {
	env := newTestEnv(t)
	inactive := uuid.New()
	env.customers.Customers[inactive] = Customer{ID: inactive, Active: false}
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: inactive,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeInactiveCustomer)
}

// BR-PRD-004
func TestCreateRejectsInactiveProduct(t *testing.T) {
	env := newTestEnv(t)
	inactive := uuid.New()
	env.products.Products[inactive] = Product{ID: inactive, Price: price8990(t), Active: false}
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: inactive, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeInactiveProduct)
}

func TestCreateRejectsUnknownProduct(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: uuid.New(), Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeNotFound)
}

// commit ok + publish fail Ã¢â€ â€™ 500, logged, order NOT lost.
func TestCreateCommitOkPublishFailReturns500AndLogs(t *testing.T) {
	env := newTestEnv(t)
	env.producer.FailNext = true
	_, err := env.service.Create(context.Background(), uuid.New(), validInput(env))
	if err == nil {
		t.Fatal("expected error when publish fails")
	}
	assertCode(t, err, apperrors.CodeInternal)
	if env.logger.count() != 1 {
		t.Fatalf("expected inconsistency logged once, got %d", env.logger.count())
	}
	if env.repo.CommitCount != 1 {
		t.Fatal("order should still be persisted")
	}
	var orderID uuid.UUID
	for id := range env.repo.orders {
		orderID = id
		break
	}
	stored, items, findErr := env.repo.FindByID(context.Background(), orderID)
	if findErr != nil || stored == nil || len(items) == 0 {
		t.Fatal("order must not be lost after publish failure")
	}
	if len(env.producer.Messages) != 0 {
		t.Fatal("publish should have failed; no message recorded")
	}
}

func TestFindByIDReturnsOrder(t *testing.T) {
	env := newTestEnv(t)
	created, err := env.service.Create(context.Background(), uuid.New(), validInput(env))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := env.service.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID != created.ID || got.TotalAmount.Cents() != created.TotalAmount.Cents() {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestFindByIDUnknownReturnsNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.FindByID(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestListReturnsCreatedOrders(t *testing.T) {
	env := newTestEnv(t)
	if _, err := env.service.Create(context.Background(), uuid.New(), validInput(env)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	page, err := env.service.List(context.Background(), OrderFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("expected 1 order, got %+v", page)
	}
}

func TestListRejectsInvalidPagination(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.List(context.Background(), OrderFilter{Page: 1, PageSize: 200})
	assertCode(t, err, apperrors.CodeValidation)

	_, err = env.service.List(context.Background(), OrderFilter{Page: -1, PageSize: 20})
	assertCode(t, err, apperrors.CodeValidation)

	_, err = env.service.List(context.Background(), OrderFilter{Page: 1, PageSize: -1})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestListRepositoryErrorIsInternal(t *testing.T) {
	env := newTestEnv(t)
	env.service.repo = &brokenRepo{inner: env.repo}
	_, err := env.service.List(context.Background(), OrderFilter{Page: 1, PageSize: 20})
	assertCode(t, err, apperrors.CodeInternal)
}

type listEnrichBrokenRepo struct {
	inner *InMemoryRepository
}

func (r *listEnrichBrokenRepo) List(ctx context.Context, filter OrderFilter) (*pagination.Page[Order], error) {
	return &pagination.Page[Order]{
		Items:      []Order{{ID: uuid.New(), CustomerID: uuid.New(), Status: StatusCreated}},
		Page:       1,
		PageSize:   20,
		Total:      1,
		TotalPages: 1,
	}, nil
}

func (r *listEnrichBrokenRepo) FindByID(ctx context.Context, id uuid.UUID) (*Order, []OrderItem, error) {
	return nil, nil, errors.New("enrich failed")
}

func (r *listEnrichBrokenRepo) CreateWithItems(ctx context.Context, order *Order, items []OrderItem) error {
	return r.inner.CreateWithItems(ctx, order, items)
}

func (r *listEnrichBrokenRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error {
	return r.inner.UpdateStatus(ctx, id, status)
}

func (r *listEnrichBrokenRepo) MarkProcessed(ctx context.Context, id uuid.UUID, status OrderStatus, processedAt time.Time) error {
	return r.inner.MarkProcessed(ctx, id, status, processedAt)
}

func TestListEnrichFailureIsInternal(t *testing.T) {
	env := newTestEnv(t)
	env.service.repo = &listEnrichBrokenRepo{inner: env.repo}
	_, err := env.service.List(context.Background(), OrderFilter{Page: 1, PageSize: 20})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestFindByIDRepositoryInternalError(t *testing.T) {
	env := newTestEnv(t)
	env.service.repo = &brokenRepo{inner: env.repo}
	_, err := env.service.FindByID(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeInternal)
}

func TestListFiltersByCustomer(t *testing.T) {
	env := newTestEnv(t)
	if _, err := env.service.Create(context.Background(), uuid.New(), validInput(env)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	page, err := env.service.List(context.Background(), OrderFilter{CustomerID: &env.customer, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected 1 order for customer, got %d", page.Total)
	}
}

func TestCreateRejectsUnknownCustomer(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: uuid.New(),
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestCreateRejectsNilProductID(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: uuid.Nil, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestNewServiceUsesDefaultLoggerWhenNil(t *testing.T) {
	svc := NewService(NewInMemoryRepository(), &FakeCustomerLookup{}, &FakeProductLookup{}, &FakeProducer{}, nil)
	if svc.logger == nil {
		t.Fatal("expected default logger")
	}
}

func TestSlogInconsistencyLoggerRecordsFailure(t *testing.T) {
	l := &slogInconsistencyLogger{logger: slog.Default()}
	l.LogPublishFailureAfterCommit(uuid.New(), errors.New("publish failed"))
}

type failingCustomerLookup struct{}

func (failingCustomerLookup) FindByID(context.Context, uuid.UUID) (*Customer, error) {
	return nil, errors.New("db down")
}

func TestCreateCustomerLookupErrorIsInternal(t *testing.T) {
	env := newTestEnv(t)
	env.service.customers = failingCustomerLookup{}
	_, err := env.service.Create(context.Background(), uuid.New(), validInput(env))
	assertCode(t, err, apperrors.CodeInternal)
}

type failingProductLookup struct{}

func (failingProductLookup) FindManyByID(context.Context, []uuid.UUID) ([]Product, error) {
	return nil, errors.New("db down")
}

func TestCreateProductLookupErrorIsInternal(t *testing.T) {
	env := newTestEnv(t)
	env.service.products = failingProductLookup{}
	_, err := env.service.Create(context.Background(), uuid.New(), validInput(env))
	assertCode(t, err, apperrors.CodeInternal)
}

type brokenRepo struct {
	inner *InMemoryRepository
}

func (b *brokenRepo) CreateWithItems(ctx context.Context, order *Order, items []OrderItem) error {
	return errors.New("persist failed")
}

func (b *brokenRepo) FindByID(ctx context.Context, id uuid.UUID) (*Order, []OrderItem, error) {
	return nil, nil, errors.New("read failed")
}

func (b *brokenRepo) List(ctx context.Context, filter OrderFilter) (*pagination.Page[Order], error) {
	return nil, errors.New("list failed")
}

func (b *brokenRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error {
	return b.inner.UpdateStatus(ctx, id, status)
}

func (b *brokenRepo) MarkProcessed(ctx context.Context, id uuid.UUID, status OrderStatus, processedAt time.Time) error {
	return b.inner.MarkProcessed(ctx, id, status, processedAt)
}

func TestCreateRepositoryErrorIsInternal(t *testing.T) {
	env := newTestEnv(t)
	env.service.repo = &brokenRepo{inner: env.repo}
	_, err := env.service.Create(context.Background(), uuid.New(), validInput(env))
	assertCode(t, err, apperrors.CodeInternal)
}

func TestRepositoryUpdateStatusAndMarkProcessed(t *testing.T) {
	repo := NewInMemoryRepository()
	orderID := uuid.New()
	price := price8990(t)
	order := &Order{
		ID: orderID, CustomerID: uuid.New(), Status: StatusCreated,
		TotalAmount: price, CreatedBy: uuid.New(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	items := []OrderItem{{
		ID: uuid.New(), OrderID: orderID, ProductID: uuid.New(),
		Quantity: 1, UnitPrice: price, TotalPrice: price,
	}}
	if err := repo.CreateWithItems(context.Background(), order, items); err != nil {
		t.Fatalf("CreateWithItems: %v", err)
	}
	if err := repo.UpdateStatus(context.Background(), orderID, StatusProcessing); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	processedAt := time.Now().UTC()
	if err := repo.MarkProcessed(context.Background(), orderID, StatusPaid, processedAt); err != nil {
		t.Fatalf("MarkProcessed: %v", err)
	}
	stored, _, err := repo.FindByID(context.Background(), orderID)
	if err != nil || stored.Status != StatusPaid || stored.ProcessedAt == nil {
		t.Fatalf("unexpected stored order: %+v err=%v", stored, err)
	}
}

func TestRepositoryListWithStatusFilter(t *testing.T) {
	repo := NewInMemoryRepository()
	orderID := uuid.New()
	price := price8990(t)
	now := time.Now()
	order := &Order{
		ID: orderID, CustomerID: uuid.New(), Status: StatusCreated,
		TotalAmount: price, CreatedBy: uuid.New(),
		CreatedAt: now, UpdatedAt: now,
	}
	items := []OrderItem{{
		ID: uuid.New(), OrderID: orderID, ProductID: uuid.New(),
		Quantity: 1, UnitPrice: price, TotalPrice: price,
	}}
	if err := repo.CreateWithItems(context.Background(), order, items); err != nil {
		t.Fatalf("CreateWithItems: %v", err)
	}
	status := StatusPaid
	page, err := repo.List(context.Background(), OrderFilter{Status: &status, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected 0 paid orders, got %d", page.Total)
	}

	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	page, err = repo.List(context.Background(), OrderFilter{
		StartDate: &start, EndDate: &end, Page: 1, PageSize: 10,
	})
	if err != nil || page.Total != 1 {
		t.Fatalf("expected order in date range, got %+v err=%v", page, err)
	}
}

func TestRepositoryCreateWithItemsIsAtomic(t *testing.T) {
	repo := NewInMemoryRepository()
	orderID := uuid.New()
	price := price8990(t)
	order := &Order{
		ID: orderID, CustomerID: uuid.New(), Status: StatusCreated,
		TotalAmount: price, CreatedBy: uuid.New(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	items := []OrderItem{{
		ID: uuid.New(), OrderID: orderID, ProductID: uuid.New(),
		Quantity: 1, UnitPrice: price, TotalPrice: price,
	}}
	if err := repo.CreateWithItems(context.Background(), order, items); err != nil {
		t.Fatalf("CreateWithItems: %v", err)
	}
	if repo.CommitCount != 1 {
		t.Fatalf("expected 1 commit, got %d", repo.CommitCount)
	}
	_, storedItems, err := repo.FindByID(context.Background(), orderID)
	if err != nil || len(storedItems) != 1 {
		t.Fatal("order and items must be stored together")
	}
}
