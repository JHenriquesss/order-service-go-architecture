package order

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"order-service-worker/internal/money"

	apperrors "order-service-worker/internal/errors"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// InconsistencyLogger records publish failures after a successful DB commit.
type InconsistencyLogger interface {
	LogPublishFailureAfterCommit(orderID uuid.UUID, err error)
}

type slogInconsistencyLogger struct {
	logger *slog.Logger
}

func (l *slogInconsistencyLogger) LogPublishFailureAfterCommit(orderID uuid.UUID, err error) {
	l.logger.Error("order created but queue publish failed",
		"order_id", orderID.String(),
		"error", err.Error(),
	)
}

// Service holds order business rules (architecture §19, §20).
type Service struct {
	repo       OrderRepository
	customers  CustomerLookup
	products   ProductLookup
	producer   OrderProducer
	logger     InconsistencyLogger
	metrics    Metrics
	processLog *slog.Logger
	now        func() time.Time
}

// NewService builds the order service over its ports.
func NewService(
	repo OrderRepository,
	customers CustomerLookup,
	products ProductLookup,
	producer OrderProducer,
	logger InconsistencyLogger,
	metrics Metrics,
	processLog *slog.Logger,
) *Service {
	if logger == nil {
		logger = &slogInconsistencyLogger{logger: slog.Default()}
	}
	if metrics == nil {
		metrics = noopMetrics{}
	}
	if processLog == nil {
		processLog = slog.Default()
	}
	return &Service{
		repo:       repo,
		customers:  customers,
		products:   products,
		producer:   producer,
		logger:     logger,
		metrics:    metrics,
		processLog: processLog,
		now:        time.Now,
	}
}

// Create validates input, loads customer/products, copies prices, computes totals,
// persists atomically, publishes after commit, and returns the created order.
func (s *Service) Create(ctx context.Context, createdBy uuid.UUID, input CreateOrderInput) (*OrderOutput, error) {
	if input.CustomerID == uuid.Nil {
		return nil, apperrors.Validation("customer_id is required")
	}
	if len(input.Items) == 0 {
		return nil, apperrors.Validation("order must have at least one item")
	}
	for _, item := range input.Items {
		if item.Quantity < 1 {
			return nil, apperrors.Validation("item quantity must be greater than zero")
		}
		if item.ProductID == uuid.Nil {
			return nil, apperrors.Validation("product_id is required")
		}
	}

	customer, err := s.customers.FindByID(ctx, input.CustomerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperrors.NotFound("customer not found")
		}
		return nil, apperrors.Internal("could not load customer", err)
	}
	if !customer.Active {
		return nil, apperrors.InactiveCustomer("inactive customers cannot create new orders")
	}

	productIDs := make([]uuid.UUID, 0, len(input.Items))
	for _, item := range input.Items {
		productIDs = append(productIDs, item.ProductID)
	}
	loaded, err := s.products.FindManyByID(ctx, productIDs)
	if err != nil {
		return nil, apperrors.Internal("could not load products", err)
	}
	productByID := make(map[uuid.UUID]Product, len(loaded))
	for _, p := range loaded {
		productByID[p.ID] = p
	}

	itemTotals := make([]money.Money, 0, len(input.Items))
	orderItems := make([]OrderItem, 0, len(input.Items))
	now := s.now()
	orderID := uuid.New()

	for _, line := range input.Items {
		product, ok := productByID[line.ProductID]
		if !ok {
			return nil, apperrors.NotFound("product not found")
		}
		if !product.Active {
			return nil, apperrors.InactiveProduct("inactive products cannot be used in new orders")
		}
		unitPrice := product.Price
		lineTotal := ItemTotal(unitPrice, line.Quantity)
		itemTotals = append(itemTotals, lineTotal)
		orderItems = append(orderItems, OrderItem{
			ID:         uuid.New(),
			OrderID:    orderID,
			ProductID:  line.ProductID,
			Quantity:   line.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: lineTotal,
		})
	}

	order := &Order{
		ID:          orderID,
		CustomerID:  input.CustomerID,
		Status:      StatusCreated,
		TotalAmount: OrderTotal(itemTotals),
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateWithItems(ctx, order, orderItems); err != nil {
		return nil, apperrors.Internal("could not create order", err)
	}

	msg := OrderCreatedMessage{
		OrderID:   order.ID,
		Event:     OrderCreatedEvent,
		CreatedAt: order.CreatedAt,
	}
	if err := s.producer.PublishOrderCreated(ctx, msg); err != nil {
		s.logger.LogPublishFailureAfterCommit(order.ID, err)
		return nil, apperrors.Internal("order was created but could not be queued for processing", err)
	}

	s.metrics.IncOrdersCreated()
	out := toOutput(order, orderItems)
	return &out, nil
}

// Process drives CREATED→PROCESSING→PAID|FAILED (BR-ORD-009, BR-ORD-010).
// Non-CREATED orders and missing orders are ignored without error.
func (s *Service) Process(ctx context.Context, orderID uuid.UUID, processor PaymentProcessor) error {
	if orderID == uuid.Nil {
		s.processLog.Warn("ignoring queue message with empty order id")
		return nil
	}

	order, _, err := s.repo.FindByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			s.processLog.Warn("ignoring queue message for unknown order", "order_id", orderID.String())
			return nil
		}
		return apperrors.Internal("could not load order for processing", err)
	}

	if order.Status != StatusCreated {
		return nil
	}

	if !CanTransition(order.Status, StatusProcessing) {
		return apperrors.InvalidOrderStatus("order cannot enter processing from its current status")
	}
	if err := s.repo.UpdateStatus(ctx, orderID, StatusProcessing); err != nil {
		return mapOrderRepoErr(err)
	}

	start := s.now()
	procErr := processor.ProcessPayment(ctx, order)
	duration := s.now().Sub(start).Seconds()

	if procErr != nil {
		processedAt := s.now()
		if err := s.repo.MarkProcessed(ctx, orderID, StatusFailed, processedAt); err != nil {
			return mapOrderRepoErr(err)
		}
		s.metrics.IncOrdersFailed()
		s.processLog.Error("order processing failed",
			"order_id", orderID.String(),
			"error", procErr.Error(),
		)
		return nil
	}

	processedAt := s.now()
	if err := s.repo.MarkProcessed(ctx, orderID, StatusPaid, processedAt); err != nil {
		return mapOrderRepoErr(err)
	}
	s.metrics.IncOrdersProcessed()
	s.metrics.RecordProcessingDuration(duration)
	return nil
}

// Cancel moves a CREATED order to CANCELED (BR-ORD-007).
func (s *Service) Cancel(ctx context.Context, id uuid.UUID) (*OrderOutput, error) {
	return s.transition(ctx, id, StatusCanceled, StatusCreated, "only CREATED orders can be canceled")
}

// Ship moves a PAID order to SHIPPED (BR-ORD-008).
func (s *Service) Ship(ctx context.Context, id uuid.UUID) (*OrderOutput, error) {
	return s.transition(ctx, id, StatusShipped, StatusPaid, "only PAID orders can be shipped")
}

func (s *Service) transition(ctx context.Context, id uuid.UUID, to, required OrderStatus, invalidMsg string) (*OrderOutput, error) {
	order, items, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapOrderRepoErr(err)
	}
	if order.Status != required {
		return nil, apperrors.InvalidOrderStatus(invalidMsg)
	}
	if !CanTransition(order.Status, to) {
		return nil, apperrors.InvalidOrderStatus(invalidMsg)
	}
	if err := s.repo.UpdateStatus(ctx, id, to); err != nil {
		return nil, mapOrderRepoErr(err)
	}
	order.Status = to
	order.UpdatedAt = s.now()
	out := toOutput(order, items)
	return &out, nil
}

// FindByID returns a single order with its items.
func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*OrderOutput, error) {
	order, items, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapOrderRepoErr(err)
	}
	out := toOutput(order, items)
	return &out, nil
}

// List returns a validated, bounded page of orders matching the filter.
func (s *Service) List(ctx context.Context, filter OrderFilter) (*Page[OrderOutput], error) {
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = DefaultPageSize
	}
	if filter.Page < 1 {
		return nil, apperrors.Validation("page must be greater than zero")
	}
	if filter.PageSize < 1 {
		return nil, apperrors.Validation("page_size must be greater than zero")
	}
	if filter.PageSize > MaxPageSize {
		return nil, apperrors.Validation("page_size exceeds the maximum of 100")
	}

	page, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.Internal("could not list orders", err)
	}

	items := make([]OrderOutput, 0, len(page.Items))
	for i := range page.Items {
		_, orderItems, err := s.repo.FindByID(ctx, page.Items[i].ID)
		if err != nil {
			return nil, mapOrderRepoErr(err)
		}
		items = append(items, toOutput(&page.Items[i], orderItems))
	}

	return &Page[OrderOutput]{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		Total:      page.Total,
		TotalPages: page.TotalPages,
	}, nil
}

func mapOrderRepoErr(err error) error {
	if errors.Is(err, ErrOrderNotFound) {
		return apperrors.NotFound("order not found")
	}
	return apperrors.Internal("repository error", err)
}
