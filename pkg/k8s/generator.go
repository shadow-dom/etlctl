package k8s

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

// ManifestOptions configures the K8s manifest generation.
type ManifestOptions struct {
	ETLName    string
	Namespace  string
	Schedule   string
	Image      string
	OutputDir  string
	ConfigFile string // path to the ETL YAML file
	Mode       string // "cronjob" (default) or "service"
}

// ConnectionInfo holds source/target connection metadata for secrets.
type ConnectionInfo struct {
	Name string
}

type templateData struct {
	ETLName        string
	Namespace      string
	Schedule       string
	Image          string
	ETLYAMLContent string
	Connections    []ConnectionInfo
}

// GenerateManifests creates K8s CronJob, ConfigMap, Secret, and Dockerfile.
func GenerateManifests(opts ManifestOptions) error {
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Read the ETL YAML for embedding in ConfigMap
	yamlContent := ""
	if opts.ConfigFile != "" {
		raw, err := os.ReadFile(opts.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to read ETL config file: %w", err)
		}
		yamlContent = indentYAML(string(raw), 4)
	}

	data := templateData{
		ETLName:        opts.ETLName,
		Namespace:      opts.Namespace,
		Schedule:       opts.Schedule,
		Image:          opts.Image,
		ETLYAMLContent: yamlContent,
	}

	mode := opts.Mode
	if mode == "" {
		mode = "cronjob"
	}

	templates := map[string]string{
		"configmap.yaml.tmpl": "configmap.yaml",
		"secret.yaml.tmpl":    "secret.yaml",
		"Dockerfile.tmpl":     "Dockerfile",
	}

	if mode == "service" {
		templates["deployment.yaml.tmpl"] = "deployment.yaml"
	} else {
		templates["cronjob.yaml.tmpl"] = "cronjob.yaml"
	}

	for tmplName, outName := range templates {
		content, err := templateFS.ReadFile("templates/" + tmplName)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tmplName, err)
		}

		tmpl, err := template.New(tmplName).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
		}

		outPath := filepath.Join(opts.OutputDir, outName)
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", outPath, err)
		}

		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
		f.Close()

		fmt.Printf("Generated: %s\n", outPath)
	}

	return nil
}

func indentYAML(content string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
