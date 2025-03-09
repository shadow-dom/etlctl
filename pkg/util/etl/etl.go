package etl

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Query struct {
	Name string `yaml:"name"`
	SQL  string `yaml:"sql"`
}

type ETL struct {
	Sources   []DBStorage `yaml:"sources"`
	Targets   []DBStorage `yaml:"targets"`
	Pipelines []Pipeline  `yaml:"pipelines"`
	Queries   []Query     `yaml:"queries"`
	Name      string      `yaml:"name"`
}

func getQueryByName(name string, queries []Query) (string, error) {
	for _, query := range queries {
		if query.Name == name {
			return query.SQL, nil
		}
	}

	response := fmt.Sprintf("invalid data storage requested: %s", name)

	return "", errors.New(response)
}

func getDBStorage(name string, storages []DBStorage) (*DBStorage, error) {
	for _, storage := range storages {
		if storage.Name == name {
			return &storage, nil
		}
	}

	response := fmt.Sprintf("invalid data storage requested: %s", name)

	return nil, errors.New(response)
}

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

func CreateETL(filePath string) (*ETL, error) {
	data, err := os.ReadFile("../etls/" + filePath)

	if err != nil {
		return nil, err
	}

	var etl ETL
	if err := yaml.Unmarshal(data, &etl); err != nil {
		return nil, err
	}

	return &etl, nil
}

func (etl *ETL) GetSource(name string) (*DBStorage, error) {
	return getDBStorage(name, etl.Sources)
}

func (etl *ETL) GetTarget(name string) (*DBStorage, error) {
	return getDBStorage(name, etl.Targets)
}

func (etl *ETL) InjectSourceQueryWithState(source string, query string, pipeline Pipeline) string {
	trackingField, lastValue := pipeline.GetTrackingState(source)
	condition := fmt.Sprintf("%s > '%s'", trackingField, lastValue)
	whereRegex := regexp.MustCompile(`(?i)\bWHERE\b`)

	if whereRegex.MatchString(pipeline.Query) {
		return whereRegex.ReplaceAllString(pipeline.Query, "WHERE "+condition+" AND")
	}

	return query + " WHERE " + condition
}

func track(msg string) (string, time.Time) {
	return msg, time.Now()
}

func duration(msg string, start time.Time) {
	log.Printf("%v: %v\n", msg, time.Since(start))
}

func (etl *ETL) Deduplicate(uniqueFields []string, data []map[string]string) []map[string]string {
	defer duration(track("dedup"))

	observed := make(map[string]bool)

	var results []map[string]string

	var sb strings.Builder

	for _, record := range data {
		for _, field := range uniqueFields {
			if value, exists := record[field]; exists {
				sb.WriteString(value)
				sb.WriteString("|")
			}
		}

		if !observed[sb.String()] {
			observed[sb.String()] = true
			results = append(results, record)
		}

		sb.Reset()
	}

	return results
}

func updateQueryWithState(pipeline Pipeline, source string, query string) (string, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimSuffix(query, ";")

	if state, ok := pipeline.State.Sources[source]; ok {
		if state.Field == "" || state.LastValue == "" {
			return query, nil
		}

		condition := fmt.Sprintf("%s > '%s'", state.Field, state.LastValue)

		whereRegex := regexp.MustCompile(`(?i)\bWHERE\b`)
		if whereRegex.MatchString(query) {
			return whereRegex.ReplaceAllString(pipeline.Query, "WHERE "+condition+" AND"), nil
		}

		return query + " WHERE " + condition, nil
	}

	return query, fmt.Errorf("missing state: could not load state for source (%s)", source)
}

func (etl *ETL) Extract(sourceName string, pipeline *Pipeline) []map[string]string {
	err := pipeline.InitState()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Pulling data from %s...\n", sourceName)

	source, err := etl.GetSource(sourceName)

	if err != nil {
		log.Fatal(err)
	}

	query, err := getQueryByName(pipeline.Query, etl.Queries)

	if err != nil {
		log.Fatal(err)
	}

	query, err = updateQueryWithState(*pipeline, sourceName, query)

	fmt.Println(query)

	if err != nil {
		log.Fatal(err)
	}

	sourceDB, err := source.Connect()

	if err != nil {
		log.Fatalf("Failed to connect to source: %v", err)
	}

	defer sourceDB.Close()

	rows, err := sourceDB.Query(query)

	if err != nil {
		log.Fatal("Query execution failed:", err)
	}

	defer rows.Close()

	cols, err := rows.Columns()

	if err != nil {
		log.Fatal("Failed to get column names:", err)
	}

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

	fmt.Println("DONE!")

	return data
}

func (etl *ETL) Load(pipeline Pipeline, data []map[string]string) error {
	sourceFields, targetFields := pipeline.GetFields()
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
		return err
	}

	targetDB, err := target.Connect()

	if err != nil {
		return err
	}

	defer targetDB.Close()

	_, err = targetDB.Exec(insertSQL)

	if err != nil {
		return err
	}

	return nil
}

func (etl *ETL) Run() {
	for _, pipeline := range etl.Pipelines {
		data := make([]map[string]string, 0)

		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, sourceName := range pipeline.Sources {
			wg.Add(1)
			go func(source string) {
				defer wg.Done()

				result := etl.Extract(source, &pipeline)

				mu.Lock()

				if len(result) > 0 {
					pipeline.UpdateTrackingState(source, result[len(result)-1])
				}

				data = append(data, result...)
				mu.Unlock()
			}(sourceName)
		}

		wg.Wait()

		if len(pipeline.UniqueFields) > 0 && len(data) > 0 {
			etl.Load(pipeline, etl.Deduplicate(pipeline.UniqueFields, data))
		} else {
			etl.Load(pipeline, data)
		}

		pipeline.SaveState()
	}
}
