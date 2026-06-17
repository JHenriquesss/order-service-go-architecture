// Command seed applies the default admin user after migrations (architecture §34).
package main

import (
	"context"
	"fmt"
	"os"

	"order-service-go/internal/auth"
	"order-service-go/internal/database"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := auth.NewPostgresUserRepository(pool)
	if err := auth.SeedDefaultAdmin(ctx, repo, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
