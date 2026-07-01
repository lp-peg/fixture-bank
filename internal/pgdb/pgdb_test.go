package pgdb_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lp-peg/fixture-bank/internal/pgdb"
)

// These tests hit a real PostgreSQL instance and are skipped unless
// FIXTURE_BANK_TEST_DATABASE_URL is set (e.g. in CI or local dev with
// `service postgresql start`), per DESIGN.md ¤5 "PostgreSQLのみ" scope.
func testDBURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("FIXTURE_BANK_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("FIXTURE_BANK_TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	return url
}

func TestDB_QueryAndExists(t *testing.T) {
	url := testDBURL(t)
	ctx := context.Background()

	raw, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer raw.Close()

	if _, err := raw.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS pgdb_test_products (id uuid PRIMARY KEY, status text);
		DELETE FROM pgdb_test_products;
		INSERT INTO pgdb_test_products (id, status) VALUES
			(gen_random_uuid(), 'active'),
			(gen_random_uuid(), 'active'),
			(gen_random_uuid(), 'inactive');
	`); err != nil {
		t.Fatalf("setup exec error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = raw.Exec(context.Background(), `DROP TABLE IF EXISTS pgdb_test_products`)
	})

	db, err := pgdb.Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer db.Close()

	pool, err := db.Query(`SELECT id FROM pgdb_test_products WHERE status = 'active'`)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(pool) != 2 {
		t.Fatalf("len(pool) = %d, want 2", len(pool))
	}
	for _, v := range pool {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("pool value %v is %T, want string (uuid normalized)", v, v)
		}
		if len(s) != 36 {
			t.Errorf("pool value %q doesn't look like a UUID string", s)
		}
	}

	exists, err := db.Exists("pgdb_test_products", "id", pool[0])
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Errorf("Exists(%v) = false, want true", pool[0])
	}

	exists, err = db.Exists("pgdb_test_products", "id", "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Errorf("Exists(zero-uuid) = true, want false")
	}
}

func TestDB_Query_EmptyResult(t *testing.T) {
	url := testDBURL(t)
	ctx := context.Background()

	db, err := pgdb.Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer db.Close()

	pool, err := db.Query(`SELECT 1 WHERE false`)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(pool) != 0 {
		t.Fatalf("len(pool) = %d, want 0", len(pool))
	}
}
