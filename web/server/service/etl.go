package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"shadow-dom/etlctl/pkg/util/etl"
	"shadow-dom/etlctl/web/server/api"

	"gopkg.in/yaml.v3"
)

type ETLService struct {
	ConfigDir string
}

func NewETLService(configDir string) *ETLService {
	return &ETLService{ConfigDir: configDir}
}

func (s *ETLService) List() ([]api.ETLResponse, error) {
	matches, err := filepath.Glob(filepath.Join(s.ConfigDir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	var results []api.ETLResponse
	for _, match := range matches {
		name := strings.TrimSuffix(filepath.Base(match), ".yaml")
		e, err := etl.CreateETL(s.ConfigDir, name)
		if err != nil {
			continue
		}
		results = append(results, toAPIResponse(e))
	}

	return results, nil
}

func (s *ETLService) Get(name string) (*api.ETLResponse, error) {
	e, err := etl.CreateETL(s.ConfigDir, name)
	if err != nil {
		return nil, err
	}
	resp := toAPIResponse(e)
	return &resp, nil
}

func (s *ETLService) Create(req api.ETLResponse) error {
	path := filepath.Join(s.ConfigDir, req.Name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("ETL %q already exists", req.Name)
	}
	return s.save(req)
}

func (s *ETLService) Update(name string, req api.ETLResponse) error {
	req.Name = name
	return s.save(req)
}

func (s *ETLService) Delete(name string) error {
	path := filepath.Join(s.ConfigDir, name+".yaml")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete ETL %q: %w", name, err)
	}
	// Clean up legacy DAG JSON if it exists
	legacyDag := filepath.Join(s.ConfigDir, name+"_dag.json")
	os.Remove(legacyDag)
	return nil
}

func (s *ETLService) Validate(name string) *api.ValidateResponse {
	e, err := etl.CreateETL(s.ConfigDir, name)
	if err != nil {
		return &api.ValidateResponse{
			Valid:  false,
			Errors: []string{err.Error()},
		}
	}

	errs := etl.ValidateETL(e)
	return &api.ValidateResponse{
		Valid:  len(errs) == 0,
		Errors: errs,
	}
}

func (s *ETLService) save(req api.ETLResponse) error {
	e := fromAPIRequest(req)

	// Preserve existing DAG layout from the file if present
	path := filepath.Join(s.ConfigDir, req.Name+".yaml")
	if existing, err := os.ReadFile(path); err == nil {
		var old etl.ETL
		if err := yaml.Unmarshal(existing, &old); err == nil && old.Dag != nil {
			e.Dag = old.Dag
		}
	}

	data, err := yaml.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal ETL: %w", err)
	}

	if err := os.MkdirAll(s.ConfigDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func toAPIResponse(e *etl.ETL) api.ETLResponse {
	resp := api.ETLResponse{
		Name:      e.Name,
		Sources:   []api.StorageConfig{},
		Targets:   []api.StorageConfig{},
		Pipelines: []api.PipelineConfig{},
		Queries:   []api.QueryConfig{},
	}

	for _, src := range e.Sources {
		resp.Sources = append(resp.Sources, api.StorageConfig{
			Name:       src.Name,
			Type:       src.Type,
			Connection: src.Connection,
		})
	}

	for _, tgt := range e.Targets {
		resp.Targets = append(resp.Targets, api.StorageConfig{
			Name:       tgt.Name,
			Type:       tgt.Type,
			Connection: tgt.Connection,
		})
	}

	for _, q := range e.Queries {
		resp.Queries = append(resp.Queries, api.QueryConfig{
			Name: q.Name,
			SQL:  q.SQL,
		})
	}

	for _, fn := range e.Functions {
		resp.Functions = append(resp.Functions, api.FunctionConfig{
			Name: fn.Name,
			Code: fn.Code,
		})
	}

	for _, p := range e.Pipelines {
		pc := api.PipelineConfig{
			Name:         p.Name,
			Sources:      p.Sources,
			Targets:      p.GetAllTargets(),
			Query:        p.Query,
			Functions:    p.Functions,
			UniqueFields: p.UniqueFields,
			KeepLast:     p.KeepLast,
		}
		if p.TrackingSpec.Field != "" {
			pc.Tracking = &api.TrackingSpec{
				Field:       p.TrackingSpec.Field,
				StorageType: p.TrackingSpec.StorageType,
			}
		}
		for _, f := range p.Fields {
			pc.Fields = append(pc.Fields, api.FieldMapping{
				Source:    f.Source,
				Target:    f.Target,
				Transform: f.Transform,
			})
		}
		resp.Pipelines = append(resp.Pipelines, pc)
	}

	if e.Deploy.Schedule != "" || e.Deploy.Mode != "" || e.Deploy.Namespace != "" || e.Deploy.Image != "" {
		resp.Deploy = &api.DeployConfig{
			Schedule:  e.Deploy.Schedule,
			Mode:      e.Deploy.Mode,
			Namespace: e.Deploy.Namespace,
			Image:     e.Deploy.Image,
		}
	}

	return resp
}

func fromAPIRequest(req api.ETLResponse) etl.ETL {
	e := etl.ETL{
		Name: req.Name,
	}

	for _, src := range req.Sources {
		e.Sources = append(e.Sources, etl.StorageConfig{
			Name:       src.Name,
			Type:       src.Type,
			Connection: src.Connection,
		})
	}

	for _, tgt := range req.Targets {
		e.Targets = append(e.Targets, etl.StorageConfig{
			Name:       tgt.Name,
			Type:       tgt.Type,
			Connection: tgt.Connection,
		})
	}

	for _, q := range req.Queries {
		e.Queries = append(e.Queries, etl.Query{
			Name: q.Name,
			SQL:  q.SQL,
		})
	}

	for _, fn := range req.Functions {
		e.Functions = append(e.Functions, etl.FunctionConfig{
			Name: fn.Name,
			Code: fn.Code,
		})
	}

	for _, p := range req.Pipelines {
		pipeline := etl.Pipeline{
			Name:         p.Name,
			Sources:      p.Sources,
			Targets:      p.Targets,
			Query:        p.Query,
			Functions:    p.Functions,
			UniqueFields: p.UniqueFields,
			KeepLast:     p.KeepLast,
		}
		if p.Tracking != nil {
			pipeline.TrackingSpec = etl.TrackingSpec{
				Field:       p.Tracking.Field,
				StorageType: p.Tracking.StorageType,
			}
		}
		for _, f := range p.Fields {
			pipeline.Fields = append(pipeline.Fields, etl.FieldMapping{
				Source:    f.Source,
				Target:    f.Target,
				Transform: f.Transform,
			})
		}
		e.Pipelines = append(e.Pipelines, pipeline)
	}

	if req.Deploy != nil {
		e.Deploy = etl.DeployConfig{
			Schedule:  req.Deploy.Schedule,
			Mode:      req.Deploy.Mode,
			Namespace: req.Deploy.Namespace,
			Image:     req.Deploy.Image,
		}
	}

	return e
}
