package etl

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

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
	etl, err := CreateETL(name + ".yaml")

	if err != nil {
		log.Fatalf("Failed to load etl from config: %v", err)
	}

	for _, pipeline := range etl.Pipelines {
		sourceFields, targetFields := pipeline.GetFields()

		sourceName, sourceTable := pipeline.GetSourceInfo()
		targetName, targetTable := pipeline.GetTargetInfo()

		source, err := etl.GetSource(sourceName)

		if err != nil {
			log.Fatal(err)
		}

		target, err := etl.GetTarget(targetName)

		if err != nil {
			log.Fatal(err)
		}

		sourceDB, err := source.Connect()

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
			query := fmt.Sprintf("SELECT %s FROM %s;", strings.Join(sourceFields, ", "), sourceTable)

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

		// Prepare insert statement for target
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
			targetTable,
			strings.Join(targetFields, ", "),
			getDataForColumns(sourceFields, data),
		)

		targetDB, err := target.Connect()

		if err != nil {
			log.Fatalf("Failed to connect to target: %v", err)
		}

		defer targetDB.Close()

		_, err = targetDB.Exec(insertSQL)

		if err != nil {
			log.Fatalf("Failed to insert: %v", err)
		}
	}
}
