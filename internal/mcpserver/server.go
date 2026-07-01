// Package mcpserver exposes fixture-bank as an MCP server, implementing
// the introspect_schema / draft_dsl / materialize / save_fixture tools
// described in docs/DESIGN.md ¤5 and docs/MCP_TOOLS.md.
package mcpserver

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lp-peg/fixture-bank/internal/fixturestore"
	"github.com/lp-peg/fixture-bank/internal/pgdb"
)

// Deps are the resources MCP tool handlers need. DB may be nil if the
// server was started without --db-url; tools that need a database
// (introspect_schema, draft_dsl's schema/db_execution stages, and
// pool_ref/unique:db during materialize) report a db_error in that case
// instead of panicking.
type Deps struct {
	DB    *pgdb.DB
	Store *fixturestore.Store
}

// New builds an MCP server exposing DESIGN.md's four MCP tools.
func New(deps Deps) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture-bank", Version: "0.1.0"}, nil)
	registerIntrospectSchema(server, deps)
	registerDraftDSL(server, deps)
	registerMaterialize(server, deps)
	registerSaveFixture(server, deps)
	return server
}
