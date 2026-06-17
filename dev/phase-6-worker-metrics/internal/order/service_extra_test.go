package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "order-service-worker/internal/errors"
)

func TestFindByIDReturnsOrder(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	out, err := env.service.FindByID(context.Background(), orderID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if out.ID != orderID {
		t.Fatalf("id mismatch")
	}
}

func TestFindByIDNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.FindByID(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestListReturnsPage(t *testing.T) {
	env := newTestEnv(t)
	_ = seedCreatedOrder(t, env)
	page, err := env.service.List(context.Background(), OrderFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items %d", len(page.Items))
	}
}

func TestListValidationRejectsInvalidPage(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.List(context.Background(), OrderFilter{Page: -1, PageSize: 10})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestMapOrderRepoErrInternal(t *testing.T) {
	err := mapOrderRepoErr(errors.New("db down"))
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeInternal {
		t.Fatalf("expected internal, got %v", err)
	}
}

func TestNoopMetrics(t *testing.T) {
	var m noopMetrics
	m.IncOrdersCreated()
	m.IncOrdersProcessed()
	m.IncOrdersFailed()
	m.RecordProcessingDuration(1)
}
