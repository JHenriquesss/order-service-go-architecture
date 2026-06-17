package queue

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"

	"order-service-go/internal/order"
)

// OrderQueueName is the Redis list the API pushes order-created events onto and
// the worker consumes from (architecture §9).
const OrderQueueName = "orders:processing"

// OrderProducer publishes order-created messages to Redis (architecture §9, §20).
// It satisfies order.OrderProducer.
type OrderProducer struct {
	client *redis.Client
	queue  string
}

// NewOrderProducer builds a Redis-backed order producer over an existing client.
func NewOrderProducer(client *redis.Client) *OrderProducer {
	return &OrderProducer{client: client, queue: OrderQueueName}
}

// PublishOrderCreated marshals the message and LPUSHes it onto the queue. It is
// called only after the order transaction has committed (architecture §20).
func (p *OrderProducer) PublishOrderCreated(ctx context.Context, msg order.OrderCreatedMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.client.LPush(ctx, p.queue, payload).Err()
}
