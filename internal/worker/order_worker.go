// Package worker runs a bounded pool of goroutines that consume order queue messages.
package worker

import (
	"context"
	"log/slog"
	"sync"

	"order-service-go/internal/order"
)

// OrderWorker supervises N context-aware workers consuming from OrderQueue.
type OrderWorker struct {
	queue      order.OrderQueue
	retryQueue order.RetryQueue
	service    *order.Service
	processor  order.PaymentProcessor
	maxRetries int
	workers    int
	logger     *slog.Logger
}

// New builds an order worker pool. workers must be >= 1; maxRetries is the
// maximum number of re-tries after the first processing attempt (0 dead-letters
// on the first transient failure).
func New(
	queue order.OrderQueue,
	retryQueue order.RetryQueue,
	service *order.Service,
	processor order.PaymentProcessor,
	maxRetries int,
	workers int,
	logger *slog.Logger,
) *OrderWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &OrderWorker{
		queue:      queue,
		retryQueue: retryQueue,
		service:    service,
		processor:  processor,
		maxRetries: maxRetries,
		workers:    workers,
		logger:     logger,
	}
}

// Run starts the worker pool and blocks until ctx is cancelled and all workers exit.
func (w *OrderWorker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.runLoop(ctx, workerID)
		}(i)
	}
	wg.Wait()
}

func (w *OrderWorker) runLoop(ctx context.Context, workerID int) {
	for {
		msg, err := w.queue.Dequeue(ctx)
		if err != nil {
			return
		}
		if err := w.service.Process(context.Background(), msg.OrderID, w.processor); err != nil {
			w.handleProcessError(workerID, msg, err)
		}
	}
}

func (w *OrderWorker) handleProcessError(workerID int, msg order.OrderCreatedMessage, processErr error) {
	w.logger.Error("worker failed to process message",
		"worker_id", workerID,
		"order_id", msg.OrderID.String(),
		"retry_count", msg.RetryCount,
		"error", processErr.Error(),
	)

	if msg.RetryCount < w.maxRetries {
		retryMsg := msg
		retryMsg.RetryCount++
		if err := w.retryQueue.Requeue(context.Background(), retryMsg); err != nil {
			w.logger.Error("failed to requeue message after transient failure",
				"worker_id", workerID,
				"order_id", msg.OrderID.String(),
				"retry_count", retryMsg.RetryCount,
				"error", err.Error(),
			)
			return
		}
		w.logger.Warn("requeued message after transient failure",
			"worker_id", workerID,
			"order_id", msg.OrderID.String(),
			"retry_count", retryMsg.RetryCount,
		)
		return
	}

	if err := w.retryQueue.DeadLetter(context.Background(), msg); err != nil {
		w.logger.Error("failed to move message to dead-letter queue",
			"worker_id", workerID,
			"order_id", msg.OrderID.String(),
			"retry_count", msg.RetryCount,
			"error", err.Error(),
		)
		return
	}
	w.logger.Error("message moved to dead-letter queue after exhausting retries",
		"worker_id", workerID,
		"order_id", msg.OrderID.String(),
		"retry_count", msg.RetryCount,
		"max_retries", w.maxRetries,
	)
}
