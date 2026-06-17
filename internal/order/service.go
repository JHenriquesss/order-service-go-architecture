package order

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"order-service-go/internal/money"
	"order-service-go/internal/pagination"

	apperrors "order-service-go/internal/errors"
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

// Service holds order business rules (architecture Ã‚Â§19, Ã‚Â§20).
type Service struct {
	repo      OrderRepository
	customers CustomerLookup
	products  ProductLookup
	producer  OrderProducer
	logger    InconsistencyLogger
	now       func() time.Time
}

// NewService builds the order service over its ports.
func NewService(
	repo OrderRepository,
	customers CustomerLookup,
	products ProductLookup,
	producer OrderProducer,
	logger InconsistencyLogger,
) *Service {
	if logger == nil {
		logger = &slogInconsistencyLogger{logger: slog.Default()}
	}
	return &Service{
		repo:      repo,
		customers: customers,
		products:  products,
		producer:  producer,
		logger:    logger,
		now:       time.Now,
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

	out := toOutput(order, orderItems)
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
func (s *Service) List(ctx context.Context, filter OrderFilter) (*pagination.Page[OrderOutput], error) {
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

	return &pagination.Page[OrderOutput]{
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
