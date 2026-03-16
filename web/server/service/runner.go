package service

import (
	"time"

	"shadow-dom/etlctl/pkg/util/etl"
	"shadow-dom/etlctl/web/server/api"
)

const maxSampleRows = 50

type RunnerService struct {
	ConfigDir string
}

func NewRunnerService(configDir string) *RunnerService {
	return &RunnerService{ConfigDir: configDir}
}

// TestETL runs an ETL step-by-step, capturing data at each stage.
// When resetState is true, incremental state tracking is ignored so all rows are returned.
func (r *RunnerService) TestETL(name string, dryRun bool, resetState bool) (*api.TestResponse, error) {
	e, err := etl.CreateETL(r.ConfigDir, name)
	if err != nil {
		return nil, err
	}
	e.ConfigDir = r.ConfigDir

	resp := &api.TestResponse{Success: true}

	// Cache extracted data so shared sources aren't re-extracted across pipelines
	extractCache := make(map[string][]map[string]string)

	for _, pipeline := range e.Pipelines {
		pipeline.ConfigDir = r.ConfigDir

		// When resetState is true, clear tracking so we get all rows
		if resetState {
			pipeline.TrackingSpec = etl.TrackingSpec{}
		}

		allData := make([]map[string]string, 0)

		// Extract from each source (reuse cached data if available)
		for _, sourceName := range pipeline.Sources {
			cacheKey := sourceName + "|" + pipeline.Query
			if cached, ok := extractCache[cacheKey]; ok {
				// Reuse previously extracted data
				step := api.TestStepResult{
					Step:     "extract",
					Node:     sourceName + " (cached)",
					RowCount: len(cached),
					Sample:   sampleRows(cached, maxSampleRows),
				}
				resp.Steps = append(resp.Steps, step)
				allData = append(allData, cached...)
				continue
			}

			start := time.Now()
			data, err := e.Extract(sourceName, &pipeline)
			elapsed := time.Since(start).Milliseconds()

			step := api.TestStepResult{
				Step:       "extract",
				Node:       sourceName,
				DurationMs: elapsed,
			}

			if err != nil {
				step.Error = err.Error()
				resp.Success = false
			} else {
				step.RowCount = len(data)
				step.Sample = sampleRows(data, maxSampleRows)
				allData = append(allData, data...)
				extractCache[cacheKey] = data
			}

			resp.Steps = append(resp.Steps, step)
		}

		// Dedup
		if len(pipeline.UniqueFields) > 0 && len(allData) > 0 {
			start := time.Now()
			before := len(allData)
			allData = e.Deduplicate(pipeline.UniqueFields, allData, pipeline.KeepLast)
			elapsed := time.Since(start).Milliseconds()

			dedupStep := api.TestStepResult{
				Step:       "dedup",
				Node:       pipeline.Name,
				RowCount:   len(allData),
				Sample:     sampleRows(allData, maxSampleRows),
				DurationMs: elapsed,
			}
			if removed := before - len(allData); removed > 0 {
				dedupStep.Error = ""
			}

			resp.Steps = append(resp.Steps, dedupStep)
		}

		// Transform / field mapping
		if len(pipeline.Fields) > 0 && len(allData) > 0 {
			hasTransforms := false
			for _, f := range pipeline.Fields {
				if f.Transform != "" {
					hasTransforms = true
					break
				}
			}

			if hasTransforms {
				start := time.Now()
				transformed, err := e.ApplyTransforms(pipeline, allData)
				elapsed := time.Since(start).Milliseconds()

				step := api.TestStepResult{
					Step:       "transform",
					Node:       pipeline.Name,
					DurationMs: elapsed,
				}

				if err != nil {
					step.Error = err.Error()
					resp.Success = false
				} else {
					allData = transformed
					step.RowCount = len(allData)
					step.Sample = sampleRows(allData, maxSampleRows)
				}

				resp.Steps = append(resp.Steps, step)
			} else {
				// Show field mapping step even without transform expressions
				resp.Steps = append(resp.Steps, api.TestStepResult{
					Step:     "field_mapping",
					Node:     pipeline.Name,
					RowCount: len(allData),
					Sample:   sampleRows(allData, maxSampleRows),
				})
			}
		}

		// Load into all targets
		for _, targetRef := range pipeline.GetAllTargets() {
			step := api.TestStepResult{
				Step:     "load",
				Node:     targetRef,
				RowCount: len(allData),
			}

			if dryRun {
				step.Sample = sampleRows(allData, maxSampleRows)
			} else if len(allData) > 0 {
				// Create a temporary single-target pipeline for loading
				loadPipeline := pipeline
				loadPipeline.Target = targetRef
				start := time.Now()
				err := e.Load(loadPipeline, allData)
				step.DurationMs = time.Since(start).Milliseconds()
				if err != nil {
					step.Error = err.Error()
					resp.Success = false
				}
			}

			resp.Steps = append(resp.Steps, step)
		}
	}

	return resp, nil
}

func sampleRows(data []map[string]string, max int) []map[string]string {
	if len(data) <= max {
		return data
	}
	return data[:max]
}
