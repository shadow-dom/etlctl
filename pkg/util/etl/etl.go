package etl

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ETL struct {
	Sources   []DBStorage `yaml:"sources"`
	Targets   []DBStorage `yaml:"targets"`
	Pipelines []Pipeline  `yaml:"pipelines"`
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
