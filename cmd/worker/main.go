// Command worker is the background order processor. This phase ships a
// compiling stub that loads config, logs, and blocks until shutdown; the real
// Redis consume/process loop arrives in Phase 6.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"order-service-go/internal/config"
	"order-service-go/internal/logger"
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
	log.Info("worker started", "worker_count", cfg.OrderWorkerCount, "app_env", cfg.AppEnv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Info("worker shutting down")
	return nil
}
