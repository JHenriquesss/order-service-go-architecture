//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const orderQueueName = "orders:processing"

func TestPostgresCustomerRoundTrip(t *testing.T) {
	dsn := requirePostgresEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool := connectPostgres(t, ctx, dsn)
	runMigrations(t, ctx, pool)

	id := uuid.New()
	doc := "inttest-" + uniqueSuffix()
	name := "Integration Customer"
	email := doc + "@example.com"
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, document, email, phone, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, $6, $7)`,
		id, name, doc, email, "+1", now, now,
	)
	if err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var gotName, gotDoc string
	err = pool.QueryRow(ctx, `SELECT name, document FROM customers WHERE id=$1`, id).Scan(&gotName, &gotDoc)
	if err != nil {
		t.Fatalf("select customer: %v", err)
	}
	if gotName != name || gotDoc != doc {
		t.Fatalf("round-trip mismatch: got name=%q doc=%q", gotName, gotDoc)
	}
}

func runMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	dir := migrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var ups []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			ups = append(ups, e.Name())
		}
	}
	sort.Strings(ups)
	for _, name := range ups {
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
}

func migrationsDir() string {
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
		return dir
	}
	return filepath.Join("migrations")
}
