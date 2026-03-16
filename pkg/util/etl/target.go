package etl

// Target loads data into an external system.
type Target interface {
	// Name returns the unique identifier for this target.
	Name() string
	// Type returns the target type key.
	Type() string
	// Connect initializes the target.
	Connect() error
	// Load writes rows to the target. tableName is used for DB targets;
	// file/API targets may ignore it.
	Load(tableName string, fields []FieldMapping, data []map[string]string) error
	// Close releases any resources.
	Close() error
}
