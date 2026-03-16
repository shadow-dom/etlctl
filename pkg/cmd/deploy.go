package cmd

import (
	"fmt"
	"path/filepath"

	"shadow-dom/etlctl/pkg/k8s"
	"shadow-dom/etlctl/pkg/util/etl"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [name]",
	Short: "Generate Kubernetes CronJob deployment manifests",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Validate the ETL config exists
		e, err := etl.CreateETL(configDir, name)
		if err != nil {
			return fmt.Errorf("failed to load ETL config: %w", err)
		}

		schedule, _ := cmd.Flags().GetString("schedule")
		namespace, _ := cmd.Flags().GetString("namespace")
		image, _ := cmd.Flags().GetString("image")
		outputDir, _ := cmd.Flags().GetString("output")
		mode, _ := cmd.Flags().GetString("mode")

		// Use deploy section from YAML if present, CLI flags override
		if e.Deploy.Schedule != "" && !cmd.Flags().Changed("schedule") {
			schedule = e.Deploy.Schedule
		}
		if e.Deploy.Namespace != "" && !cmd.Flags().Changed("namespace") {
			namespace = e.Deploy.Namespace
		}
		if e.Deploy.Image != "" && !cmd.Flags().Changed("image") {
			image = e.Deploy.Image
		}
		if e.Deploy.Mode != "" && !cmd.Flags().Changed("mode") {
			mode = e.Deploy.Mode
		}

		configFile := filepath.Join(configDir, name+".yaml")

		fmt.Printf("Generating K8s manifests for ETL: %s\n", name)

		return k8s.GenerateManifests(k8s.ManifestOptions{
			ETLName:    name,
			Namespace:  namespace,
			Schedule:   schedule,
			Image:      image,
			OutputDir:  outputDir,
			ConfigFile: configFile,
			Mode:       mode,
		})
	},
}

func init() {
	deployCmd.Flags().String("schedule", "0 0 * * *", "Cron schedule expression")
	deployCmd.Flags().String("namespace", "default", "Kubernetes namespace")
	deployCmd.Flags().String("image", "etlctl:latest", "Container image")
	deployCmd.Flags().StringP("output", "o", "./deploy", "Output directory for manifests")
	deployCmd.Flags().String("mode", "cronjob", "Deployment mode: 'cronjob' or 'service'")
	rootCmd.AddCommand(deployCmd)
}
