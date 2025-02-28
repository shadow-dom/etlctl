package etl

import (
	"fmt"
	"log"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

func getDataForColumns(fields []string, data []map[string]string) string {
	results := make([]string, 0, len(data))

	for _, row := range data {
		rowValues := make([]string, 0, len(fields))

		for _, field := range fields {
			rowValues = append(rowValues, fmt.Sprintf("'%s'", row[field]))
		}

		results = append(results, fmt.Sprintf("(%s)", strings.Join(rowValues, ", ")))
	}

	return strings.Join(results, ", ")
}

func Run(name string) {
	etl, err := CreateETL(name + ".yaml")

	if err != nil {
		log.Fatalf("Failed to load etl from config: %v", err)
	}

	for _, pipeline := range etl.Pipelines {
		sourceFields, targetFields := pipeline.GetFields()

		data := make([]map[string]string, 0)

		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, sourceName := range pipeline.Sources {
			wg.Add(1)
			go func(source string) {
				defer wg.Done()

				result := etl.Extract(source, pipeline.Query)

				mu.Lock()
				data = append(data, result...)
				mu.Unlock()
			}(sourceName)
		}

		wg.Wait()

		targetName, targetTable := pipeline.GetTargetInfo()

		fmt.Printf("Writing data to target %s...\n", targetName)

		// Prepare insert statement for target
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
			targetTable,
			strings.Join(targetFields, ", "),
			getDataForColumns(sourceFields, data),
		)

		target, err := etl.GetTarget(targetName)

		if err != nil {
			log.Fatal(err)
		}

		targetDB, err := target.Connect()

		if err != nil {
			log.Fatalf("Failed to connect to target: %v", err)
		}

		defer targetDB.Close()

		_, err = targetDB.Exec(insertSQL)

		if err != nil {
			log.Fatalf("Failed to insert: %v", err)
		}

		fmt.Println("DONE!")
	}
}
