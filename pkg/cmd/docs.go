package cmd

import (
	"fmt"

	"shadow-dom/etlctl/pkg/docs"
	"shadow-dom/etlctl/pkg/util/etl"

	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs [name]",
	Short: "Generate documentation for an ETL definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		e, err := etl.CreateETL(configDir, name)
		if err != nil {
			return fmt.Errorf("failed to load ETL config: %w", err)
		}
		docs.GenerateETLDocs(*e)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
