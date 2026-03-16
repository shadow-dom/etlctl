package service

import (
	"fmt"
	"os"
	"path/filepath"

	"shadow-dom/etlctl/pkg/util/etl"
	"shadow-dom/etlctl/web/server/api"

	"gopkg.in/yaml.v3"
)

type DAGService struct {
	ConfigDir string
}

func NewDAGService(configDir string) *DAGService {
	return &DAGService{ConfigDir: configDir}
}

func (d *DAGService) yamlPath(name string) string {
	return filepath.Join(d.ConfigDir, name+".yaml")
}

func (d *DAGService) Get(name string) (*api.DAGMeta, error) {
	path := d.yamlPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &api.DAGMeta{
				Nodes: []api.DAGNode{},
				Edges: []api.DAGEdge{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read ETL config: %w", err)
	}

	var e etl.ETL
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("failed to parse ETL config: %w", err)
	}

	if e.Dag == nil {
		// Fall back to legacy _dag.json if it exists
		return d.getLegacyDag(name)
	}

	return dagLayoutToAPI(e.Dag), nil
}

func (d *DAGService) Save(name string, meta *api.DAGMeta) error {
	path := d.yamlPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read ETL config for DAG save: %w", err)
	}

	var e etl.ETL
	if err := yaml.Unmarshal(data, &e); err != nil {
		return fmt.Errorf("failed to parse ETL config: %w", err)
	}

	e.Dag = apiToDagLayout(meta)

	out, err := yaml.Marshal(&e)
	if err != nil {
		return fmt.Errorf("failed to marshal ETL config: %w", err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("failed to write ETL config: %w", err)
	}

	// Clean up legacy _dag.json if it exists
	legacyPath := filepath.Join(d.ConfigDir, name+"_dag.json")
	os.Remove(legacyPath)

	return nil
}

// getLegacyDag reads from the old _dag.json file for backward compatibility.
func (d *DAGService) getLegacyDag(name string) (*api.DAGMeta, error) {
	legacyPath := filepath.Join(d.ConfigDir, name+"_dag.json")
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return &api.DAGMeta{
			Nodes: []api.DAGNode{},
			Edges: []api.DAGEdge{},
		}, nil
	}

	// Parse as generic JSON (the legacy format matches api.DAGMeta)
	var meta api.DAGMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return &api.DAGMeta{
			Nodes: []api.DAGNode{},
			Edges: []api.DAGEdge{},
		}, nil
	}

	return &meta, nil
}

func dagLayoutToAPI(layout *etl.DAGLayout) *api.DAGMeta {
	meta := &api.DAGMeta{
		Nodes: make([]api.DAGNode, len(layout.Nodes)),
		Edges: make([]api.DAGEdge, len(layout.Edges)),
	}
	for i, n := range layout.Nodes {
		meta.Nodes[i] = api.DAGNode{
			ID:     n.ID,
			Type:   n.Type,
			Label:  n.Label,
			X:      n.X,
			Y:      n.Y,
			Config: n.Config,
		}
	}
	for i, e := range layout.Edges {
		meta.Edges[i] = api.DAGEdge{
			ID:     e.ID,
			Source: e.Source,
			Target: e.Target,
		}
	}
	return meta
}

func apiToDagLayout(meta *api.DAGMeta) *etl.DAGLayout {
	layout := &etl.DAGLayout{
		Nodes: make([]etl.DAGNodeMeta, len(meta.Nodes)),
		Edges: make([]etl.DAGEdgeMeta, len(meta.Edges)),
	}
	for i, n := range meta.Nodes {
		layout.Nodes[i] = etl.DAGNodeMeta{
			ID:     n.ID,
			Type:   n.Type,
			Label:  n.Label,
			X:      n.X,
			Y:      n.Y,
			Config: n.Config,
		}
	}
	for i, e := range meta.Edges {
		layout.Edges[i] = etl.DAGEdgeMeta{
			ID:     e.ID,
			Source: e.Source,
			Target: e.Target,
		}
	}
	return layout
}
