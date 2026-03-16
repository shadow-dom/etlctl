package etl

import (
	"fmt"
	"strings"
)

// ValidateETL checks an ETL config for structural correctness.
func ValidateETL(e *ETL) []string {
	var errs []string

	if e.Name == "" {
		errs = append(errs, "missing ETL name")
	}

	if len(e.Sources) == 0 {
		errs = append(errs, "no sources defined")
	}

	if len(e.Targets) == 0 {
		errs = append(errs, "no targets defined")
	}

	if len(e.Pipelines) == 0 {
		errs = append(errs, "no pipelines defined")
	}

	sourceNames := make(map[string]bool)
	for _, s := range e.Sources {
		if s.Name == "" {
			errs = append(errs, "source missing name")
		}
		if s.Type == "" {
			errs = append(errs, fmt.Sprintf("source %q missing type", s.Name))
		}
		sourceNames[s.Name] = true
	}

	targetNames := make(map[string]bool)
	for _, t := range e.Targets {
		if t.Name == "" {
			errs = append(errs, "target missing name")
		}
		if t.Type == "" {
			errs = append(errs, fmt.Sprintf("target %q missing type", t.Name))
		}
		targetNames[t.Name] = true
	}

	queryNames := make(map[string]bool)
	for _, q := range e.Queries {
		if q.Name == "" {
			errs = append(errs, "query missing name")
		}
		if q.SQL == "" {
			errs = append(errs, fmt.Sprintf("query %q has empty SQL", q.Name))
		}
		queryNames[q.Name] = true
	}

	functionNames := make(map[string]bool)
	for _, f := range e.Functions {
		if f.Name == "" {
			errs = append(errs, "function missing name")
		}
		if f.Code == "" {
			errs = append(errs, fmt.Sprintf("function %q has empty code", f.Name))
		}
		functionNames[f.Name] = true
	}

	for i, p := range e.Pipelines {
		prefix := fmt.Sprintf("pipeline[%d]", i)
		if p.Name == "" {
			errs = append(errs, prefix+": missing name")
		}

		if len(p.Sources) == 0 {
			errs = append(errs, prefix+": no sources")
		}
		for _, s := range p.Sources {
			if !sourceNames[s] {
				errs = append(errs, fmt.Sprintf("%s: source %q not defined", prefix, s))
			}
		}

		allTargets := p.GetAllTargets()
		if len(allTargets) == 0 {
			errs = append(errs, prefix+": no target")
		}
		for _, t := range allTargets {
			targetName := strings.Split(t, ".")[0]
			if !targetNames[targetName] {
				errs = append(errs, fmt.Sprintf("%s: target %q not defined", prefix, targetName))
			}
		}

		if p.Query != "" && !queryNames[p.Query] {
			errs = append(errs, fmt.Sprintf("%s: query %q not defined", prefix, p.Query))
		}

		for _, fn := range p.Functions {
			if !functionNames[fn] {
				errs = append(errs, fmt.Sprintf("%s: function %q not defined", prefix, fn))
			}
		}

		// Empty fields is allowed — targets will pass through all columns
	}

	return errs
}
