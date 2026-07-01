package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lp-peg/fixture-bank/internal/materialize"
)

// RenderSQL renders records as PostgreSQL INSERT statements (DESIGN.md
// ¤3.3, DSL_SPEC.md ¤4). Each record becomes one INSERT into a table named
// after its entity; parents are always emitted before their children
// (depth-first), so INSERT order respects FK dependencies.
func RenderSQL(records []*materialize.Record) (string, error) {
	var sb strings.Builder
	for _, r := range records {
		if err := renderRecordSQL(&sb, r); err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

func renderRecordSQL(sb *strings.Builder, r *materialize.Record) error {
	cols := make([]string, len(r.FieldOrder))
	vals := make([]string, len(r.FieldOrder))
	for i, name := range r.FieldOrder {
		cols[i] = quoteIdent(name)
		v, err := sqlLiteral(r.Values[name])
		if err != nil {
			return fmt.Errorf("%s.%s: %w", r.Entity, name, err)
		}
		vals[i] = v
	}
	fmt.Fprintf(sb, "INSERT INTO %s (%s) VALUES (%s);\n",
		quoteIdent(r.Entity), strings.Join(cols, ", "), strings.Join(vals, ", "))

	for _, relName := range r.RelationOrder {
		for _, child := range r.Children[relName] {
			if err := renderRecordSQL(sb, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sqlLiteral(v any) (string, error) {
	switch val := v.(type) {
	case nil:
		return "NULL", nil
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'", nil
	case bool:
		if val {
			return "TRUE", nil
		}
		return "FALSE", nil
	case int:
		return strconv.Itoa(val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unsupported value type %T", v)
	}
}
