package etl

import (
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func Run(configDir string, name string) error {
	etl, err := CreateETL(configDir, name)
	if err != nil {
		return err
	}

	return etl.Run()
}

// Listen starts the named ETL in listener mode for event-driven sources.
func Listen(configDir string, name string) error {
	etl, err := CreateETL(configDir, name)
	if err != nil {
		return err
	}

	return etl.Listen()
}

// RunSimple is a convenience wrapper that uses the default config directory.
// Kept for backward compatibility.
func RunSimple(name string) {
	if err := Run("../etls", name); err != nil {
		log.Fatalf("Failed to run ETL: %v", err)
	}
}
