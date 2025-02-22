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

func extract() (interface{}, error) {
	return nil, nil
}

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

func getSQLColumns(fields []string) string {
	return strings.Join(fields, ", ")
}

func getDataStorageInfo(name string, storages []DBStorage) (*DBStorage, error) {
	for _, storage := range storages {
		if storage.Name == name {
			return &storage, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("invalid data storage requested: %s", name))
}

func Run(name string) {
	etl, err := readConfig(name + ".yaml")

	if err != nil {
		log.Fatalf("Failed to load etl from config: %v", err)
	}

	for _, pipeline := range etl.Pipelines {
		sourceFields, destinationFields := getFields(pipeline.Fields)

		sourceColumns := getSQLColumns(sourceFields)
		destinationColumns := getSQLColumns(destinationFields)

		fmt.Println(sourceColumns)
		fmt.Println(destinationColumns)

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

		fmt.Println(destination)

		sourceDB, err := connectToDatabase("../etls/data/" + source.Connection["filepath"])

		if err != nil {
			log.Fatalf("Failed to connect to destination: %v", err)
		}

		defer sourceDB.Close()

		fmt.Println(source.Query)

		if source.Query != "" {
			rows, err := sourceDB.Query(source.Query)

			if err != nil {
				log.Fatal("Query execution failed:", err)
			}
			defer rows.Close()

			cols, err := rows.Columns()

			if err != nil {
				log.Fatal("Failed to get column names:", err)
			}

			data := make(map[string]string)

			if rows.Next() {
				columnPointers := make([]interface{}, len(cols))
				columnValues := make([]sql.NullString, len(cols))

				for i := range columnPointers {
					columnPointers[i] = &columnValues[i]
				}

				if err := rows.Scan(columnPointers...); err != nil {
					log.Fatal("Failed to scan row:", err)
				}

				for i, colName := range cols {
					if columnValues[i].Valid {
						data[colName] = columnValues[i].String
					} else {
						data[colName] = ""
					}
				}
			}

			fmt.Println(data)
		}
	}

}
