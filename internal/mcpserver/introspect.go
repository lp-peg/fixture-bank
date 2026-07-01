package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lp-peg/fixture-bank/internal/ferr"
	"github.com/lp-peg/fixture-bank/internal/pgdb"
)

type introspectSchemaIn struct {
	Tables []string `json:"tables,omitempty" jsonschema:"対象テーブル名でのフィルタ。省略時はpublicスキーマの全テーブル"`
}

type introspectSchemaOut struct {
	Tables []pgdb.TableSchema `json:"tables"`
}

func registerIntrospectSchema(server *mcp.Server, deps Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "introspect_schema",
		Description: "対象PostgreSQLデータベースのテーブル・カラム・制約(PRIMARY KEY/UNIQUE/FOREIGN KEY)を調査する",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in introspectSchemaIn) (*mcp.CallToolResult, introspectSchemaOut, error) {
		if deps.DB == nil {
			return nil, introspectSchemaOut{}, ferr.New(ferr.TypeDBError, "introspect_schema requires the server to be started with --db-url")
		}
		tables, err := deps.DB.IntrospectSchema(ctx, in.Tables)
		if err != nil {
			return nil, introspectSchemaOut{}, ferr.New(ferr.TypeDBError, "introspect_schema failed: %v", err)
		}
		return nil, introspectSchemaOut{Tables: tables}, nil
	})
}
