package cmd

import (
	"fmt"

	"shadow-dom/etlctl/pkg/util/etl"
	_ "shadow-dom/etlctl/pkg/util/etl/sources"
	_ "shadow-dom/etlctl/pkg/util/etl/targets"

	"github.com/spf13/cobra"
)

var listenCmd = &cobra.Command{
	Use:   "listen [name]",
	Short: "Run an event-driven ETL pipeline continuously",
	Long:  `Starts an ETL pipeline in listener mode for event-driven sources like RabbitMQ or webhooks. Runs until interrupted.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("Listening ETL: %s\n", name)
		return etl.Listen(configDir, name)
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)
}
