package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
	"github.com/lp-peg/fixture-bank/internal/materialize"
	"github.com/lp-peg/fixture-bank/internal/output"
	"github.com/lp-peg/fixture-bank/internal/pgdb"
)

type draftDSLIn struct {
	DSL string `json:"dsl" jsonschema:"検証対象のFixture Bank DSL(YAML文字列)"`
}

type draftDSLOut struct {
	Valid bool `json:"valid"`
	// Stage identifies which of DSL_SPEC.md ¤5's three steps rejected the
	// DSL: "syntax", "schema", or "db_execution". Empty when Valid is true.
	Stage     string `json:"stage,omitempty"`
	ErrorType string `json:"error_type,omitempty"`
	Message   string `json:"message,omitempty"`
}

// registerDraftDSL implements DSL_SPEC.md ¤5's three-stage validation:
// syntax → schema integrity → sandbox DB execution. A rejection at any
// stage is a normal (Valid: false) tool result, not a Go error — the
// caller is expected to iterate on the DSL based on the reported stage.
func registerDraftDSL(server *mcp.Server, deps Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "draft_dsl",
		Description: "Fixture Bank DSLをDSL_SPEC.md 5節の3段階(構文検証→スキーマ整合検証→DB実行検証)で検証する",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in draftDSLIn) (*mcp.CallToolResult, draftDSLOut, error) {
		fx, err := dsl.Parse([]byte(in.DSL))
		if err != nil {
			return nil, invalidResult("syntax", err), nil
		}

		if deps.DB == nil {
			return nil, invalidResult("schema", ferr.New(ferr.TypeDBError,
				"schema/db_execution validation requires the server to be started with --db-url")), nil
		}

		schema, err := deps.DB.IntrospectSchema(ctx, nil)
		if err != nil {
			return nil, invalidResult("schema", ferr.New(ferr.TypeDBError, "introspect_schema failed: %v", err)), nil
		}
		if err := validateAgainstSchema(fx, schema); err != nil {
			return nil, invalidResult("schema", err), nil
		}

		one := 1
		records, err := materialize.Run(fx, materialize.Options{Count: &one, DB: deps.DB})
		if err != nil {
			return nil, invalidResult("db_execution", err), nil
		}
		sqlText, err := output.RenderSQL(records)
		if err != nil {
			return nil, invalidResult("db_execution", err), nil
		}
		if err := deps.DB.TrySandbox(ctx, sqlText); err != nil {
			return nil, invalidResult("db_execution", ferr.New(ferr.TypeDBError, "sandbox execution failed: %v", err)), nil
		}

		return nil, draftDSLOut{Valid: true}, nil
	})
}

func invalidResult(stage string, err error) draftDSLOut {
	out := draftDSLOut{Valid: false, Stage: stage, Message: err.Error()}
	if fe, ok := err.(*ferr.Error); ok {
		out.ErrorType = fe.ErrorType
	}
	return out
}

// validateAgainstSchema implements DSL_SPEC.md ¤5 step 2: every entity
// (root + relations) must match a real table, and every field must match
// a real column of that table.
func validateAgainstSchema(fx *dsl.Fixture, schema []pgdb.TableSchema) error {
	tables := make(map[string]map[string]bool, len(schema))
	for _, t := range schema {
		cols := make(map[string]bool, len(t.Columns))
		for _, c := range t.Columns {
			cols[c.Name] = true
		}
		tables[t.Name] = cols
	}
	return checkEntitySchema(fx.Entity, fx.Fields, fx.Relations, tables)
}

func checkEntitySchema(entity string, fields dsl.Fields, relations dsl.Relations, tables map[string]map[string]bool) error {
	cols, ok := tables[entity]
	if !ok {
		return ferr.New(ferr.TypeSchemaMismatch, "entity %q does not match any table in the database", entity)
	}
	for _, fe := range fields {
		if !cols[fe.Name] {
			return ferr.New(ferr.TypeSchemaMismatch, "%s.%s: no such column in table %q", entity, fe.Name, entity)
		}
	}
	for _, re := range relations {
		if err := checkEntitySchema(re.Relation.Entity, re.Relation.Fields, re.Relation.Relations, tables); err != nil {
			return err
		}
	}
	return nil
}
