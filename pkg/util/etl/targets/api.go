package targets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"shadow-dom/etlctl/pkg/util/etl"
)

type APITarget struct {
	name    string
	url     string
	method  string
	headers map[string]string
	client  *http.Client
}

func (t *APITarget) Name() string { return t.name }
func (t *APITarget) Type() string { return "api" }

func (t *APITarget) Connect() error {
	t.client = &http.Client{}
	return nil
}

func (t *APITarget) Load(tableName string, fields []etl.FieldMapping, data []map[string]string) error {
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

	body, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequest(t.method, t.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

func (t *APITarget) Close() error { return nil }

func init() {
	etl.RegisterTarget("api", func(name string, config map[string]string) (etl.Target, error) {
		url := config["url"]
		if url == "" {
			return nil, fmt.Errorf("api target %q requires 'url' in connection config", name)
		}
		method := config["method"]
		if method == "" {
			method = "POST"
		}

		headers := make(map[string]string)
		if raw := config["headers"]; raw != "" {
			for _, pair := range strings.Split(raw, ";") {
				parts := strings.SplitN(pair, ":", 2)
				if len(parts) == 2 {
					headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}

		if token := config["auth_token"]; token != "" {
			authType := config["auth_type"]
			if authType == "" || authType == "none" {
				authType = "Bearer"
			}
			headers["Authorization"] = authType + " " + token
		}

		return &APITarget{
			name:    name,
			url:     url,
			method:  strings.ToUpper(method),
			headers: headers,
		}, nil
	})
}
