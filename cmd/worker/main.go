// Command worker consumes order-created messages from Redis and drives each
// order through processing. It shares the order domain with the API; the two
// run as separate processes communicating via Redis and (in production) a
// shared PostgreSQL database.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"order-service-go/internal/config"
	"order-service-go/internal/database"
	"order-service-go/internal/logger"
	"order-service-go/internal/metrics"
	"order-service-go/internal/money"
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

	metricsCollector := metrics.NewCollector()
	var repo order.OrderRepository = order.NewPostgresRepository(pool)
	if cfg.TransientFailureTotal != "" {
		sentinel, err := money.ParseString(cfg.TransientFailureTotal)
		if err != nil {
			return fmt.Errorf("ORDER_TRANSIENT_FAILURE_TOTAL: %w", err)
		}
		repo = newTransientFailureRepo(repo, sentinel)
	}
	svc := order.NewService(repo, nil, nil, nil, nil, metricsCollector, log)
	consumer := queue.NewOrderConsumer(redisClient)
	retryQueue := queue.NewOrderRetryQueue(redisClient)
	workerPool := worker.New(consumer, retryQueue, svc, simulatedProcessor{}, cfg.OrderMaxRetries, cfg.OrderWorkerCount, log)

	metricsMux := http.NewServeMux()
	metrics.NewHandler(metricsCollector).Register(metricsMux)
	metricsMux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	metricsAddr := ":" + cfg.MetricsPort
	metricsSrv := &http.Server{
		Addr:              metricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		log.Info("worker metrics server starting", "addr", metricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	poolDone := make(chan struct{})
	go func() {
		log.Info("worker pool starting", "workers", cfg.OrderWorkerCount)
		workerPool.Run(ctx)
		log.Info("worker pool stopped")
		close(poolDone)
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received, draining")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("metrics server shutdown: %w", err)
	}
	<-poolDone
	return nil
}

// simulatedProcessor approves every order except those with the integration
// failure sentinel total (architecture §9: simulate payment).
type simulatedProcessor struct{}

var failureTotal, _ = money.ParseString("13.37")

func (simulatedProcessor) ProcessPayment(_ context.Context, o *order.Order) error {
	if o.TotalAmount == failureTotal {
		return errors.New("simulated payment failure")
	}
	return nil
}
