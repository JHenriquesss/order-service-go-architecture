package queue

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"

	"order-service-go/internal/order"
)

// OrderConsumer reads order-created messages from Redis for the worker pool. It
// satisfies order.OrderQueue. Producer LPUSHes and consumer BRPOPs, giving FIFO.
type OrderConsumer struct {
	client *redis.Client
	queue  string
}

// NewOrderConsumer builds a Redis-backed order consumer over an existing client.
func NewOrderConsumer(client *redis.Client) *OrderConsumer {
	return &OrderConsumer{client: client, queue: OrderQueueName}
}

// Dequeue blocks until a message is available or ctx is cancelled. On
// cancellation it returns ctx.Err(), which stops the worker loop cleanly.
func (c *OrderConsumer) Dequeue(ctx context.Context) (order.OrderCreatedMessage, error) {
	res, err := c.client.BRPop(ctx, 0, c.queue).Result()
	if err != nil {
		return order.OrderCreatedMessage{}, err
	}
	// res is [queueName, payload].
	var msg order.OrderCreatedMessage
	if err := json.Unmarshal([]byte(res[1]), &msg); err != nil {
		return order.OrderCreatedMessage{}, err
	}
	return msg, nil
}
