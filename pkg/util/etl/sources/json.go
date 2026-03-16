package sources

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"shadow-dom/etlctl/pkg/util/etl"
)

func expandJSONPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		usr, err := user.Current()
		if err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	}
	return path
}

type JSONSource struct {
	name       string
	filepath   string
	inlineData string
	data       []map[string]string
}

func (s *JSONSource) Name() string { return s.name }
func (s *JSONSource) Type() string { return "json" }

func (s *JSONSource) Connect() error {
	var raw []byte
	if s.inlineData != "" {
		raw = []byte(s.inlineData)
	} else {
		var err error
		raw, err = os.ReadFile(s.filepath)
		if err != nil {
			return fmt.Errorf("failed to read JSON file %s: %w", s.filepath, err)
		}
	}

	// Try array of objects first
	var records []map[string]any
	if err := json.Unmarshal(raw, &records); err != nil {
		// Try single object wrapped in array
		var single map[string]any
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		records = []map[string]any{single}
	}

	// Convert all values to strings
	s.data = make([]map[string]string, len(records))
	for i, record := range records {
		row := make(map[string]string)
		for k, v := range record {
			row[k] = fmt.Sprintf("%v", v)
		}
		s.data[i] = row
	}

	return nil
}

func (s *JSONSource) Extract(query string) ([]map[string]string, error) {
	return s.data, nil
}

func (s *JSONSource) Close() error {
	s.data = nil
	return nil
}

func (s *JSONSource) ListTables() ([]string, error) {
	return []string{""}, nil
}

func (s *JSONSource) DescribeTable(table string) (*etl.TableSchema, error) {
	// Read first record to get keys
	var raw []byte
	if s.inlineData != "" {
		raw = []byte(s.inlineData)
	} else {
		var err error
		raw, err = os.ReadFile(s.filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to read JSON file: %w", err)
		}
	}

	var records []map[string]any
	if err := json.Unmarshal(raw, &records); err != nil {
		var single map[string]any
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
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
	etl.RegisterSource("json", func(name string, config map[string]string) (etl.Source, error) {
		fp := config["filepath"]
		inlineData := config["inline_data"]
		if fp == "" && inlineData == "" {
			return nil, fmt.Errorf("json source %q requires 'filepath' or 'inline_data' in connection config", name)
		}
		return &JSONSource{name: name, filepath: expandJSONPath(fp), inlineData: inlineData}, nil
	})
}
