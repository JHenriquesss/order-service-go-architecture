package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OrderCreatedEvent is the queue event name (architecture §9).
const OrderCreatedEvent = "ORDER_CREATED"

// OrderCreatedMessage is the Redis queue payload (architecture §9).
type OrderCreatedMessage struct {
	OrderID   uuid.UUID `json:"order_id"`
	Event     string    `json:"event"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderProducer publishes order-created messages after DB commit (BR-ORD-006).
type OrderProducer interface {
	PublishOrderCreated(ctx context.Context, msg OrderCreatedMessage) error
}

// OrderQueue consumes order-created messages for the worker pool.
type OrderQueue interface {
	Dequeue(ctx context.Context) (OrderCreatedMessage, error)
}

// ErrQueueClosed indicates the queue will not deliver more messages.
var ErrQueueClosed = errors.New("queue closed")

// FakeProducer is an in-memory OrderProducer for unit tests.
type FakeProducer struct {
	mu        sync.Mutex
	Messages  []OrderCreatedMessage
	FailNext  bool
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

// FakeQueue is a controllable in-memory OrderQueue for deterministic tests.
type FakeQueue struct {
	mu     sync.Mutex
	msgs   []OrderCreatedMessage
	notify chan struct{}
	closed bool
}

// NewFakeQueue builds an empty fake queue.
func NewFakeQueue() *FakeQueue {
	return &FakeQueue{notify: make(chan struct{}, 1)}
}

// Enqueue adds a message for workers to consume.
func (q *FakeQueue) Enqueue(msg OrderCreatedMessage) {
	q.mu.Lock()
	q.msgs = append(q.msgs, msg)
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

// Close stops delivering new messages; in-flight dequeue calls unblock.
func (q *FakeQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *FakeQueue) Dequeue(ctx context.Context) (OrderCreatedMessage, error) {
	for {
		q.mu.Lock()
		if len(q.msgs) > 0 {
			msg := q.msgs[0]
			q.msgs = q.msgs[1:]
			q.mu.Unlock()
			return msg, nil
		}
		if q.closed {
			q.mu.Unlock()
			return OrderCreatedMessage{}, ErrQueueClosed
		}
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			return OrderCreatedMessage{}, ctx.Err()
		case <-q.notify:
		}
	}
}
