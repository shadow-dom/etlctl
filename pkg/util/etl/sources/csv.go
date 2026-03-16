package sources

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"shadow-dom/etlctl/pkg/util/etl"
)

func expandCSVPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		usr, err := user.Current()
		if err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	}
	return path
}

type CSVSource struct {
	name       string
	filepath   string
	inlineData string
	delimiter  rune
	data       []map[string]string
}

func (s *CSVSource) Name() string { return s.name }
func (s *CSVSource) Type() string { return "csv" }

func (s *CSVSource) Connect() error {
	var r io.Reader
	if s.inlineData != "" {
		r = strings.NewReader(s.inlineData)
	} else {
		f, err := os.Open(s.filepath)
		if err != nil {
			return fmt.Errorf("failed to open CSV file %s: %w", s.filepath, err)
		}
		defer f.Close()
		r = f
	}

	reader := csv.NewReader(r)
	reader.Comma = s.delimiter

	// First row is headers
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	s.data = nil
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV row: %w", err)
		}

		row := make(map[string]string)
		for i, header := range headers {
			if i < len(record) {
				row[header] = record[i]
			}
		}
		s.data = append(s.data, row)
	}

	return nil
}

func (s *CSVSource) Extract(query string) ([]map[string]string, error) {
	return s.data, nil
}

func (s *CSVSource) Close() error {
	s.data = nil
	return nil
}

func (s *CSVSource) ListTables() ([]string, error) {
	return []string{""}, nil
}

func (s *CSVSource) DescribeTable(table string) (*etl.TableSchema, error) {
	var r io.Reader
	if s.inlineData != "" {
		r = strings.NewReader(s.inlineData)
	} else {
		f, err := os.Open(s.filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to open CSV file: %w", err)
		}
		defer f.Close()
		r = f
	}

	reader := csv.NewReader(r)
	reader.Comma = s.delimiter
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
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
	etl.RegisterSource("csv", func(name string, config map[string]string) (etl.Source, error) {
		fp := config["filepath"]
		inlineData := config["inline_data"]
		if fp == "" && inlineData == "" {
			return nil, fmt.Errorf("csv source %q requires 'filepath' or 'inline_data' in connection config", name)
		}
		delimiter := ','
		if d, ok := config["delimiter"]; ok && len(d) > 0 {
			delimiter = rune(d[0])
		}
		return &CSVSource{name: name, filepath: expandCSVPath(fp), inlineData: inlineData, delimiter: delimiter}, nil
	})
}
