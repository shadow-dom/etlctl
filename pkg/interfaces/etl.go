package interfaces

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
	Name         string      `yaml:"name"`
	Sources      []DBStorage `yaml:"sources"`
	Destinations []DBStorage `yaml:"destinations"`
}

type Pipeline struct {
	Name string `yaml:"name"`
	Jobs []ETL  `yaml:"jobs"`
}
