package etl

import "fmt"

// SourceFactory creates a Source from a name and connection config.
type SourceFactory func(name string, config map[string]string) (Source, error)

// TargetFactory creates a Target from a name and connection config.
type TargetFactory func(name string, config map[string]string) (Target, error)

var sourceRegistry = map[string]SourceFactory{}
var targetRegistry = map[string]TargetFactory{}

// RegisterSource registers a source factory for a given type name.
func RegisterSource(typeName string, factory SourceFactory) {
	sourceRegistry[typeName] = factory
}

// RegisterTarget registers a target factory for a given type name.
func RegisterTarget(typeName string, factory TargetFactory) {
	targetRegistry[typeName] = factory
}

// RegisteredSourceTypes returns all registered source type names.
func RegisteredSourceTypes() []string {
	names := make([]string, 0, len(sourceRegistry))
	for name := range sourceRegistry {
		names = append(names, name)
	}
	return names
}

// RegisteredTargetTypes returns all registered target type names.
func RegisteredTargetTypes() []string {
	names := make([]string, 0, len(targetRegistry))
	for name := range targetRegistry {
		names = append(names, name)
	}
	return names
}

// NewSource creates a Source instance from the registry.
func NewSource(typeName, name string, config map[string]string) (Source, error) {
	factory, ok := sourceRegistry[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown source type: %s", typeName)
	}
	return factory(name, config)
}

// NewTarget creates a Target instance from the registry.
func NewTarget(typeName, name string, config map[string]string) (Target, error) {
	factory, ok := targetRegistry[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown target type: %s", typeName)
	}
	return factory(name, config)
}
