package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OrderCreatedEvent is the queue event name (architecture Â§9).
const OrderCreatedEvent = "ORDER_CREATED"

// OrderCreatedMessage is the Redis queue payload (architecture Â§9).
type OrderCreatedMessage struct {
	OrderID   uuid.UUID `json:"order_id"`
	Event     string    `json:"event"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderProducer publishes order-created messages after DB commit (BR-ORD-006).
type OrderProducer interface {
	PublishOrderCreated(ctx context.Context, msg OrderCreatedMessage) error
}

// FakeProducer is an in-memory OrderProducer for unit tests.
type FakeProducer struct {
	mu       sync.Mutex
	Messages []OrderCreatedMessage
	FailNext bool
	// OnPublish is called when PublishOrderCreated runs (after commit in service).
	OnPublish func(msg OrderCreatedMessage)
}

func (p *FakeProducer) PublishOrderCreated(_ context.Context, msg OrderCreatedMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.FailNext {
		p.FailNext = false
		return errors.New("publish failed")
	}
	p.Messages = append(p.Messages, msg)
	if p.OnPublish != nil {
		p.OnPublish(msg)
	}
	return nil
}

func (p *FakeProducer) LastMessage() (OrderCreatedMessage, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.Messages) == 0 {
		return OrderCreatedMessage{}, false
	}
	return p.Messages[len(p.Messages)-1], true
}
