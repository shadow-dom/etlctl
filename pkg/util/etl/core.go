package etl

import (
	"os"

	"gopkg.in/yaml.v3"
)

func ReadConfig(path string) ETL {
	data, err := os.ReadFile(path)

	if err != nil {
		panic(err)
	}

	var etl ETL
	if err := yaml.Unmarshal(data, &etl); err != nil {
		panic(err)
	}

	return etl
}
