// Package pgdb implements generator.DB against a real PostgreSQL database,
// backing pool_ref (DSL_SPEC.md ¤2.2) and unique: db (¤2.3). PostgreSQL is
// the only backend targeted in v1 (DESIGN.md ¤5 "やらないこと").
package pgdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// Connect opens a pool against url and verifies connectivity with a ping.
func Connect(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("pgdb: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgdb: ping: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Close releases all pooled connections.
func (db *DB) Close() {
	db.pool.Close()
}

// Query runs a read-only query and returns the first column of each row.
// Used once per pool_ref field to build its candidate pool.
func (db *DB) Query(query string) ([]any, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		if len(vals) == 0 {
			continue
		}
		out = append(out, normalize(vals[0]))
	}
	return out, rows.Err()
}

// normalize converts pgx's raw Go representations of PostgreSQL types into
// the plain scalars the rest of fixture-bank works with (JSON/SQL
// renderers, ref resolution). Without this, e.g. a uuid column comes back
// as [16]byte and gets serialized as a byte array instead of a string.
func normalize(v any) any {
	switch val := v.(type) {
	case [16]byte:
		return uuid.UUID(val).String()
	default:
		return v
	}
}

// Exists runs a lightweight SELECT existence check against entity's
// column, for unique: db (DSL_SPEC.md ¤2.3).
func (db *DB) Exists(entity, column string, value any) (bool, error) {
	ctx := context.Background()
	query := fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)`,
		quoteIdent(entity), quoteIdent(column),
	)
	var exists bool
	if err := db.pool.QueryRow(ctx, query, value).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// quoteIdent double-quotes a PostgreSQL identifier, doubling any embedded
// double quotes. entity/column names are DSL-authored (not raw end-user
// input), but this keeps them safe to interpolate regardless, since
// identifiers can't be passed as query parameters.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
