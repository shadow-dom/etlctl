package etl

import (
	"database/sql"
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/microsoft/go-mssqldb"
)

// DBStorage is kept for backward compatibility with the template/generated code.
type DBStorage = StorageConfig

func expandPath(path string) (string, error) {
	if len(path) >= 2 && path[:2] == "~/" {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[2:]), nil
	}
	return path, nil
}

func getDSN(dbType string, connection map[string]string) (string, error) {
	switch dbType {
	case "sqlserver":
		u, ok := connection["user"]
		if ok {
			return fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s",
				u, connection["password"],
				connection["host"], connection["port"], connection["database"]), nil
		}
		return fmt.Sprintf("sqlserver://@%s:%s?database=%s&trusted_connection=yes",
			connection["host"], connection["port"], connection["database"]), nil
	case "sqlite3":
		path, err := expandPath(connection["filepath"])
		if err != nil {
			return "", fmt.Errorf("error expanding file path: %w", err)
		}
		return path, nil
	case "postgres":
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			connection["host"], connection["port"],
			connection["user"], connection["password"],
			connection["database"], connection["sslmode"]), nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// DBSource implements the Source interface for SQL databases.
type DBSource struct {
	name       string
	dbType     string
	connection map[string]string
	db         *sql.DB
}

func (s *DBSource) Name() string { return s.name }
func (s *DBSource) Type() string { return s.dbType }

func (s *DBSource) Connect() error {
	dsn, err := getDSN(s.dbType, s.connection)
	if err != nil {
		return err
	}
	db, err := sql.Open(s.dbType, dsn)
	if err != nil {
		return err
	}
	s.db = db
	return nil
}

func (s *DBSource) Extract(query string) ([]map[string]string, error) {
	if s.db == nil {
		return nil, fmt.Errorf("source %s not connected", s.name)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	var data []map[string]string

	for rows.Next() {
		columnValues := make([]sql.NullString, len(cols))
		columnPointers := make([]any, len(cols))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowData := make(map[string]string)
		for i, colName := range cols {
			if columnValues[i].Valid {
				rowData[colName] = columnValues[i].String
			} else {
				rowData[colName] = ""
			}
		}

		data = append(data, rowData)
	}

	return data, nil
}

func (s *DBSource) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *DBSource) ListTables() ([]string, error) {
	if s.db == nil {
		return nil, fmt.Errorf("source %s not connected", s.name)
	}
	return dbListTables(s.db, s.dbType)
}

func (s *DBSource) DescribeTable(table string) (*TableSchema, error) {
	if s.db == nil {
		return nil, fmt.Errorf("source %s not connected", s.name)
	}
	return dbDescribeTable(s.db, s.dbType, table)
}

// DBTarget implements the Target interface for SQL databases.
type DBTarget struct {
	name       string
	dbType     string
	connection map[string]string
	db         *sql.DB
}

func (t *DBTarget) Name() string { return t.name }
func (t *DBTarget) Type() string { return t.dbType }

func (t *DBTarget) Connect() error {
	dsn, err := getDSN(t.dbType, t.connection)
	if err != nil {
		return err
	}
	db, err := sql.Open(t.dbType, dsn)
	if err != nil {
		return err
	}
	t.db = db
	return nil
}

func (t *DBTarget) Load(tableName string, fields []FieldMapping, data []map[string]string) error {
	if t.db == nil {
		return fmt.Errorf("target %s not connected", t.name)
	}

	if tableName == "" {
		// Default to target name as table name
		tableName = t.name
	}

	if len(data) == 0 {
		return nil
	}

	// If no field mappings, pass through all columns
	if len(fields) == 0 {
		fields = IdentityMappings(data[0])
	}

	sourceFields := make([]string, len(fields))
	targetFields := make([]string, len(fields))
	for i, f := range fields {
		sourceFields[i] = f.Source
		targetFields[i] = f.Target
	}

	// Auto-create table if it doesn't exist
	colDefs := make([]string, len(targetFields))
	for i, col := range targetFields {
		colDefs[i] = col + " TEXT"
	}
	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(colDefs, ", "))
	if _, err := t.db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(targetFields, ", "),
		getDataForColumns(sourceFields, data),
	)

	_, err := t.db.Exec(insertSQL)
	return err
}

func (t *DBTarget) Close() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}

func (t *DBTarget) ListTables() ([]string, error) {
	if t.db == nil {
		return nil, fmt.Errorf("target %s not connected", t.name)
	}
	return dbListTables(t.db, t.dbType)
}

func (t *DBTarget) DescribeTable(table string) (*TableSchema, error) {
	if t.db == nil {
		return nil, fmt.Errorf("target %s not connected", t.name)
	}
	return dbDescribeTable(t.db, t.dbType, table)
}

func newDBSource(name string, config map[string]string) (Source, error) {
	return &DBSource{name: name, connection: config}, nil
}

func newDBTarget(name string, config map[string]string) (Target, error) {
	return &DBTarget{name: name, connection: config}, nil
}

func dbListTables(db *sql.DB, dbType string) ([]string, error) {
	var query string
	switch dbType {
	case "sqlite3":
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	case "postgres":
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name"
	case "sqlserver":
		query = "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"
	default:
		return nil, fmt.Errorf("unsupported database type for introspection: %s", dbType)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func dbDescribeTable(db *sql.DB, dbType, table string) (*TableSchema, error) {
	if table == "" {
		return &TableSchema{Table: ""}, nil
	}

	var query string
	switch dbType {
	case "sqlite3":
		query = fmt.Sprintf("PRAGMA table_info(%s)", table)
	case "postgres":
		query = fmt.Sprintf("SELECT column_name, data_type, CASE WHEN is_nullable = 'YES' THEN 'true' ELSE 'false' END FROM information_schema.columns WHERE table_schema = 'public' AND table_name = '%s' ORDER BY ordinal_position", table)
	case "sqlserver":
		query = fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE, CASE WHEN IS_NULLABLE = 'YES' THEN 'true' ELSE 'false' END FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = '%s' ORDER BY ORDINAL_POSITION", table)
	default:
		return nil, fmt.Errorf("unsupported database type for introspection: %s", dbType)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	schema := &TableSchema{Table: table}

	if dbType == "sqlite3" {
		// PRAGMA table_info returns: cid, name, type, notnull, dflt_value, pk
		for rows.Next() {
			var cid int
			var name, colType string
			var notnull int
			var dfltValue sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err != nil {
				return nil, err
			}
			schema.Columns = append(schema.Columns, ColumnInfo{
				Name:     name,
				DataType: colType,
				Nullable: notnull == 0,
			})
		}
	} else {
		for rows.Next() {
			var name, dataType, nullable string
			if err := rows.Scan(&name, &dataType, &nullable); err != nil {
				return nil, err
			}
			schema.Columns = append(schema.Columns, ColumnInfo{
				Name:     name,
				DataType: dataType,
				Nullable: nullable == "true",
			})
		}
	}

	return schema, nil
}

func init() {
	for _, t := range []string{"sqlite3", "sqlserver", "postgres"} {
		RegisterSource(t, func(name string, config map[string]string) (Source, error) {
			return &DBSource{name: name, dbType: t, connection: config}, nil
		})
		RegisterTarget(t, func(name string, config map[string]string) (Target, error) {
			return &DBTarget{name: name, dbType: t, connection: config}, nil
		})
	}
}
