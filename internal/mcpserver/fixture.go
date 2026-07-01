package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lp-peg/fixture-bank/internal/dsl"
)

type saveFixtureIn struct {
	DSL string `json:"dsl" jsonschema:"保存するFixture Bank DSL(YAML文字列)"`
	Tag string `json:"tag" jsonschema:"保存タグ(例: user:level50:has_premium_pass)"`
}

type saveFixtureOut struct {
	Tag string `json:"tag"`
}

func registerSaveFixture(server *mcp.Server, deps Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "save_fixture",
		Description: "Fixture Bank DSLをタグ付きで保存する",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in saveFixtureIn) (*mcp.CallToolResult, saveFixtureOut, error) {
		if _, err := dsl.Parse([]byte(in.DSL)); err != nil {
			return nil, saveFixtureOut{}, err
		}
		if err := deps.Store.Save(in.Tag, []byte(in.DSL)); err != nil {
			return nil, saveFixtureOut{}, err
		}
		return nil, saveFixtureOut{Tag: in.Tag}, nil
	})
}
