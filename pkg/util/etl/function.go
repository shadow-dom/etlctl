package etl

import "fmt"

// FunctionConfig defines a named Go function step in the ETL YAML.
// The Code field contains a Go function body that receives
// `rows []map[string]string` and must return `[]map[string]string, error`.
type FunctionConfig struct {
	Name string `yaml:"name" json:"name"`
	Code string `yaml:"code" json:"code"`
}

// PipelineFunc is a compiled function that processes rows in a pipeline.
type PipelineFunc func(data []map[string]string) ([]map[string]string, error)

var functionRegistry = map[string]PipelineFunc{}

// RegisterFunction registers a named pipeline function.
// This is used to make Go functions available at runtime — either built-in
// or compiled from user code during the build/generate step.
func RegisterFunction(name string, fn PipelineFunc) {
	functionRegistry[name] = fn
}

// RegisteredFunctions returns all registered function names.
func RegisteredFunctions() []string {
	names := make([]string, 0, len(functionRegistry))
	for name := range functionRegistry {
		names = append(names, name)
	}
	return names
}

// RunRegisteredFunction executes a previously registered function by name.
func RunRegisteredFunction(name string, data []map[string]string) ([]map[string]string, error) {
	fn, ok := functionRegistry[name]
	if !ok {
		return nil, fmt.Errorf("function %q is not registered — it must be compiled and registered before use", name)
	}
	return fn(data)
}
