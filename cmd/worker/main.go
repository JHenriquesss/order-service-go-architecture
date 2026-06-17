// Command worker consumes order-created messages from Redis and drives each
// order through processing. It shares the order domain with the API; the two
// run as separate processes communicating via Redis and (in production) a
// shared PostgreSQL database.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"

	"order-service-go/internal/config"
	"order-service-go/internal/database"
	"order-service-go/internal/logger"
	"order-service-go/internal/metrics"
	"order-service-go/internal/order"
	"order-service-go/internal/queue"
	"order-service-go/internal/worker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, os.Stdout)

	pool, err := database.NewPostgresPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	defer func() { _ = redisClient.Close() }()

	svc := order.NewService(order.NewPostgresRepository(pool), nil, nil, nil, nil, metrics.NewCollector(), log)
	consumer := queue.NewOrderConsumer(redisClient)
	workerPool := worker.New(consumer, svc, simulatedProcessor{}, cfg.OrderWorkerCount, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("worker pool starting", "workers", cfg.OrderWorkerCount)
	workerPool.Run(ctx)
	log.Info("worker pool stopped")
	return nil
}

// simulatedProcessor approves every order (architecture §9: simulate payment).
type simulatedProcessor struct{}

func (simulatedProcessor) ProcessPayment(context.Context, *order.Order) error {
	return nil
}
