package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ETLVersion struct {
	Version   int    `json:"version" yaml:"version"`
	Label     string `json:"label" yaml:"label"`
	CreatedAt string `json:"createdAt" yaml:"created_at"`
	IsDraft   bool   `json:"isDraft" yaml:"is_draft"`
}

type VersionService struct {
	ConfigDir string
}

func NewVersionService(configDir string) *VersionService {
	return &VersionService{ConfigDir: configDir}
}

func (s *VersionService) versionsDir(etlName string) string {
	return filepath.Join(s.ConfigDir, "_versions", etlName)
}

func (s *VersionService) ListVersions(etlName string) ([]ETLVersion, error) {
	dir := s.versionsDir(etlName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ETLVersion{}, nil
		}
		return nil, err
	}

	var versions []ETLVersion
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		// Read just the metadata from the file header
		var meta struct {
			VersionMeta ETLVersion `yaml:"_version_meta"`
		}
		if err := yaml.Unmarshal(data, &meta); err != nil {
			continue
		}
		if meta.VersionMeta.Version > 0 {
			versions = append(versions, meta.VersionMeta)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (s *VersionService) SaveVersion(etlName, label string, isDraft bool) (*ETLVersion, error) {
	// Read current config
	srcPath := filepath.Join(s.ConfigDir, etlName+".yaml")
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ETL config: %w", err)
	}

	// Determine next version number
	versions, _ := s.ListVersions(etlName)
	nextVersion := 1
	if len(versions) > 0 {
		nextVersion = versions[0].Version + 1
	}

	// Create version metadata
	ver := ETLVersion{
		Version:   nextVersion,
		Label:     label,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		IsDraft:   isDraft,
	}

	// Prepend version metadata to the config
	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	config["_version_meta"] = ver

	out, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	// Write to versions directory
	dir := s.versionsDir(etlName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	versionFile := filepath.Join(dir, fmt.Sprintf("v%s.yaml", strconv.Itoa(nextVersion)))
	if err := os.WriteFile(versionFile, out, 0644); err != nil {
		return nil, err
	}

	return &ver, nil
}

func (s *VersionService) RestoreVersion(etlName string, version int) error {
	versionFile := filepath.Join(s.versionsDir(etlName), fmt.Sprintf("v%d.yaml", version))
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return fmt.Errorf("version %d not found: %w", version, err)
	}

	// Remove version metadata before restoring
	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	delete(config, "_version_meta")

	out, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	destPath := filepath.Join(s.ConfigDir, etlName+".yaml")
	return os.WriteFile(destPath, out, 0644)
}
