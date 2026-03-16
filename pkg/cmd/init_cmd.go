package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

const etlTemplate = `name: "{{.Name}}"
sources:
  - name: "source_1"
    type: "sqlite3"
    connection:
      filepath: "./data/source.db"
queries:
  - name: "get_data"
    sql: |
      SELECT id, col1, col2 FROM table_name;
targets:
  - name: "target_1"
    type: "sqlite3"
    connection:
      filepath: "./data/target.db"
pipelines:
  - name: "pipeline_1"
    sources:
      - "source_1"
    query: "get_data"
    target: "target_1.table_name"
    fields:
      - source: "col1"
        target: "col1"
      - source: "col2"
        target: "col2"
    tracking:
      field: "id"
      storage_type: file
`

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold a new ETL pipeline definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		outPath := filepath.Join(configDir, name+".yaml")

		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("ETL definition %q already exists at %s", name, outPath)
		}

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create state directory
		if err := os.MkdirAll(filepath.Join(configDir, "state"), 0755); err != nil {
			return fmt.Errorf("failed to create state directory: %w", err)
		}

		tmpl, err := template.New("etl").Parse(etlTemplate)
		if err != nil {
			return err
		}

		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, struct{ Name string }{Name: name}); err != nil {
			return err
		}

		fmt.Printf("Created ETL definition: %s\n", outPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
