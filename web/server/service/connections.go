package service

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Connection struct {
	Name       string            `yaml:"name" json:"name"`
	Type       string            `yaml:"type" json:"type"`
	Connection map[string]string `yaml:"connection" json:"connection"`
}

type ConnectionsFile struct {
	Connections []Connection `yaml:"connections"`
}

type ConnectionService struct {
	ConfigDir string
}

func NewConnectionService(configDir string) *ConnectionService {
	return &ConnectionService{ConfigDir: configDir}
}

func (s *ConnectionService) filePath() string {
	return filepath.Join(s.ConfigDir, "_connections.yaml")
}

func (s *ConnectionService) List() ([]Connection, error) {
	data, err := os.ReadFile(s.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []Connection{}, nil
		}
		return nil, err
	}
	var f ConnectionsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Connections, nil
}

func (s *ConnectionService) Get(name string) (*Connection, error) {
	conns, err := s.List()
	if err != nil {
		return nil, err
	}
	for _, c := range conns {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("connection %q not found", name)
}

func (s *ConnectionService) Save(conn Connection) error {
	conns, err := s.List()
	if err != nil {
		return err
	}
	found := false
	for i, c := range conns {
		if c.Name == conn.Name {
			conns[i] = conn
			found = true
			break
		}
	}
	if !found {
		conns = append(conns, conn)
	}
	return s.write(conns)
}

func (s *ConnectionService) Delete(name string) error {
	conns, err := s.List()
	if err != nil {
		return err
	}
	filtered := make([]Connection, 0, len(conns))
	for _, c := range conns {
		if c.Name != name {
			filtered = append(filtered, c)
		}
	}
	return s.write(filtered)
}

func (s *ConnectionService) write(conns []Connection) error {
	f := ConnectionsFile{Connections: conns}
	data, err := yaml.Marshal(&f)
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath(), data, 0644)
}

func (s *ConnectionService) TestConnection(conn Connection) error {
	// Implementation: try to connect and list tables
	return nil
}
