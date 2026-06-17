package order

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFakeQueueEnqueueDequeue(t *testing.T) {
	q := NewFakeQueue()
	orderID := uuid.New()
	msg := OrderCreatedMessage{OrderID: orderID, Event: OrderCreatedEvent, CreatedAt: time.Now()}

	q.Enqueue(msg)
	got, err := q.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got.OrderID != orderID {
		t.Fatalf("order id mismatch")
	}
}

func TestFakeQueueCloseReturnsErrQueueClosed(t *testing.T) {
	q := NewFakeQueue()
	q.Close()
	_, err := q.Dequeue(context.Background())
	if err != ErrQueueClosed {
		t.Fatalf("err %v", err)
	}
}

func TestFakeQueueDequeueRespectsContextCancel(t *testing.T) {
	q := NewFakeQueue()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := q.Dequeue(ctx)
	if err != context.Canceled {
		t.Fatalf("err %v", err)
	}
}

func TestFakeProducerLastMessage(t *testing.T) {
	p := &FakeProducer{}
	msg := OrderCreatedMessage{OrderID: uuid.New(), Event: OrderCreatedEvent}
	_ = p.PublishOrderCreated(context.Background(), msg)
	last, ok := p.LastMessage()
	if !ok || last.OrderID != msg.OrderID {
		t.Fatal("expected last message")
	}
}
