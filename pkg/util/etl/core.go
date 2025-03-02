package etl

import (
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func Run(name string) {
	etl, err := CreateETL(name + ".yaml")

	if err != nil {
		log.Fatalf("Failed to load etl from config: %v", err)
	}

	etl.Run()
}
