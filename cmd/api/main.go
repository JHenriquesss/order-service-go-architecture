// Command api composes the configuration, logger, database pool and HTTP
// router, then serves requests. All fallible setup returns an error to main;
// only main exits the process.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"order-service-go/internal/auth"
	"order-service-go/internal/config"
	"order-service-go/internal/customer"
	"order-service-go/internal/database"
	"order-service-go/internal/logger"
	"order-service-go/internal/metrics"
	"order-service-go/internal/order"
	"order-service-go/internal/product"
	"order-service-go/internal/queue"
	"order-service-go/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// seedAdmin creates the default admin user (architecture §34) if it does not
// already exist. It is idempotent: a present admin or a concurrent insert is a
// no-op.
func seedAdmin(ctx context.Context, repo *auth.PostgresUserRepository, log *slog.Logger) error {
	const adminEmail = "admin@example.com"
	if _, err := repo.FindByEmail(ctx, adminEmail); err == nil {
		return nil
	} else if !errors.Is(err, auth.ErrUserNotFound) {
		return err
	}
	hash, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}
	now := time.Now()
	user := &auth.User{
		ID:           uuid.NewString(),
		Name:         "Admin",
		Email:        adminEmail,
		PasswordHash: hash,
		Role:         auth.RoleAdmin,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Create(ctx, user); err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			return nil
		}
		return err
	}
	log.Info("seeded default admin user", "email", adminEmail)
	return nil
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

	tokens, err := auth.NewTokenManager(cfg.JWTSecret, time.Duration(cfg.JWTExpirationMinutes)*time.Minute)
	if err != nil {
		return err
	}
	userRepo := auth.NewPostgresUserRepository(pool)
	if err := seedAdmin(context.Background(), userRepo, log); err != nil {
		return err
	}
	authService := auth.NewService(userRepo, tokens)
	authHandler := auth.NewHandler(authService)

	customerSvc := customer.NewService(customer.NewPostgresRepository(pool))
	productSvc := product.NewService(product.NewPostgresRepository(pool))
	customerHandler := customer.NewHandler(customerSvc)
	productHandler := product.NewHandler(productSvc)

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	defer func() { _ = redisClient.Close() }()
	orderProducer := queue.NewOrderProducer(redisClient)
	metricsCollector := metrics.NewCollector()
	orderSvc := order.NewService(
		order.NewPostgresRepository(pool),
		customerLookup{svc: customerSvc},
		productLookup{svc: productSvc},
		orderProducer,
		nil,
		metricsCollector,
		log,
	)
	orderHandler := order.NewHandler(orderSvc)

	handler := server.New(log, authHandler, customerHandler, productHandler, orderHandler, metricsCollector, tokens)
	addr := ":" + cfg.HTTPPort
	log.Info("starting api server", "addr", addr, "app_env", cfg.AppEnv)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received, draining")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
