package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
	"github.com/lp-peg/fixture-bank/internal/materialize"
	"github.com/lp-peg/fixture-bank/internal/output"
)

type materializeIn struct {
	DSL    string `json:"dsl" jsonschema:"Fixture Bank DSL(YAML文字列)"`
	Count  *int   `json:"count,omitempty" jsonschema:"ルートエンティティの生成件数を上書き"`
	Seed   *int64 `json:"seed,omitempty" jsonschema:"乱数seedを上書き"`
	Format string `json:"format" jsonschema:"出力フォーマット: json または sql"`
}

type materializeOut struct {
	Output string `json:"output"`
}

func registerMaterialize(server *mcp.Server, deps Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "materialize",
		Description: "DSLをもとに任意件数のデータをJSON/SQLとして生成する",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in materializeIn) (*mcp.CallToolResult, materializeOut, error) {
		if in.Format != "json" && in.Format != "sql" {
			return nil, materializeOut{}, ferr.New(ferr.TypeUnsupportedFormat, "format must be \"json\" or \"sql\", got %q", in.Format)
		}
		fx, err := dsl.Parse([]byte(in.DSL))
		if err != nil {
			return nil, materializeOut{}, err
		}

		opts := materialize.Options{Count: in.Count, Seed: in.Seed}
		if deps.DB != nil {
			opts.DB = deps.DB
		}
		records, err := materialize.Run(fx, opts)
		if err != nil {
			return nil, materializeOut{}, err
		}

		var rendered string
		switch in.Format {
		case "json":
			data, err := output.RenderJSON(records)
			if err != nil {
				return nil, materializeOut{}, err
			}
			rendered = string(data)
		case "sql":
			rendered, err = output.RenderSQL(records)
			if err != nil {
				return nil, materializeOut{}, err
			}
		}
		return nil, materializeOut{Output: rendered}, nil
	})
}
