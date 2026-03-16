package cmd

import (
	"fmt"

	"shadow-dom/etlctl/pkg/util/etl"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [name]",
	Short: "Validate an ETL definition for correctness",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		e, err := etl.CreateETL(configDir, name)
		if err != nil {
			return fmt.Errorf("failed to load ETL config: %w", err)
		}

		errs := etl.ValidateETL(e)
		if len(errs) > 0 {
			fmt.Printf("Validation failed for %s:\n", name)
			for _, e := range errs {
				fmt.Printf("  - %s\n", e)
			}
			return fmt.Errorf("found %d validation error(s)", len(errs))
		}

		fmt.Printf("ETL %s is valid.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
