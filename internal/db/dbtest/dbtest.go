package dbtest

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testDatabaseURLEnv = "TEST_DATABASE_URL"

// NewPool connects to the integration test database.
func NewPool(t testing.TB) *pgxpool.Pool {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv(testDatabaseURLEnv))
	if dsn == "" {
		t.Skip(testDatabaseURLEnv + " is required for integration tests")
	}
	requireTestDatabase(t, dsn)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	return pool
}

func requireTestDatabase(t testing.TB, rawURL string) {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %s: %v", testDatabaseURLEnv, err)
	}

	database := strings.Trim(parsed.Path, "/")
	if !strings.Contains(strings.ToLower(database), "test") {
		t.Fatalf("%s database name must contain %q, got %q", testDatabaseURLEnv, "test", database)
	}
}

// ApplyMigrations executes all up migrations against the test database.
func ApplyMigrations(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()

	dir := findMigrationDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	sort.Strings(files)

	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(file), err)
		}
	}
}

// TruncateTables removes data from test tables while keeping schema intact.
func TruncateTables(t testing.TB, pool *pgxpool.Pool, tables ...string) {
	t.Helper()

	if len(tables) == 0 {
		return
	}
	statement := "TRUNCATE TABLE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE"
	if _, err := pool.Exec(context.Background(), statement); err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}
}

func findMigrationDir(t testing.TB) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for {
		candidate := filepath.Join(dir, "sql", "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("sql/migrations directory not found")
		}
		dir = parent
	}
}
