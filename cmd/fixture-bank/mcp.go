package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/lp-peg/fixture-bank/internal/fixturestore"
	"github.com/lp-peg/fixture-bank/internal/mcpserver"
	"github.com/lp-peg/fixture-bank/internal/pgdb"
)

func newMCPCmd() *cobra.Command {
	var (
		dbURL    string
		storeDir string
	)

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCPサーバーとして起動する(stdio transport)。docs/MCP_TOOLS.mdを参照",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			deps := mcpserver.Deps{Store: fixturestore.New(storeDir)}
			if dbURL != "" {
				db, err := pgdb.Connect(ctx, dbURL)
				if err != nil {
					return err
				}
				defer db.Close()
				deps.DB = db
			}

			return mcpserver.New(deps).Run(ctx, &mcp.StdioTransport{})
		},
	}

	cmd.Flags().StringVar(&dbURL, "db-url", "", "introspect_schema / draft_dsl / pool_ref / unique:db に使うPostgreSQL接続文字列")
	cmd.Flags().StringVar(&storeDir, "store-dir", "./fixtures", "save_fixture の保存先ディレクトリ")
	return cmd
}
