package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"shadow-dom/etlctl/pkg/util/etl"
)

type APISource struct {
	name    string
	url     string
	method  string
	headers map[string]string
	body    string
	client  *http.Client
	data    []map[string]string
}

func (s *APISource) Name() string { return s.name }
func (s *APISource) Type() string { return "api" }

func (s *APISource) Connect() error {
	s.client = &http.Client{}

	var bodyReader io.Reader
	if s.body != "" {
		bodyReader = strings.NewReader(s.body)
	}

	req, err := http.NewRequest(s.method, s.url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read API response: %w", err)
	}

	// Try array of objects
	var records []map[string]any
	if err := json.Unmarshal(raw, &records); err != nil {
		// Try single object
		var single map[string]any
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return fmt.Errorf("failed to parse API response as JSON: %w", err)
		}
		records = []map[string]any{single}
	}

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

func (s *APISource) Extract(query string) ([]map[string]string, error) {
	return s.data, nil
}

func (s *APISource) Close() error {
	s.data = nil
	return nil
}

func (s *APISource) ListTables() ([]string, error) {
	return []string{""}, nil
}

func (s *APISource) DescribeTable(table string) (*etl.TableSchema, error) {
	// Need data to have been fetched first
	if len(s.data) == 0 {
		return nil, fmt.Errorf("no data available; call Connect() first to fetch a sample")
	}
	schema := &etl.TableSchema{Table: ""}
	for k := range s.data[0] {
		schema.Columns = append(schema.Columns, etl.ColumnInfo{
			Name:     k,
			DataType: "string",
			Nullable: true,
		})
	}
	return schema, nil
}

func parseHeaders(raw string) map[string]string {
	headers := make(map[string]string)
	if raw == "" {
		return headers
	}
	for _, pair := range strings.Split(raw, ";") {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}

func init() {
	etl.RegisterSource("api", func(name string, config map[string]string) (etl.Source, error) {
		url := config["url"]
		if url == "" {
			return nil, fmt.Errorf("api source %q requires 'url' in connection config", name)
		}
		method := config["method"]
		if method == "" {
			method = "GET"
		}

		headers := parseHeaders(config["headers"])

		// Support bearer token auth
		if token := config["auth_token"]; token != "" {
			authType := config["auth_type"]
			if authType == "" || authType == "none" {
				authType = "Bearer"
			}
			headers["Authorization"] = authType + " " + token
		}

		return &APISource{
			name:    name,
			url:     url,
			method:  strings.ToUpper(method),
			headers: headers,
			body:    config["body"],
		}, nil
	})
}
