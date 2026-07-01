package pgdb

import "context"

// ColumnSchema describes one column, as introspected from
// information_schema. It backs the MCP `introspect_schema` tool
// (docs/MCP_TOOLS.md), which draft_dsl's schema-integrity check
// (DSL_SPEC.md ¤5 step 2) is validated against.
type ColumnSchema struct {
	Name       string         `json:"name"`
	DataType   string         `json:"data_type"`
	Nullable   bool           `json:"nullable"`
	PrimaryKey bool           `json:"primary_key"`
	Unique     bool           `json:"unique"`
	ForeignKey *ForeignKeyRef `json:"foreign_key,omitempty"`
}

// ForeignKeyRef is the (table, column) a foreign key column points to.
type ForeignKeyRef struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

// TableSchema is one table and its columns.
type TableSchema struct {
	Name    string         `json:"name"`
	Columns []ColumnSchema `json:"columns"`
}

const columnsQuery = `
SELECT table_name, column_name, data_type, (is_nullable = 'YES') AS nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND ($1::text[] IS NULL OR table_name = ANY($1))
ORDER BY table_name, ordinal_position
`

const constraintsQuery = `
SELECT tc.table_name, tc.constraint_type, kcu.column_name,
       ccu.table_name AS foreign_table_name, ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
LEFT JOIN information_schema.constraint_column_usage ccu
  ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema
  AND tc.constraint_type = 'FOREIGN KEY'
WHERE tc.table_schema = 'public'
  AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE', 'FOREIGN KEY')
  AND ($1::text[] IS NULL OR tc.table_name = ANY($1))
`

// IntrospectSchema returns every table (optionally filtered to tables) in
// the "public" schema, per DESIGN.md's `introspect_schema` MCP tool.
// PostgreSQL is the only backend targeted in v1 (DESIGN.md ¤5).
func (db *DB) IntrospectSchema(ctx context.Context, tables []string) ([]TableSchema, error) {
	var tableFilter []string
	if len(tables) > 0 {
		tableFilter = tables
	}

	order := make([]string, 0)
	byTable := make(map[string]*TableSchema)

	rows, err := db.pool.Query(ctx, columnsQuery, tableFilter)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tableName, columnName, dataType string
		var nullable bool
		if err := rows.Scan(&tableName, &columnName, &dataType, &nullable); err != nil {
			rows.Close()
			return nil, err
		}
		t, ok := byTable[tableName]
		if !ok {
			t = &TableSchema{Name: tableName}
			byTable[tableName] = t
			order = append(order, tableName)
		}
		t.Columns = append(t.Columns, ColumnSchema{Name: columnName, DataType: dataType, Nullable: nullable})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Built only after every table's Columns slice is done growing: taking
	// a column's address while its slice might still reallocate (via
	// append above) would leave byColumn holding dangling pointers.
	byColumn := make(map[string]map[string]*ColumnSchema, len(byTable))
	for tableName, t := range byTable {
		cols := make(map[string]*ColumnSchema, len(t.Columns))
		for i := range t.Columns {
			cols[t.Columns[i].Name] = &t.Columns[i]
		}
		byColumn[tableName] = cols
	}

	rows, err = db.pool.Query(ctx, constraintsQuery, tableFilter)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tableName, constraintType, columnName string
		var fkTable, fkColumn *string
		if err := rows.Scan(&tableName, &constraintType, &columnName, &fkTable, &fkColumn); err != nil {
			rows.Close()
			return nil, err
		}
		col, ok := byColumn[tableName][columnName]
		if !ok {
			continue
		}
		switch constraintType {
		case "PRIMARY KEY":
			col.PrimaryKey = true
		case "UNIQUE":
			col.Unique = true
		case "FOREIGN KEY":
			if fkTable != nil && fkColumn != nil {
				col.ForeignKey = &ForeignKeyRef{Table: *fkTable, Column: *fkColumn}
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]TableSchema, 0, len(order))
	for _, name := range order {
		out = append(out, *byTable[name])
	}
	return out, nil
}
