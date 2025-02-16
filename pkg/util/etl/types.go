package etl

type DBStorage struct {
	Name       string            `yaml:"name"`
	Type       string            `yaml:"type"`
	Connection map[string]string `yaml:"connection"`
	Query      string            `yaml:"query"`
}

type Transformation struct {
	Source string         `yaml:"source"`
	Target string         `yaml:"target"`
	Fields []FieldMapping `yaml:"fields"`
}

type FieldMapping struct {
	Source    string `yaml:"source"`
	Target    string `yaml:"target"`
	Transform string `yaml:"transform,omitempty"`
}

type ETL struct {
	Sources         []DBStorage      `yaml:"sources"`
	Destinations    []DBStorage      `yaml:"destinations"`
	Transformations []Transformation `yaml:"transformations"`
}
