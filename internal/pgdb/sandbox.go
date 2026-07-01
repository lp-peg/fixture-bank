package pgdb

import (
	"context"
	"strings"
)

// TrySandbox executes each statement inside a transaction that is always
// rolled back, so DSL_SPEC.md ¤5 step 3 ("DB実行検証") can surface real
// constraint violations (unique, FK, CHECK) against the live schema
// without ever persisting the trial rows.
func (db *DB) TrySandbox(ctx context.Context, sqlText string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, stmt := range strings.Split(sqlText, "\n") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
