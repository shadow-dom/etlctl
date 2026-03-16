package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"shadow-dom/etlctl/pkg/util/etl"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all ETL definitions",
	RunE: func(cmd *cobra.Command, args []string) error {
		matches, err := filepath.Glob(filepath.Join(configDir, "*.yaml"))
		if err != nil {
			return fmt.Errorf("failed to scan config directory: %w", err)
		}

		if len(matches) == 0 {
			fmt.Println("No ETL definitions found in", configDir)
			return nil
		}

		fmt.Printf("%-30s %-10s %-10s %-10s\n", "NAME", "SOURCES", "TARGETS", "PIPELINES")
		fmt.Println(strings.Repeat("-", 65))

		for _, match := range matches {
			name := strings.TrimSuffix(filepath.Base(match), ".yaml")
			e, err := etl.CreateETL(configDir, name)
			if err != nil {
				fmt.Printf("%-30s (error: %v)\n", name, err)
				continue
			}

			fmt.Printf("%-30s %-10d %-10d %-10d\n",
				e.Name, len(e.Sources), len(e.Targets), len(e.Pipelines))
		}

		return nil
	},
}

func init() {
	listCmd.SilenceUsage = true
	rootCmd.AddCommand(listCmd)
}
