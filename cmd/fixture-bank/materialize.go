package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
	"github.com/lp-peg/fixture-bank/internal/fixturestore"
	"github.com/lp-peg/fixture-bank/internal/generator"
	"github.com/lp-peg/fixture-bank/internal/materialize"
	"github.com/lp-peg/fixture-bank/internal/output"
	"github.com/lp-peg/fixture-bank/internal/pgdb"
)

func newMaterializeCmd() *cobra.Command {
	var (
		dslPath    string
		fixtureTag string
		count      int
		seed       int64
		format     string
		storeDir   string
		dbURL      string
		outPath    string
	)

	cmd := &cobra.Command{
		Use:   "materialize",
		Short: "DSLをもとに任意件数のデータをSQL/JSONとして生成する",
		RunE: func(cmd *cobra.Command, args []string) error {
			if (dslPath == "") == (fixtureTag == "") {
				return fmt.Errorf("exactly one of --dsl or --fixture must be given")
			}
			if format != "json" && format != "sql" {
				return ferr.New(ferr.TypeUnsupportedFormat, "--format must be \"json\" or \"sql\", got %q", format)
			}

			fx, err := loadFixture(dslPath, fixtureTag, storeDir)
			if err != nil {
				return err
			}

			opts := materialize.Options{}
			if cmd.Flags().Changed("count") {
				opts.Count = &count
			}
			if cmd.Flags().Changed("seed") {
				opts.Seed = &seed
			}

			if dbURL != "" {
				db, err := pgdb.Connect(context.Background(), dbURL)
				if err != nil {
					return err
				}
				defer db.Close()
				opts.DB = db
			}

			records, err := materialize.Run(fx, opts)
			if err != nil {
				return err
			}

			var out []byte
			switch format {
			case "json":
				out, err = output.RenderJSON(records)
				if err != nil {
					return err
				}
				out = append(out, '\n')
			case "sql":
				sqlText, err := output.RenderSQL(records)
				if err != nil {
					return err
				}
				out = []byte(sqlText)
			}

			if outPath == "" || outPath == "-" {
				_, err = os.Stdout.Write(out)
				return err
			}
			return os.WriteFile(outPath, out, 0o644)
		},
	}

	cmd.Flags().StringVar(&dslPath, "dsl", "", "DSL YAMLファイルのパス（--fixtureと排他）")
	cmd.Flags().StringVar(&fixtureTag, "fixture", "", "保存済みFixtureのタグ（例: user:level50:has_premium_pass）")
	cmd.Flags().IntVar(&count, "count", 0, "ルートエンティティの生成件数を上書き")
	cmd.Flags().Int64Var(&seed, "seed", 0, "乱数seedを上書き")
	cmd.Flags().StringVar(&format, "format", "json", "出力フォーマット: json | sql")
	cmd.Flags().StringVar(&storeDir, "store-dir", "./fixtures", "--fixture解決に使うFixture保存先ディレクトリ")
	cmd.Flags().StringVar(&dbURL, "db-url", "", "pool_ref / unique:db に使うPostgreSQL接続文字列")
	cmd.Flags().StringVar(&outPath, "out", "-", "出力先ファイルパス（省略時は標準出力）")

	return cmd
}

func loadFixture(dslPath, fixtureTag, storeDir string) (*dsl.Fixture, error) {
	if dslPath != "" {
		data, err := os.ReadFile(dslPath)
		if err != nil {
			return nil, err
		}
		return dsl.Parse(data)
	}
	return fixturestore.New(storeDir).Load(fixtureTag)
}

// generator.DB is referenced only to keep the import used by pgdb.DB's
// implicit interface satisfaction visible to readers; pgdb.DB satisfies it
// structurally.
var _ generator.DB = (*pgdb.DB)(nil)
