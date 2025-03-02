package etl

import (
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

func Run(name string) {
	etl, err := CreateETL(name + ".yaml")

	if err != nil {
		log.Fatalf("Failed to load etl from config: %v", err)
	}

	for _, pipeline := range etl.Pipelines {
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

		etl.Load(pipeline, data)
	}
}
