package cmd

import (
	"fmt"

	"shadow-dom/etlctl/pkg/builder"

	"github.com/spf13/cobra"
)

var outputDir string

var generateCmd = &cobra.Command{
	Use:   "generate [name]",
	Short: "Generate Go source code from an ETL definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("Generating ETL: %s\n", name)
		return builder.GenerateETL(configDir, name, outputDir)
	},
}

func init() {
	generateCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for generated Go code (default: <config-dir>/gen)")
	rootCmd.AddCommand(generateCmd)
}
