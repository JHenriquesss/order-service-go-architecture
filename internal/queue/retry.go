package queue

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"

	"order-service-go/internal/order"
)

// OrderDeadLetterQueueName is the Redis list for messages that exhausted retries.
const OrderDeadLetterQueueName = "orders:dead-letter"

// OrderRetryQueue re-publishes failed messages to the processing queue or moves
// them to the dead-letter queue.
type OrderRetryQueue struct {
	client *redis.Client
}

// NewOrderRetryQueue builds a Redis-backed retry/dead-letter publisher.
func NewOrderRetryQueue(client *redis.Client) *OrderRetryQueue {
	return &OrderRetryQueue{client: client}
}

// Requeue LPUSHes the message onto the processing queue with an incremented retry count.
func (q *OrderRetryQueue) Requeue(ctx context.Context, msg order.OrderCreatedMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, OrderQueueName, payload).Err()
}

// DeadLetter LPUSHes the message onto the dead-letter queue.
func (q *OrderRetryQueue) DeadLetter(ctx context.Context, msg order.OrderCreatedMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, OrderDeadLetterQueueName, payload).Err()
}
