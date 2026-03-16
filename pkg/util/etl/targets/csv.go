package targets

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"shadow-dom/etlctl/pkg/util/etl"
)

func expandTargetCSVPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		usr, err := user.Current()
		if err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	}
	return path
}

type CSVTarget struct {
	name      string
	filepath  string
	delimiter rune
}

func (t *CSVTarget) Name() string { return t.name }
func (t *CSVTarget) Type() string { return "csv" }
func (t *CSVTarget) Connect() error { return nil }

func (t *CSVTarget) Load(tableName string, fields []etl.FieldMapping, data []map[string]string) error {
	if len(data) == 0 {
		return nil
	}

	f, err := os.Create(t.filepath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = t.delimiter

	// If no field mappings, pass through all columns
	if len(fields) == 0 {
		fields = etl.IdentityMappings(data[0])
	}

	// Write header row using target field names
	headers := make([]string, len(fields))
	for i, field := range fields {
		headers[i] = field.Target
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, row := range data {
		record := make([]string, len(fields))
		for i, field := range fields {
			record[i] = row[field.Source]
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	return writer.Error()
}

func (t *CSVTarget) Close() error { return nil }

func (t *CSVTarget) ListTables() ([]string, error) {
	return []string{""}, nil
}

func (t *CSVTarget) DescribeTable(table string) (*etl.TableSchema, error) {
	f, err := os.Open(t.filepath)
	if err != nil {
		// File may not exist yet for a target — return empty schema
		return &etl.TableSchema{Table: ""}, nil
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = t.delimiter
	headers, err := reader.Read()
	if err != nil {
		return &etl.TableSchema{Table: ""}, nil
	}

	schema := &etl.TableSchema{Table: ""}
	for _, h := range headers {
		schema.Columns = append(schema.Columns, etl.ColumnInfo{
			Name:     h,
			DataType: "string",
			Nullable: true,
		})
	}
	return schema, nil
}

func init() {
	etl.RegisterTarget("csv", func(name string, config map[string]string) (etl.Target, error) {
		fp := config["filepath"]
		if fp == "" {
			return nil, fmt.Errorf("csv target %q requires 'filepath' in connection config", name)
		}
		delimiter := ','
		if d, ok := config["delimiter"]; ok && len(d) > 0 {
			delimiter = rune(d[0])
		}
		return &CSVTarget{name: name, filepath: expandTargetCSVPath(fp), delimiter: delimiter}, nil
	})
}
