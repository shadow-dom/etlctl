package etl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FieldMapping struct {
	Source    string `yaml:"source"`
	Target    string `yaml:"target"`
	Transform string `yaml:"transform,omitempty"`
}

type SourceTracking struct {
	Field     string `yaml:"field"`
	LastValue string `yaml:"last_value"`
}

type State struct {
	Sources     map[string]SourceTracking `yaml:"sources"`
	StorageType string                    `yaml:"storage_type"`
}

type TrackingSpec struct {
	Field       string `yaml:"field"`
	StorageType string `yaml:"storage_type"`
}

type Pipeline struct {
	Sources      []string       `yaml:"sources"`
	Target       string         `yaml:"target"`
	Fields       []FieldMapping `yaml:"fields"`
	Query        string         `yaml:"query,omitempty"`
	TrackingSpec TrackingSpec   `yaml:"tracking"`
	Name         string         `yaml:"name"`
	State        State
}

func (pipeline *Pipeline) GetFields() ([]string, []string) {
	sourceFields := make([]string, 0, len(pipeline.Fields))
	targetFields := make([]string, 0, len(pipeline.Fields))

	for _, field := range pipeline.Fields {
		sourceFields = append(sourceFields, field.Source)
		targetFields = append(targetFields, field.Target)
	}

	return sourceFields, targetFields
}

func (pipeline *Pipeline) GetTargetInfo() (string, string) {
	info := strings.Split(pipeline.Target, ".")

	if len(info) > 1 {
		return info[0], info[1]
	}

	return info[0], ""
}

func (pipeline *Pipeline) GetState() error {
	file := filepath.Join("../etls", "state", pipeline.Name+"_state.yaml")

	if _, err := os.Stat(file); err == nil {
		data, err := os.ReadFile(file)

		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(data, &pipeline.State); err != nil {
			return err
		}
	} else {
		pipeline.State = State{
			StorageType: "file",
			Sources:     make(map[string]SourceTracking),
		}

		for _, source := range pipeline.Sources {
			pipeline.State.Sources[source] = SourceTracking{
				Field:     pipeline.TrackingSpec.Field,
				LastValue: "",
			}
		}
	}

	return nil
}

func (pipeline *Pipeline) SaveState() error {
	data, err := yaml.Marshal(pipeline.State)

	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join("../etls", "state", pipeline.Name+"_state.yaml"), data, 0644)
}

func (pipeline *Pipeline) GetTrackingState(source string) (string, string) {
	trackingField := "id"
	lastValue := ""

	if state, ok := pipeline.State.Sources[source]; ok {
		trackingField = state.Field
		lastValue = state.LastValue
	}

	return trackingField, lastValue
}

func (pipeline *Pipeline) UpdateTrackingState(source string, lastRecord map[string]string) {
	trackingField, _ := pipeline.GetTrackingState(source)

	fmt.Println((lastRecord))

	if lastVal, exists := lastRecord[trackingField]; exists {
		pipeline.State.Sources[source] = SourceTracking{
			Field:     trackingField,
			LastValue: lastVal,
		}
	}
}

func (pipeline *Pipeline) UpdateQueryWithState() {}
