package mcpserver_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lp-peg/fixture-bank/internal/fixturestore"
	"github.com/lp-peg/fixture-bank/internal/mcpserver"
	"github.com/lp-peg/fixture-bank/internal/pgdb"
)

// connect wires an in-process MCP client to a fresh server built from deps,
// with no subprocess or network transport needed.
func connect(t *testing.T, deps mcpserver.Deps) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := mcpserver.New(deps)
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) transport error = %v", name, err)
	}
	return res
}

func structured(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	m, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent is %T, want map[string]any (content: %+v)", res.StructuredContent, res.Content)
	}
	return m
}

const simpleDSL = `
entity: user
count: 2
seed: 1
fields:
  id: {type: uuid, generator: uuid_v4}
  level: {type: int, generator: fixed, value: 50}
`

func TestMaterialize_JSON(t *testing.T) {
	cs := connect(t, mcpserver.Deps{Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "materialize", map[string]any{"dsl": simpleDSL, "format": "json"})
	if res.IsError {
		t.Fatalf("materialize returned IsError: %+v", res.Content)
	}
	out := structured(t, res)
	rendered, _ := out["output"].(string)
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, rendered)
	}
	if len(decoded) != 2 {
		t.Fatalf("len(decoded) = %d, want 2", len(decoded))
	}
}

func TestMaterialize_CountOverride(t *testing.T) {
	cs := connect(t, mcpserver.Deps{Store: fixturestore.New(t.TempDir())})
	count := 5
	res := callTool(t, cs, "materialize", map[string]any{"dsl": simpleDSL, "format": "json", "count": count})
	if res.IsError {
		t.Fatalf("materialize returned IsError: %+v", res.Content)
	}
	out := structured(t, res)
	rendered, _ := out["output"].(string)
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 5 {
		t.Fatalf("len(decoded) = %d, want 5 (count override)", len(decoded))
	}
}

func TestMaterialize_InvalidFormat(t *testing.T) {
	cs := connect(t, mcpserver.Deps{Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "materialize", map[string]any{"dsl": simpleDSL, "format": "xml"})
	if !res.IsError {
		t.Fatalf("expected IsError for unsupported format, got: %+v", res.StructuredContent)
	}
}

func TestMaterialize_SyntaxError(t *testing.T) {
	cs := connect(t, mcpserver.Deps{Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "materialize", map[string]any{"dsl": "entity: user\nfields: {}\n", "format": "json"})
	if !res.IsError {
		t.Fatalf("expected IsError for DSL with no fields, got: %+v", res.StructuredContent)
	}
}

func TestSaveFixture_WritesToStore(t *testing.T) {
	dir := t.TempDir()
	store := fixturestore.New(dir)
	cs := connect(t, mcpserver.Deps{Store: store})

	res := callTool(t, cs, "save_fixture", map[string]any{"dsl": simpleDSL, "tag": "user:level50"})
	if res.IsError {
		t.Fatalf("save_fixture returned IsError: %+v", res.Content)
	}
	out := structured(t, res)
	if out["tag"] != "user:level50" {
		t.Errorf("tag = %v, want user:level50", out["tag"])
	}

	fx, err := store.Load("user:level50")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if fx.Entity != "user" {
		t.Errorf("Entity = %q, want user", fx.Entity)
	}
}

func TestIntrospectSchema_NoDBConfigured(t *testing.T) {
	cs := connect(t, mcpserver.Deps{Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "introspect_schema", map[string]any{})
	if !res.IsError {
		t.Fatalf("expected IsError when no --db-url configured, got: %+v", res.StructuredContent)
	}
}

func TestDraftDSL_SyntaxStageFailure(t *testing.T) {
	cs := connect(t, mcpserver.Deps{Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "draft_dsl", map[string]any{"dsl": "entity: user\nfields: {}\n"})
	if res.IsError {
		t.Fatalf("draft_dsl should report failures via Valid:false, not IsError: %+v", res.Content)
	}
	out := structured(t, res)
	if out["valid"] != false {
		t.Fatalf("valid = %v, want false", out["valid"])
	}
	if out["stage"] != "syntax" {
		t.Errorf("stage = %v, want syntax", out["stage"])
	}
}

func TestDraftDSL_NoDBConfigured(t *testing.T) {
	cs := connect(t, mcpserver.Deps{Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "draft_dsl", map[string]any{"dsl": simpleDSL})
	if res.IsError {
		t.Fatalf("draft_dsl should report failures via Valid:false, not IsError: %+v", res.Content)
	}
	out := structured(t, res)
	if out["valid"] != false {
		t.Fatalf("valid = %v, want false", out["valid"])
	}
	if out["stage"] != "schema" {
		t.Errorf("stage = %v, want schema", out["stage"])
	}
	if out["error_type"] != "db_error" {
		t.Errorf("error_type = %v, want db_error", out["error_type"])
	}
}

// --- PostgreSQL-backed tests (skipped unless FIXTURE_BANK_TEST_DATABASE_URL is set) ---

func testDBURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("FIXTURE_BANK_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("FIXTURE_BANK_TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	return url
}

func setupSchema(t *testing.T, url string) {
	t.Helper()
	ctx := context.Background()
	raw, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer raw.Close()

	if _, err := raw.Exec(ctx, `
		DROP TABLE IF EXISTS mcp_test_order_item;
		DROP TABLE IF EXISTS mcp_test_order;
		DROP TABLE IF EXISTS mcp_test_user;
		CREATE TABLE mcp_test_user (id uuid PRIMARY KEY, email text UNIQUE);
		CREATE TABLE mcp_test_order (id uuid PRIMARY KEY, user_id uuid REFERENCES mcp_test_user(id));
	`); err != nil {
		t.Fatalf("setup exec error = %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		raw, err := pgxpool.New(ctx, url)
		if err == nil {
			defer raw.Close()
			_, _ = raw.Exec(ctx, `DROP TABLE IF EXISTS mcp_test_order; DROP TABLE IF EXISTS mcp_test_user;`)
		}
	})
}

func TestIntrospectSchema_WithDB(t *testing.T) {
	url := testDBURL(t)
	setupSchema(t, url)
	ctx := context.Background()
	db, err := pgdb.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cs := connect(t, mcpserver.Deps{DB: db, Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "introspect_schema", map[string]any{"tables": []string{"mcp_test_user"}})
	if res.IsError {
		t.Fatalf("introspect_schema returned IsError: %+v", res.Content)
	}
	out := structured(t, res)
	tables, _ := out["tables"].([]any)
	if len(tables) != 1 {
		t.Fatalf("len(tables) = %d, want 1", len(tables))
	}
	table := tables[0].(map[string]any)
	if table["name"] != "mcp_test_user" {
		t.Errorf("table name = %v, want mcp_test_user", table["name"])
	}
	cols, _ := table["columns"].([]any)
	foundPK := false
	for _, c := range cols {
		col := c.(map[string]any)
		if col["name"] == "id" && col["primary_key"] == true {
			foundPK = true
		}
	}
	if !foundPK {
		t.Errorf("expected id column to be reported as primary_key, got columns: %+v", cols)
	}
}

func TestDraftDSL_ValidThroughAllStages(t *testing.T) {
	url := testDBURL(t)
	setupSchema(t, url)
	ctx := context.Background()
	db, err := pgdb.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cs := connect(t, mcpserver.Deps{DB: db, Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "draft_dsl", map[string]any{"dsl": `
entity: mcp_test_user
count: 1
fields:
  id: {type: uuid, generator: uuid_v4}
  email: {type: string, generator: faker, provider: email}
`})
	if res.IsError {
		t.Fatalf("draft_dsl returned IsError: %+v", res.Content)
	}
	out := structured(t, res)
	if out["valid"] != true {
		t.Fatalf("valid = %v, want true; got %+v", out["valid"], out)
	}
}

func TestDraftDSL_SchemaStageFailure(t *testing.T) {
	url := testDBURL(t)
	setupSchema(t, url)
	ctx := context.Background()
	db, err := pgdb.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cs := connect(t, mcpserver.Deps{DB: db, Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "draft_dsl", map[string]any{"dsl": `
entity: mcp_test_user
count: 1
fields:
  id: {type: uuid, generator: uuid_v4}
  no_such_column: {type: string, generator: fixed, value: x}
`})
	if res.IsError {
		t.Fatalf("draft_dsl returned IsError: %+v", res.Content)
	}
	out := structured(t, res)
	if out["valid"] != false || out["stage"] != "schema" {
		t.Fatalf("expected valid=false stage=schema, got %+v", out)
	}
	if !strings.Contains(out["message"].(string), "no_such_column") {
		t.Errorf("message = %v, want it to mention no_such_column", out["message"])
	}
}

func TestDraftDSL_DBExecutionStageFailure(t *testing.T) {
	url := testDBURL(t)
	setupSchema(t, url)
	ctx := context.Background()
	db, err := pgdb.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// order.user_id has no matching user row -> FK violation during the
	// stage-3 sandbox insert.
	cs := connect(t, mcpserver.Deps{DB: db, Store: fixturestore.New(t.TempDir())})
	res := callTool(t, cs, "draft_dsl", map[string]any{"dsl": `
entity: mcp_test_order
count: 1
fields:
  id: {type: uuid, generator: uuid_v4}
  user_id: {type: uuid, generator: uuid_v4}
`})
	if res.IsError {
		t.Fatalf("draft_dsl returned IsError: %+v", res.Content)
	}
	out := structured(t, res)
	if out["valid"] != false || out["stage"] != "db_execution" {
		t.Fatalf("expected valid=false stage=db_execution, got %+v", out)
	}
}
