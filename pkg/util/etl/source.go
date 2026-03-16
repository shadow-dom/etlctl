package etl

// Source extracts data from an external system.
type Source interface {
	// Name returns the unique identifier for this source.
	Name() string
	// Type returns the source type key (e.g., "sqlite3", "postgres", "json", "csv", "api").
	Type() string
	// Connect initializes the source (open file, establish DB connection, etc.)
	Connect() error
	// Extract reads data. The query parameter is used by DB sources as SQL;
	// file/API sources may ignore it.
	Extract(query string) ([]map[string]string, error)
	// Close releases any resources.
	Close() error
}

// ColumnInfo describes a single column.
type ColumnInfo struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
}

// TableSchema describes a table's columns.
type TableSchema struct {
	Table   string       `json:"table"`
	Columns []ColumnInfo `json:"columns"`
}

// Listener is optionally implemented by Sources that receive data
// continuously (e.g. message queues, webhooks) rather than pulling on demand.
type Listener interface {
	IsListener() bool
}

// Acknowledger is optionally implemented by Sources that support
// deferred message acknowledgement (e.g. message queues).
type Acknowledger interface {
	AckLast() error
	NackLast() error
}

// Aliver is optionally implemented by Sources that can report
// whether their underlying connection is still alive.
type Aliver interface {
	IsAlive() bool
}

// SchemaIntrospector is optionally implemented by Sources/Targets
// that can report their schema (tables + columns).
type SchemaIntrospector interface {
	// ListTables returns available table names.
	ListTables() ([]string, error)
	// DescribeTable returns column info for the given table.
	DescribeTable(table string) (*TableSchema, error)
}
