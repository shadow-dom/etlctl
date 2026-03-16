package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var configDir string

var rootCmd = &cobra.Command{
	Use:   "etlctl",
	Short: "Schema-driven ETL pipeline management tool",
	Long: `etlctl manages ETL pipelines: define sources, targets, transforms,
and pipelines in YAML, then generate, validate, run, and deploy them.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "./etls", "Directory containing ETL YAML definitions")
}
