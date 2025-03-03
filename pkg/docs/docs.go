package docs

import (
	"fmt"
	"os"
	"shadow-dom/etlctl/pkg/util/etl"
	"strings"
)

func GenerateETLDocs(etlConfig etl.ETL) {
	// Create Markdown file
	docFile, err := os.Create("etl_doc.md")
	if err != nil {
		fmt.Println("Error creating documentation file:", err)
		return
	}
	defer docFile.Close()

	// Create Mermaid file
	diagramFile, err := os.Create("etl_diagram.md")
	if err != nil {
		fmt.Println("Error creating diagram file:", err)
		return
	}
	defer diagramFile.Close()

	// Write Markdown documentation
	docContent := fmt.Sprintf("# ETL Configuration\n\n")

	// Sources
	docContent += "## Sources\n"
	for _, src := range etlConfig.Sources {
		docContent += fmt.Sprintf("- **%s** (%s) → `%s`\n", src.Name, src.Type, src.Connection["filepath"])
	}

	// Queries
	docContent += "\n## Queries\n"
	for _, query := range etlConfig.Queries {
		docContent += fmt.Sprintf("- **%s**\n  ```sql\n  %s\n  ```\n", query.Name, query.SQL)
	}

	// Targets
	docContent += "\n## Targets\n"
	for _, tgt := range etlConfig.Targets {
		docContent += fmt.Sprintf("- **%s** (%s) → `%s`\n", tgt.Name, tgt.Type, tgt.Connection["filepath"])
	}

	// Pipelines
	docContent += "\n## Pipelines\n"
	for i, pipeline := range etlConfig.Pipelines {
		docContent += fmt.Sprintf("\n### **Pipeline %d**\n", i+1)
		docContent += fmt.Sprintf("- **Sources:** %s\n", strings.Join(pipeline.Sources, ", "))
		docContent += fmt.Sprintf("- **Query:** `%s`\n", pipeline.Query)
		docContent += fmt.Sprintf("- **Target:** `%s`\n", pipeline.Target)
		docContent += "- **Field Mappings:**\n"
		for _, field := range pipeline.Fields {
			docContent += fmt.Sprintf("  - `%s` → `%s`\n", field.Source, field.Target)
		}
	}

	docFile.WriteString(docContent)

	// Write Mermaid diagram
	diagramContent := "```mermaid\ngraph TD;\n"
	for _, pipeline := range etlConfig.Pipelines {
		for _, src := range pipeline.Sources {
			diagramContent += fmt.Sprintf("    %s -->|%s| %s;\n", src, pipeline.Query, pipeline.Target)
		}
	}
	diagramContent += "```\n"

	diagramFile.WriteString(diagramContent)

	fmt.Println("ETL documentation generated: etl_doc.md and etl_diagram.md")
}
