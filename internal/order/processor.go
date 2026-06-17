package order

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

// PaymentProcessor simulates payment processing for worker-driven lifecycle.
type PaymentProcessor interface {
	ProcessPayment(ctx context.Context, order *Order) error
}

// FakePaymentProcessor is a controllable processor for deterministic tests.
type FakePaymentProcessor struct {
	mu        sync.Mutex
	FailFor   map[uuid.UUID]error
	OnProcess func(orderID uuid.UUID)
	// BlockUntil released before ProcessPayment returns (for shutdown tests).
	BlockUntil chan struct{}
}

func (p *FakePaymentProcessor) ProcessPayment(ctx context.Context, order *Order) error {
	if p.OnProcess != nil {
		p.OnProcess(order.ID)
	}
	if p.BlockUntil != nil {
		<-p.BlockUntil
	}
	p.mu.Lock()
	err, fail := p.FailFor[order.ID]
	p.mu.Unlock()
	if fail {
		return err
	}
	return nil
}

// SetFailure configures the processor to fail for a specific order.
func (p *FakePaymentProcessor) SetFailure(orderID uuid.UUID, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.FailFor == nil {
		p.FailFor = make(map[uuid.UUID]error)
	}
	p.FailFor[orderID] = err
}

// AlwaysFailProcessor always returns an error.
type AlwaysFailProcessor struct{}

func (AlwaysFailProcessor) ProcessPayment(context.Context, *Order) error {
	return errors.New("payment failed")
}
