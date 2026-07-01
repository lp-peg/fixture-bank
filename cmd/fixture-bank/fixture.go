package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/fixturestore"
)

func newFixtureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fixture",
		Short: "保存済みFixtureの管理（保存・一覧）",
	}
	cmd.AddCommand(newFixtureSaveCmd())
	cmd.AddCommand(newFixtureListCmd())
	return cmd
}

func newFixtureSaveCmd() *cobra.Command {
	var (
		dslPath  string
		tag      string
		storeDir string
	)

	cmd := &cobra.Command{
		Use:   "save",
		Short: "DSLファイルをタグ付きでFixtureとして保存する",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dslPath == "" || tag == "" {
				return fmt.Errorf("--dsl と --tag は両方必須です")
			}
			data, err := os.ReadFile(dslPath)
			if err != nil {
				return err
			}
			// Syntax-validate before saving so broken fixtures never make
			// it into the store.
			if _, err := dsl.Parse(data); err != nil {
				return err
			}
			if err := fixturestore.New(storeDir).Save(tag, data); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved: %s\n", tag)
			return nil
		},
	}

	cmd.Flags().StringVar(&dslPath, "dsl", "", "保存するDSL YAMLファイルのパス")
	cmd.Flags().StringVar(&tag, "tag", "", "保存タグ（例: user:level50:has_premium_pass）")
	cmd.Flags().StringVar(&storeDir, "store-dir", "./fixtures", "Fixture保存先ディレクトリ")
	return cmd
}

func newFixtureListCmd() *cobra.Command {
	var storeDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "保存済みFixtureのタグ一覧を表示する",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := fixturestore.New(storeDir).List()
			if err != nil {
				return err
			}
			for _, t := range tags {
				fmt.Fprintln(cmd.OutOrStdout(), t)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&storeDir, "store-dir", "./fixtures", "Fixture保存先ディレクトリ")
	return cmd
}
