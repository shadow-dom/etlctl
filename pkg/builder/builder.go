package builder

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"shadow-dom/etlctl/pkg/util/etl"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed etl.tmpl
var templateFS embed.FS

func GenerateETL(configDir string, name string, outputDir string) error {
	configPath := filepath.Join(configDir, name+".yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading YAML file: %w", err)
	}

	var e etl.ETL
	if err := yaml.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("error parsing YAML: %w", err)
	}

	functions := template.FuncMap{
		"join": func(items []string, sep string) string {
			return `"` + strings.Join(items, `", "`) + `"`
		},
	}

	content, err := templateFS.ReadFile("etl.tmpl")
	if err != nil {
		return fmt.Errorf("error reading template: %w", err)
	}

	tmpl, err := template.New("etl").Funcs(functions).Parse(string(content))
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	if outputDir == "" {
		outputDir = filepath.Join(configDir, "gen")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	fileName := filepath.Join(outputDir, name+".go")
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, e); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Println("ETL Go file generated:", fileName)
	return nil
}
