package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"order-service-worker/internal/config"
	"order-service-worker/internal/metrics"
	"order-service-worker/internal/order"
	"order-service-worker/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		logger.Error("config load failed", "error", err.Error())
		os.Exit(1)
	}

	repo := order.NewInMemoryRepository()
	svc := order.NewService(repo, nil, nil, nil, nil, metrics.NewCollector(), logger)
	proc := order.FakePaymentProcessor{}
	queue := order.NewFakeQueue()

	pool := worker.New(queue, svc, &proc, cfg.OrderWorkerCount, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("worker pool starting", "workers", cfg.OrderWorkerCount)
	pool.Run(ctx)
	logger.Info("worker pool stopped")
}
