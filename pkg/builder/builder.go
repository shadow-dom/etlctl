package builder

import (
	"fmt"
	"os"
	"shadow-dom/etlctl/pkg/util/etl"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

func GenerateETL(etlFile string) {
	data, err := os.ReadFile("../etls/" + etlFile + ".yaml")

	if err != nil {
		fmt.Println("Error reading YAML file:", err)
		return
	}

	var etl etl.ETL

	if err := yaml.Unmarshal(data, &etl); err != nil {
		fmt.Println("Error parsing YAML:", err)
		return
	}

	functions := template.FuncMap{
		"join": func(items []string, sep string) string {
			return `"` + strings.Join(items, `", "`) + `"`
		},
	}

	content, err := os.ReadFile("../pkg/builder/etl.tmpl")
	if err != nil {
		fmt.Println("Error reading template file:", err)
		return
	}

	template, err := template.New("etl").Funcs(functions).Parse(string(content))

	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	fileName := "./util/etl/gen/" + etlFile + ".go"
	file, err := os.Create(fileName)

	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	err = template.Execute(file, etl)

	if err != nil {
		fmt.Println("Error writing file:", err)
	}

	fmt.Println("ETL Go file generated:", fileName)
}
