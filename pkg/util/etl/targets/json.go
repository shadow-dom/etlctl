package targets

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"shadow-dom/etlctl/pkg/util/etl"
)

func expandTargetJSONPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		usr, err := user.Current()
		if err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	}
	return path
}

type JSONTarget struct {
	name     string
	filepath string
}

func (t *JSONTarget) Name() string { return t.name }
func (t *JSONTarget) Type() string { return "json" }
func (t *JSONTarget) Connect() error { return nil }

func (t *JSONTarget) Load(tableName string, fields []etl.FieldMapping, data []map[string]string) error {
	if len(data) == 0 {
		return nil
	}

	// If no field mappings, pass through all columns
	if len(fields) == 0 {
		fields = etl.IdentityMappings(data[0])
	}

	// Map source fields to target field names
	output := make([]map[string]string, len(data))
	for i, row := range data {
		mapped := make(map[string]string)
		for _, f := range fields {
			mapped[f.Target] = row[f.Source]
		}
		output[i] = mapped
	}

	raw, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(t.filepath, raw, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

func (t *JSONTarget) Close() error { return nil }

func (t *JSONTarget) ListTables() ([]string, error) {
	return []string{""}, nil
}

func (t *JSONTarget) DescribeTable(table string) (*etl.TableSchema, error) {
	raw, err := os.ReadFile(t.filepath)
	if err != nil {
		// File may not exist yet
		return &etl.TableSchema{Table: ""}, nil
	}

	var records []map[string]any
	if err := json.Unmarshal(raw, &records); err != nil {
		var single map[string]any
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return &etl.TableSchema{Table: ""}, nil
		}
		records = []map[string]any{single}
	}

	schema := &etl.TableSchema{Table: ""}
	if len(records) > 0 {
		for k := range records[0] {
			schema.Columns = append(schema.Columns, etl.ColumnInfo{
				Name:     k,
				DataType: "string",
				Nullable: true,
			})
		}
	}
	return schema, nil
}

func init() {
	etl.RegisterTarget("json", func(name string, config map[string]string) (etl.Target, error) {
		fp := config["filepath"]
		if fp == "" {
			return nil, fmt.Errorf("json target %q requires 'filepath' in connection config", name)
		}
		return &JSONTarget{name: name, filepath: expandTargetJSONPath(fp)}, nil
	})
}
