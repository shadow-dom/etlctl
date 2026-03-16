package transform

import (
	"fmt"
	"strings"
)

// TransformFunc takes a string value and returns the transformed value.
type TransformFunc func(value string) (string, error)

// ParamTransformFunc takes a value and a parameter string.
type ParamTransformFunc func(value string, param string) (string, error)

var registry = map[string]TransformFunc{}
var paramRegistry = map[string]ParamTransformFunc{}

// Register adds a simple (no-parameter) transform.
func Register(name string, fn TransformFunc) {
	registry[name] = fn
}

// RegisterParameterized adds a parameterized transform (e.g., "round:2").
func RegisterParameterized(name string, fn ParamTransformFunc) {
	paramRegistry[name] = fn
}

// Apply executes a transform expression on a value.
// Expressions can be chained with "|": "trim|uppercase"
// Parameters use ":": "round:2", "replace:old->new", "default:N/A"
func Apply(expression string, value string) (string, error) {
	steps := strings.Split(expression, "|")

	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}

		name, param := parseStep(step)

		// Try simple transform first
		if fn, ok := registry[name]; ok {
			var err error
			value, err = fn(value)
			if err != nil {
				return "", fmt.Errorf("transform %q failed: %w", name, err)
			}
			continue
		}

		// Try parameterized transform
		if fn, ok := paramRegistry[name]; ok {
			var err error
			value, err = fn(value, param)
			if err != nil {
				return "", fmt.Errorf("transform %q failed: %w", name, err)
			}
			continue
		}

		return "", fmt.Errorf("unknown transform: %s", name)
	}

	return value, nil
}

// RegisteredTransforms returns all simple (no-parameter) transform names.
func RegisteredTransforms() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// RegisteredParamTransforms returns all parameterized transform names.
func RegisteredParamTransforms() []string {
	names := make([]string, 0, len(paramRegistry))
	for name := range paramRegistry {
		names = append(names, name)
	}
	return names
}

func parseStep(step string) (string, string) {
	parts := strings.SplitN(step, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}
