// Package worker runs a bounded pool of goroutines that consume order queue messages.
package worker

import (
	"context"
	"log/slog"
	"sync"

	"order-service-worker/internal/order"
)

// OrderWorker supervises N context-aware workers consuming from OrderQueue.
type OrderWorker struct {
	queue     order.OrderQueue
	service   *order.Service
	processor order.PaymentProcessor
	workers   int
	logger    *slog.Logger
}

// New builds an order worker pool. workers must be >= 1.
func New(
	queue order.OrderQueue,
	service *order.Service,
	processor order.PaymentProcessor,
	workers int,
	logger *slog.Logger,
) *OrderWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &OrderWorker{
		queue:     queue,
		service:   service,
		processor: processor,
		workers:   workers,
		logger:    logger,
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
			w.logger.Error("worker failed to process message",
				"worker_id", workerID,
				"order_id", msg.OrderID.String(),
				"error", err.Error(),
			)
		}
	}
}
