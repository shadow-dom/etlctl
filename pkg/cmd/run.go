package cmd

import (
	"fmt"

	"shadow-dom/etlctl/pkg/util/etl"
	_ "shadow-dom/etlctl/pkg/util/etl/sources"
	_ "shadow-dom/etlctl/pkg/util/etl/targets"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Execute an ETL pipeline",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("Running ETL: %s\n", name)
		return etl.Run(configDir, name)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
