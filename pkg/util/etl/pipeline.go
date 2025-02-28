package etl

import "strings"

type FieldMapping struct {
	Source    string `yaml:"source"`
	Target    string `yaml:"target"`
	Transform string `yaml:"transform,omitempty"`
}

type Pipeline struct {
	Sources []string       `yaml:"sources"`
	Target  string         `yaml:"target"`
	Fields  []FieldMapping `yaml:"fields"`
	Query   string         `yaml:"query,omitempty"`
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
