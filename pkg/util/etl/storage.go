package etl

// StorageConfig is the YAML-level representation for sources and targets.
// It is type-agnostic; the registry resolves it to a Source or Target at runtime.
type StorageConfig struct {
	Name          string            `yaml:"name"`
	Type          string            `yaml:"type"`
	Connection    map[string]string `yaml:"connection,omitempty"`
	ConnectionRef string           `yaml:"connection_ref,omitempty" json:"connection_ref,omitempty"`
}
