package etl

import (
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

// IdentityMappings builds a pass-through mapping for every key in the row.
func IdentityMappings(row map[string]string) []FieldMapping {
	fields := make([]FieldMapping, 0, len(row))
	for k := range row {
		fields = append(fields, FieldMapping{Source: k, Target: k})
	}
	return fields
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
	Targets      []string       `yaml:"targets,omitempty"`
	Fields       []FieldMapping `yaml:"fields"`
	Query        string         `yaml:"query,omitempty"`
	Functions    []string       `yaml:"functions,omitempty"`
	TrackingSpec TrackingSpec   `yaml:"tracking"`
	Name         string         `yaml:"name"`
	UniqueFields []string       `yaml:"uniqueFields,omitempty"`
	KeepLast     bool            `yaml:"keepLast,omitempty"`
	State        State
	ConfigDir    string `yaml:"-"`
}

// GetAllTargets returns all targets for this pipeline (supports both single Target and multi Targets).
func (pipeline *Pipeline) GetAllTargets() []string {
	if len(pipeline.Targets) > 0 {
		return pipeline.Targets
	}
	if pipeline.Target != "" {
		return []string{pipeline.Target}
	}
	return nil
}

func (pipeline *Pipeline) stateDir() string {
	if pipeline.ConfigDir != "" {
		return filepath.Join(pipeline.ConfigDir, "state")
	}
	return filepath.Join("../etls", "state")
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

func (pipeline *Pipeline) InitState() error {
	file := filepath.Join(pipeline.stateDir(), pipeline.Name+"_state.yaml")

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

	dir := pipeline.stateDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, pipeline.Name+"_state.yaml"), data, 0644)
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

	if pipeline.State.Sources == nil {
		pipeline.State.Sources = make(map[string]SourceTracking)
	}

	if lastVal, exists := lastRecord[trackingField]; exists {
		pipeline.State.Sources[source] = SourceTracking{
			Field:     trackingField,
			LastValue: lastVal,
		}
	}
}
