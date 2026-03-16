package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"shadow-dom/etlctl/pkg/util/etl"
)

type WebhookSource struct {
	name    string
	path    string
	port    string
	timeout time.Duration
	server  *http.Server
	events  chan map[string]string
}

func (s *WebhookSource) Name() string     { return s.name }
func (s *WebhookSource) Type() string     { return "webhook" }
func (s *WebhookSource) IsListener() bool { return true }

func (s *WebhookSource) Connect() error {
	s.events = make(chan map[string]string, 1000)

	mux := http.NewServeMux()
	mux.HandleFunc(s.path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Try array of objects
		var records []map[string]any
		if err := json.Unmarshal(body, &records); err != nil {
			// Try single object
			var single map[string]any
			if err2 := json.Unmarshal(body, &single); err2 != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			records = []map[string]any{single}
		}

		for _, record := range records {
			row := make(map[string]string)
			for k, v := range record {
				row[k] = fmt.Sprintf("%v", v)
			}
			select {
			case s.events <- row:
			default:
				http.Error(w, "buffer full", http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`))
	})

	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}

	go s.server.ListenAndServe()
	fmt.Printf("Webhook listening on :%s%s\n", s.port, s.path)

	return nil
}

func (s *WebhookSource) Extract(query string) ([]map[string]string, error) {
	var results []map[string]string

	if s.timeout == 0 {
		// Zero timeout: block until at least one event, then return immediately
		row, ok := <-s.events
		if !ok {
			return nil, fmt.Errorf("webhook event channel closed")
		}
		results = append(results, row)
		// Drain any additional buffered events
		for {
			select {
			case r, ok := <-s.events:
				if !ok {
					return results, nil
				}
				results = append(results, r)
			default:
				return results, nil
			}
		}
	}

	// Positive timeout: collect events within the window
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	for {
		select {
		case row, ok := <-s.events:
			if !ok {
				return results, nil
			}
			results = append(results, row)
		case <-ctx.Done():
			return results, nil
		}
	}
}

func (s *WebhookSource) Close() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
	if s.events != nil {
		close(s.events)
	}
	return nil
}

func (s *WebhookSource) ListTables() ([]string, error) {
	return []string{""}, nil
}

func (s *WebhookSource) DescribeTable(table string) (*etl.TableSchema, error) {
	return &etl.TableSchema{Table: ""}, nil
}

func init() {
	etl.RegisterSource("webhook", func(name string, config map[string]string) (etl.Source, error) {
		path := config["path"]
		if path == "" {
			return nil, fmt.Errorf("webhook source %q requires 'path' in connection config", name)
		}

		port := config["port"]
		if port == "" {
			port = "9090"
		}

		var timeout time.Duration
		if v := config["timeout"]; v != "" && v != "0" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("webhook source %q: invalid timeout: %w", name, err)
			}
			timeout = d
		}

		return &WebhookSource{
			name:    name,
			path:    path,
			port:    port,
			timeout: timeout,
		}, nil
	})
}
