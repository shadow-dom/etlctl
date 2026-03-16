package etl

import "strings"

// AutoMapResult contains the result of auto-mapping source to target columns.
type AutoMapResult struct {
	Fields          []FieldMapping `json:"fields"`
	UnmatchedSource []string       `json:"unmatched_source"`
	UnmatchedTarget []string       `json:"unmatched_target"`
}

// normalize strips underscores, hyphens, and spaces, then lowercases.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

// AutoMap matches source columns to target columns by name.
// Priority: 1) exact case-insensitive match, 2) normalized match.
func AutoMap(source, target *TableSchema) *AutoMapResult {
	result := &AutoMapResult{}

	// Build lookup maps for target columns
	targetExact := make(map[string]string)    // lowercase -> original name
	targetNorm := make(map[string]string)     // normalized -> original name
	targetMatched := make(map[string]bool)

	for _, col := range target.Columns {
		targetExact[strings.ToLower(col.Name)] = col.Name
		targetNorm[normalize(col.Name)] = col.Name
	}

	for _, srcCol := range source.Columns {
		matched := false

		// 1. Exact case-insensitive match
		if tgtName, ok := targetExact[strings.ToLower(srcCol.Name)]; ok && !targetMatched[tgtName] {
			result.Fields = append(result.Fields, FieldMapping{
				Source: srcCol.Name,
				Target: tgtName,
			})
			targetMatched[tgtName] = true
			matched = true
		}

		// 2. Normalized match
		if !matched {
			if tgtName, ok := targetNorm[normalize(srcCol.Name)]; ok && !targetMatched[tgtName] {
				result.Fields = append(result.Fields, FieldMapping{
					Source: srcCol.Name,
					Target: tgtName,
				})
				targetMatched[tgtName] = true
				matched = true
			}
		}

		if !matched {
			result.UnmatchedSource = append(result.UnmatchedSource, srcCol.Name)
		}
	}

	for _, col := range target.Columns {
		if !targetMatched[col.Name] {
			result.UnmatchedTarget = append(result.UnmatchedTarget, col.Name)
		}
	}

	return result
}
