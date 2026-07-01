// Command fixture-bank generates load-test fixture data from the Fixture
// Bank DSL. See docs/DESIGN.md and docs/DSL_SPEC.md.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:           "fixture-bank",
		Short:         "負荷試験の前提データを、DSLを介して宣言的に量産するツール",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(newMaterializeCmd())
	root.AddCommand(newFixtureCmd())
	root.AddCommand(newMCPCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
