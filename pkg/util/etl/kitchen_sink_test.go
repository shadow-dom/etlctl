package etl_test

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"shadow-dom/etlctl/pkg/util/etl"
	_ "shadow-dom/etlctl/pkg/util/etl/sources"
	_ "shadow-dom/etlctl/pkg/util/etl/targets"
	"shadow-dom/etlctl/pkg/util/etl/transform"
)

// testAPIServer provides GET /items (returns JSON array) and POST /items (accepts JSON array).
type testAPIServer struct {
	mu       sync.Mutex
	received []map[string]string
	items    []map[string]string
}

func newTestAPIServer() *testAPIServer {
	return &testAPIServer{
		items: []map[string]string{
			{"id": "1", "title": "API Item A", "status": "active"},
			{"id": "2", "title": "API Item B", "status": "inactive"},
			{"id": "3", "title": "API Item C", "status": "active"},
		},
	}
}

func (s *testAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET" && r.URL.Path == "/items":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.items)
	case r.Method == "POST" && r.URL.Path == "/items":
		var data []map[string]string
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		s.mu.Lock()
		s.received = append(s.received, data...)
		s.mu.Unlock()
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"accepted": %d}`, len(data))
	default:
		http.NotFound(w, r)
	}
}

func (s *testAPIServer) Received() []map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]string{}, s.received...)
}

// setupTestDir creates a temp directory with seed data and returns cleanup func.
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Copy CSV
	csvData := `id,name,email,department,salary
1,Alice Johnson,alice@example.com,Engineering,95000
2,Bob Smith,bob@example.com,Marketing,72000
3,Carol White,carol@example.com,Engineering,105000
4,David Brown,david@example.com,Sales,68000
5,Eve Davis,eve@example.com,Marketing,77000
`
	os.WriteFile(filepath.Join(dir, "employees.csv"), []byte(csvData), 0644)

	// Create JSON
	jsonData := `[
  {"sku": "WIDGET-001", "name": "Blue Widget", "price": "19.99", "category": "Widgets"},
  {"sku": "GADGET-002", "name": "red gadget", "price": "49.50", "category": "Gadgets"},
  {"sku": "WIDGET-003", "name": "green widget", "price": "24.99", "category": "Widgets"}
]`
	os.WriteFile(filepath.Join(dir, "products.json"), []byte(jsonData), 0644)

	// Create seed SQLite DB
	dbPath := filepath.Join(dir, "seed.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("create seed db: %v", err)
	}
	defer db.Close()

	db.Exec(`CREATE TABLE orders (
		order_id TEXT, customer TEXT, product TEXT, quantity TEXT, total TEXT
	)`)
	db.Exec(`INSERT INTO orders VALUES ('ORD-001','Alice','Widget',    '2','39.98')`)
	db.Exec(`INSERT INTO orders VALUES ('ORD-002','Bob',  'Gadget',   '1','49.50')`)
	db.Exec(`INSERT INTO orders VALUES ('ORD-003','Carol','Widget',   '5','99.95')`)
	db.Exec(`INSERT INTO orders VALUES ('ORD-004','Alice','Gadget',   '1','49.50')`)

	return dir
}

func init() {
	// Register a custom Go transform for testing
	transform.Register("email_domain", func(value string) (string, error) {
		parts := strings.SplitN(value, "@", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid email: %s", value)
		}
		return parts[1], nil
	})
}

// writeYAML is a helper to write an ETL config YAML to the given directory.
func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}

// --- Individual pipeline tests ---

func TestCSVToSQLite(t *testing.T) {
	dir := setupTestDir(t)
	outDB := filepath.Join(dir, "out.db")

	writeYAML(t, dir, "csv-to-sqlite", fmt.Sprintf(`
name: csv-to-sqlite
sources:
  - name: employees
    type: csv
    connection:
      filepath: %s/employees.csv
targets:
  - name: empdb
    type: sqlite3
    connection:
      filepath: %s
pipelines:
  - name: load_employees
    sources: [employees]
    target: empdb.employees
    fields:
      - { source: id, target: id }
      - { source: name, target: full_name }
      - { source: email, target: email }
      - { source: department, target: dept }
      - { source: salary, target: salary }
`, dir, outDB))

	if err := etl.Run(dir, "csv-to-sqlite"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify
	db, err := sql.Open("sqlite3", outDB)
	if err != nil {
		t.Fatalf("open output db: %v", err)
	}
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM employees").Scan(&count)
	if count != 5 {
		t.Errorf("expected 5 rows, got %d", count)
	}

	var name string
	db.QueryRow("SELECT full_name FROM employees WHERE id = '1'").Scan(&name)
	if name != "Alice Johnson" {
		t.Errorf("expected 'Alice Johnson', got %q", name)
	}
}

func TestJSONToCSV(t *testing.T) {
	dir := setupTestDir(t)
	outCSV := filepath.Join(dir, "products_out.csv")

	writeYAML(t, dir, "json-to-csv", fmt.Sprintf(`
name: json-to-csv
sources:
  - name: products
    type: json
    connection:
      filepath: %s/products.json
targets:
  - name: products_csv
    type: csv
    connection:
      filepath: %s
pipelines:
  - name: export_products
    sources: [products]
    target: products_csv
    fields:
      - { source: sku, target: product_sku }
      - { source: name, target: product_name, transform: "title" }
      - { source: price, target: unit_price }
      - { source: category, target: category }
`, dir, outCSV))

	if err := etl.Run(dir, "json-to-csv"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	f, err := os.Open(outCSV)
	if err != nil {
		t.Fatalf("open output csv: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	// Header + 3 data rows
	if len(records) != 4 {
		t.Fatalf("expected 4 rows (1 header + 3 data), got %d", len(records))
	}

	// Check header
	if records[0][0] != "product_sku" || records[0][1] != "product_name" {
		t.Errorf("unexpected headers: %v", records[0])
	}

	// Check title transform applied (second row was "red gadget" -> "Red Gadget")
	if records[2][1] != "Red Gadget" {
		t.Errorf("expected 'Red Gadget' (title transform), got %q", records[2][1])
	}
}

func TestSQLiteToJSON(t *testing.T) {
	dir := setupTestDir(t)
	outJSON := filepath.Join(dir, "orders_out.json")

	writeYAML(t, dir, "sqlite-to-json", fmt.Sprintf(`
name: sqlite-to-json
sources:
  - name: orderdb
    type: sqlite3
    connection:
      filepath: %s/seed.db
targets:
  - name: orders_json
    type: json
    connection:
      filepath: %s
queries:
  - name: all_orders
    sql: "SELECT order_id, customer, product, quantity, total FROM orders"
pipelines:
  - name: export_orders
    sources: [orderdb]
    target: orders_json
    query: all_orders
`, dir, outJSON))

	if err := etl.Run(dir, "sqlite-to-json"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	raw, err := os.ReadFile(outJSON)
	if err != nil {
		t.Fatalf("read output json: %v", err)
	}

	var records []map[string]string
	if err := json.Unmarshal(raw, &records); err != nil {
		t.Fatalf("parse json: %v", err)
	}

	if len(records) != 4 {
		t.Errorf("expected 4 orders, got %d", len(records))
	}

	if records[0]["customer"] != "Alice" {
		t.Errorf("expected 'Alice', got %q", records[0]["customer"])
	}
}

func TestAPIToSQLite(t *testing.T) {
	srv := newTestAPIServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	dir := setupTestDir(t)
	outDB := filepath.Join(dir, "api_out.db")

	writeYAML(t, dir, "api-to-sqlite", fmt.Sprintf(`
name: api-to-sqlite
sources:
  - name: api_items
    type: api
    connection:
      url: %s/items
      method: GET
targets:
  - name: itemsdb
    type: sqlite3
    connection:
      filepath: %s
pipelines:
  - name: ingest_items
    sources: [api_items]
    target: itemsdb.items
    fields:
      - { source: id, target: item_id }
      - { source: title, target: title }
      - { source: status, target: status, transform: "uppercase" }
`, ts.URL, outDB))

	if err := etl.Run(dir, "api-to-sqlite"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	db, err := sql.Open("sqlite3", outDB)
	if err != nil {
		t.Fatalf("open output db: %v", err)
	}
	defer db.Close()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 items, got %d", count)
	}

	var status string
	db.QueryRow("SELECT status FROM items WHERE item_id = '1'").Scan(&status)
	if status != "ACTIVE" {
		t.Errorf("expected 'ACTIVE' (uppercase transform), got %q", status)
	}
}

func TestSQLiteToAPI(t *testing.T) {
	srv := newTestAPIServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	dir := setupTestDir(t)

	writeYAML(t, dir, "sqlite-to-api", fmt.Sprintf(`
name: sqlite-to-api
sources:
  - name: orderdb
    type: sqlite3
    connection:
      filepath: %s/seed.db
targets:
  - name: api_target
    type: api
    connection:
      url: %s/items
      method: POST
queries:
  - name: all_orders
    sql: "SELECT order_id, customer, product FROM orders"
pipelines:
  - name: push_orders
    sources: [orderdb]
    target: api_target
    query: all_orders
`, dir, ts.URL))

	if err := etl.Run(dir, "sqlite-to-api"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	received := srv.Received()
	if len(received) != 4 {
		t.Errorf("expected API to receive 4 records, got %d", len(received))
	}
}

func TestCustomGoTransform(t *testing.T) {
	dir := setupTestDir(t)
	outJSON := filepath.Join(dir, "domains.json")

	writeYAML(t, dir, "custom-transform", fmt.Sprintf(`
name: custom-transform
sources:
  - name: employees
    type: csv
    connection:
      filepath: %s/employees.csv
targets:
  - name: domain_report
    type: json
    connection:
      filepath: %s
pipelines:
  - name: extract_domains
    sources: [employees]
    target: domain_report
    fields:
      - { source: name, target: name }
      - { source: email, target: email_domain, transform: "email_domain" }
      - { source: salary, target: salary_doubled, transform: "multiply:2" }
`, dir, outJSON))

	if err := etl.Run(dir, "custom-transform"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	raw, err := os.ReadFile(outJSON)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	var records []map[string]string
	if err := json.Unmarshal(raw, &records); err != nil {
		t.Fatalf("parse json: %v", err)
	}

	if len(records) != 5 {
		t.Errorf("expected 5 records, got %d", len(records))
	}

	// Check custom email_domain transform
	if records[0]["email_domain"] != "example.com" {
		t.Errorf("expected 'example.com', got %q", records[0]["email_domain"])
	}

	// Check multiply:2 transform (95000 * 2 = 190000)
	if records[0]["salary_doubled"] != "190000" {
		t.Errorf("expected '190000', got %q", records[0]["salary_doubled"])
	}
}

func TestChainedTransforms(t *testing.T) {
	dir := setupTestDir(t)
	outJSON := filepath.Join(dir, "chained.json")

	writeYAML(t, dir, "chained-transform", fmt.Sprintf(`
name: chained-transform
sources:
  - name: products
    type: json
    connection:
      filepath: %s/products.json
targets:
  - name: chained_out
    type: json
    connection:
      filepath: %s
pipelines:
  - name: chain_test
    sources: [products]
    target: chained_out
    fields:
      - { source: name, target: product_name, transform: "trim|uppercase" }
      - { source: price, target: display_price, transform: "prefix:$" }
`, dir, outJSON))

	if err := etl.Run(dir, "chained-transform"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	raw, err := os.ReadFile(outJSON)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	var records []map[string]string
	json.Unmarshal(raw, &records)

	if records[0]["product_name"] != "BLUE WIDGET" {
		t.Errorf("expected 'BLUE WIDGET', got %q", records[0]["product_name"])
	}

	if records[0]["display_price"] != "$19.99" {
		t.Errorf("expected '$19.99', got %q", records[0]["display_price"])
	}
}

func TestDeduplication(t *testing.T) {
	dir := setupTestDir(t)

	// CSV with duplicates
	csvData := `id,name,value
1,Alice,100
2,Bob,200
1,Alice,150
3,Carol,300
2,Bob,250
`
	os.WriteFile(filepath.Join(dir, "dupes.csv"), []byte(csvData), 0644)

	outJSON := filepath.Join(dir, "deduped.json")

	writeYAML(t, dir, "dedup-test", fmt.Sprintf(`
name: dedup-test
sources:
  - name: dupes
    type: csv
    connection:
      filepath: %s/dupes.csv
targets:
  - name: deduped
    type: json
    connection:
      filepath: %s
pipelines:
  - name: dedup_pipeline
    sources: [dupes]
    target: deduped
    uniqueFields: [id]
    keepLast: true
`, dir, outJSON))

	if err := etl.Run(dir, "dedup-test"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	raw, _ := os.ReadFile(outJSON)
	var records []map[string]string
	json.Unmarshal(raw, &records)

	if len(records) != 3 {
		t.Errorf("expected 3 unique records, got %d", len(records))
	}

	// keepLast: Alice should have value 150, Bob 250
	for _, r := range records {
		switch r["id"] {
		case "1":
			if r["value"] != "150" {
				t.Errorf("Alice should have value 150 (keepLast), got %s", r["value"])
			}
		case "2":
			if r["value"] != "250" {
				t.Errorf("Bob should have value 250 (keepLast), got %s", r["value"])
			}
		}
	}
}

func TestMultiTargetPipeline(t *testing.T) {
	dir := setupTestDir(t)
	outJSON := filepath.Join(dir, "multi_out.json")
	outCSV := filepath.Join(dir, "multi_out.csv")

	writeYAML(t, dir, "multi-target", fmt.Sprintf(`
name: multi-target
sources:
  - name: employees
    type: csv
    connection:
      filepath: %s/employees.csv
targets:
  - name: json_out
    type: json
    connection:
      filepath: %s
  - name: csv_out
    type: csv
    connection:
      filepath: %s
pipelines:
  - name: fan_out
    sources: [employees]
    targets: [json_out, csv_out]
    fields:
      - { source: name, target: name }
      - { source: department, target: dept }
`, dir, outJSON, outCSV))

	if err := etl.Run(dir, "multi-target"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check JSON output
	raw, err := os.ReadFile(outJSON)
	if err != nil {
		t.Fatalf("read json output: %v", err)
	}
	var jsonRecords []map[string]string
	json.Unmarshal(raw, &jsonRecords)
	if len(jsonRecords) != 5 {
		t.Errorf("JSON: expected 5 records, got %d", len(jsonRecords))
	}

	// Check CSV output
	f, err := os.Open(outCSV)
	if err != nil {
		t.Fatalf("open csv output: %v", err)
	}
	defer f.Close()
	csvRecords, _ := csv.NewReader(f).ReadAll()
	if len(csvRecords) != 6 { // 1 header + 5 data
		t.Errorf("CSV: expected 6 rows (header + 5), got %d", len(csvRecords))
	}
}

// TestKitchenSinkFullRun runs a single ETL config that exercises
// CSV, JSON, SQLite, and API sources/targets together with transforms.
func TestKitchenSinkFullRun(t *testing.T) {
	srv := newTestAPIServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	dir := setupTestDir(t)
	empDB := filepath.Join(dir, "emp_out.db")
	ordersJSON := filepath.Join(dir, "orders_export.json")
	productsCSV := filepath.Join(dir, "products_export.csv")
	domainsJSON := filepath.Join(dir, "domains.json")

	writeYAML(t, dir, "kitchen-sink", fmt.Sprintf(`
name: kitchen-sink
sources:
  - name: csv_employees
    type: csv
    connection:
      filepath: %s/employees.csv
  - name: json_products
    type: json
    connection:
      filepath: %s/products.json
  - name: sqlite_orders
    type: sqlite3
    connection:
      filepath: %s/seed.db
  - name: api_items
    type: api
    connection:
      url: %s/items
      method: GET
targets:
  - name: emp_db
    type: sqlite3
    connection:
      filepath: %s
  - name: orders_json
    type: json
    connection:
      filepath: %s
  - name: products_csv
    type: csv
    connection:
      filepath: %s
  - name: api_sink
    type: api
    connection:
      url: %s/items
      method: POST
  - name: domains_json
    type: json
    connection:
      filepath: %s
queries:
  - name: all_orders
    sql: "SELECT order_id, customer, product, quantity, total FROM orders"
pipelines:
  - name: csv_to_sqlite
    sources: [csv_employees]
    target: emp_db.employees
    fields:
      - { source: id, target: id }
      - { source: name, target: full_name }
      - { source: email, target: email }
      - { source: department, target: dept }
      - { source: salary, target: salary }
  - name: sqlite_to_json
    sources: [sqlite_orders]
    target: orders_json
    query: all_orders
  - name: json_to_csv
    sources: [json_products]
    target: products_csv
    fields:
      - { source: sku, target: product_sku }
      - { source: name, target: product_name, transform: "title" }
      - { source: price, target: unit_price }
  - name: api_to_sqlite
    sources: [api_items]
    target: emp_db.api_items
    fields:
      - { source: id, target: item_id }
      - { source: title, target: title, transform: "uppercase" }
      - { source: status, target: status }
  - name: csv_to_api
    sources: [csv_employees]
    target: api_sink
    fields:
      - { source: name, target: employee_name }
      - { source: department, target: dept }
  - name: custom_transform
    sources: [csv_employees]
    target: domains_json
    fields:
      - { source: name, target: name }
      - { source: email, target: domain, transform: "email_domain" }
      - { source: salary, target: doubled_salary, transform: "multiply:2" }
`, dir, dir, dir, ts.URL, empDB, ordersJSON, productsCSV, ts.URL, domainsJSON))

	if err := etl.Run(dir, "kitchen-sink"); err != nil {
		t.Fatalf("kitchen-sink Run failed: %v", err)
	}

	// --- Verify all outputs ---

	// 1. CSV -> SQLite
	db, err := sql.Open("sqlite3", empDB)
	if err != nil {
		t.Fatalf("open emp db: %v", err)
	}
	defer db.Close()

	var empCount int
	db.QueryRow("SELECT COUNT(*) FROM employees").Scan(&empCount)
	if empCount != 5 {
		t.Errorf("employees table: expected 5, got %d", empCount)
	}

	// 2. API -> SQLite (same db)
	var itemCount int
	db.QueryRow("SELECT COUNT(*) FROM api_items").Scan(&itemCount)
	if itemCount != 3 {
		t.Errorf("api_items table: expected 3, got %d", itemCount)
	}
	var title string
	db.QueryRow("SELECT title FROM api_items WHERE item_id = '1'").Scan(&title)
	if title != "API ITEM A" {
		t.Errorf("api_items uppercase transform: expected 'API ITEM A', got %q", title)
	}

	// 3. SQLite -> JSON
	raw, err := os.ReadFile(ordersJSON)
	if err != nil {
		t.Fatalf("read orders json: %v", err)
	}
	var orders []map[string]string
	json.Unmarshal(raw, &orders)
	if len(orders) != 4 {
		t.Errorf("orders json: expected 4, got %d", len(orders))
	}

	// 4. JSON -> CSV
	csvF, err := os.Open(productsCSV)
	if err != nil {
		t.Fatalf("open products csv: %v", err)
	}
	defer csvF.Close()
	csvRows, _ := csv.NewReader(csvF).ReadAll()
	if len(csvRows) != 4 { // header + 3 data
		t.Errorf("products csv: expected 4 rows, got %d", len(csvRows))
	}
	// Title transform: "red gadget" -> "Red Gadget"
	if csvRows[2][1] != "Red Gadget" {
		t.Errorf("title transform: expected 'Red Gadget', got %q", csvRows[2][1])
	}

	// 5. CSV -> API
	received := srv.Received()
	if len(received) < 5 {
		t.Errorf("api sink: expected at least 5 records, got %d", len(received))
	}

	// 6. Custom Go transform (email_domain + multiply:2)
	raw, err = os.ReadFile(domainsJSON)
	if err != nil {
		t.Fatalf("read domains json: %v", err)
	}
	var domains []map[string]string
	json.Unmarshal(raw, &domains)
	if len(domains) != 5 {
		t.Errorf("domains: expected 5, got %d", len(domains))
	}
	if domains[0]["domain"] != "example.com" {
		t.Errorf("email_domain transform: expected 'example.com', got %q", domains[0]["domain"])
	}
	if domains[0]["doubled_salary"] != "190000" {
		t.Errorf("multiply:2 transform: expected '190000', got %q", domains[0]["doubled_salary"])
	}
}
