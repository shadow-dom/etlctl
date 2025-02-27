package etl

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

func readConfig(path string) (*ETL, error) {
	data, err := os.ReadFile("../etls/" + path)

	if err != nil {
		return nil, err
	}

	var etl ETL
	if err := yaml.Unmarshal(data, &etl); err != nil {
		return nil, err
	}

	return &etl, nil
}

func connectToDatabase(file string) (*sql.DB, error) {
	return sql.Open("sqlite3", file)
}

func getFields(fields []FieldMapping) ([]string, []string) {
	sourceFields := make([]string, 0, len(fields))
	targetFields := make([]string, 0, len(fields))

	for _, field := range fields {
		sourceFields = append(sourceFields, field.Source)
		targetFields = append(targetFields, field.Target)
	}

	return sourceFields, targetFields
}

func getDataForColumns(fields []string, data []map[string]string) string {
	results := make([]string, 0, len(data))

	for _, row := range data {
		rowValues := make([]string, 0, len(fields))

		for _, field := range fields {
			rowValues = append(rowValues, fmt.Sprintf("\"%s\"", row[field]))
		}

		results = append(results, fmt.Sprintf("(%s)", strings.Join(rowValues, ", ")))
	}

	return strings.Join(results, ", ")
}

func getSQLColumns(fields []string) string {
	return strings.Join(fields, ", ")
}

func getDataStorageInfo(name string, storages []DBStorage) (*DBStorage, error) {
	for _, storage := range storages {
		if storage.Name == name {
			return &storage, nil
		}
	}

	response := fmt.Sprintf("invalid data storage requested: %s", name)

	return nil, errors.New(response)
}

func extract(rows *sql.Rows, cols []string) []map[string]string {
	data := []map[string]string{}

	for rows.Next() {
		columnValues := make([]sql.NullString, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			log.Fatal("Failed to scan row:", err)
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

	return data
}

func Run(name string) {
	etl, err := readConfig(name + ".yaml")

	if err != nil {
		log.Fatalf("Failed to load etl from config: %v", err)
	}

	for _, pipeline := range etl.Pipelines {
		sourceFields, destinationFields := getFields(pipeline.Fields)

		sourceInfo := strings.Split(pipeline.Source, ".")
		destinationInfo := strings.Split(pipeline.Target, ".")

		source, err := getDataStorageInfo(sourceInfo[0], etl.Sources)

		if err != nil {
			log.Fatal(err)
		}

		destination, err := getDataStorageInfo(destinationInfo[0], etl.Destinations)

		if err != nil {
			log.Fatal(err)
		}

		sourceDB, err := connectToDatabase("../etls/data/" + source.Connection["filepath"])

		if err != nil {
			log.Fatalf("Failed to connect to source: %v", err)
		}

		defer sourceDB.Close()

		var rows *sql.Rows

		if source.Query != "" {
			rows, err = sourceDB.Query(source.Query)

			if err != nil {
				log.Fatal("Query execution failed:", err)
			}
			defer rows.Close()
		} else {
			query := fmt.Sprintf("SELECT %s FROM %s;", getSQLColumns(sourceFields), sourceInfo[1])

			rows, err = sourceDB.Query(query)

			if err != nil {
				log.Fatal("Query execution failed:", err)
			}
			defer rows.Close()
		}

		cols, err := rows.Columns()

		if err != nil {
			log.Fatal("Failed to get column names:", err)
		}

		data := extract(rows, cols)

		// Prepare insert statement for destination
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
			destinationInfo[1],
			getSQLColumns(destinationFields),
			getDataForColumns(sourceFields, data),
		)

		destinationDB, err := connectToDatabase("../etls/data/" + destination.Connection["filepath"])

		if err != nil {
			log.Fatalf("Failed to connect to destination: %v", err)
		}

		defer destinationDB.Close()

		_, err = destinationDB.Exec(insertSQL)

		if err != nil {
			log.Fatalf("Failed to insert: %v", err)
		}
	}
}
