package etl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"shadow-dom/etlctl/pkg/util/etl/transform"

	"gopkg.in/yaml.v3"
)

type Query struct {
	Name string `yaml:"name"`
	SQL  string `yaml:"sql"`
}

type DeployConfig struct {
	Schedule  string `yaml:"schedule,omitempty"`
	Mode      string `yaml:"mode,omitempty"` // "cronjob" (default) or "service"
	Namespace string `yaml:"namespace,omitempty"`
	Image     string `yaml:"image,omitempty"`
}

// DAGNodeMeta stores visual layout for a node in the DAG editor.
type DAGNodeMeta struct {
	ID     string         `yaml:"id"     json:"id"`
	Type   string         `yaml:"type"   json:"type"`
	Label  string         `yaml:"label"  json:"label"`
	X      float64        `yaml:"x"      json:"x"`
	Y      float64        `yaml:"y"      json:"y"`
	Config map[string]any `yaml:"config" json:"config"`
}

// DAGEdgeMeta stores a connection between two nodes.
type DAGEdgeMeta struct {
	ID     string `yaml:"id"     json:"id"`
	Source string `yaml:"source" json:"source"`
	Target string `yaml:"target" json:"target"`
}

// DAGLayout stores the visual DAG editor state.
type DAGLayout struct {
	Nodes []DAGNodeMeta `yaml:"nodes" json:"nodes"`
	Edges []DAGEdgeMeta `yaml:"edges" json:"edges"`
}

type ETL struct {
	Sources   []StorageConfig  `yaml:"sources"`
	Targets   []StorageConfig  `yaml:"targets"`
	Pipelines []Pipeline       `yaml:"pipelines"`
	Queries   []Query          `yaml:"queries"`
	Functions []FunctionConfig `yaml:"functions,omitempty"`
	Name      string           `yaml:"name"`
	Deploy    DeployConfig     `yaml:"deploy,omitempty"`
	Dag       *DAGLayout       `yaml:"dag,omitempty"`
	ConfigDir string           `yaml:"-"`
}

func getQueryByName(name string, queries []Query) (string, error) {
	for _, query := range queries {
		if query.Name == name {
			return query.SQL, nil
		}
	}
	return "", fmt.Errorf("invalid query requested: %s", name)
}

func getStorageConfig(name string, storages []StorageConfig) (*StorageConfig, error) {
	for _, storage := range storages {
		if storage.Name == name {
			return &storage, nil
		}
	}
	return nil, fmt.Errorf("invalid data storage requested: %s", name)
}

func getFunctionConfig(name string, functions []FunctionConfig) (*FunctionConfig, error) {
	for _, fn := range functions {
		if fn.Name == name {
			return &fn, nil
		}
	}
	return nil, fmt.Errorf("unknown function: %s", name)
}

// ApplyFunctions runs the pipeline's function steps in order.
func (etl *ETL) ApplyFunctions(pipeline Pipeline, data []map[string]string) ([]map[string]string, error) {
	for _, fnName := range pipeline.Functions {
		_, err := getFunctionConfig(fnName, etl.Functions)
		if err != nil {
			return nil, err
		}
		data, err = RunRegisteredFunction(fnName, data)
		if err != nil {
			return nil, fmt.Errorf("function %q failed: %w", fnName, err)
		}
		fmt.Printf("Applied function %s: %d rows\n", fnName, len(data))
	}
	return data, nil
}

func getDataForColumns(fields []string, data []map[string]string) string {
	results := make([]string, 0, len(data))

	for _, row := range data {
		rowValues := make([]string, 0, len(fields))

		for _, field := range fields {
			rowValues = append(rowValues, fmt.Sprintf("'%s'", row[field]))
		}

		results = append(results, fmt.Sprintf("(%s)", strings.Join(rowValues, ", ")))
	}

	return strings.Join(results, ", ")
}

func CreateETL(configDir string, name string) (*ETL, error) {
	filePath := filepath.Join(configDir, name+".yaml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ETL config %s: %w", filePath, err)
	}

	var etl ETL
	if err := yaml.Unmarshal(data, &etl); err != nil {
		return nil, fmt.Errorf("failed to parse ETL config: %w", err)
	}

	etl.ConfigDir = configDir

	// Migrate legacy target -> targets
	for i := range etl.Pipelines {
		if len(etl.Pipelines[i].Targets) == 0 && etl.Pipelines[i].Target != "" {
			etl.Pipelines[i].Targets = []string{etl.Pipelines[i].Target}
		}
	}

	// Resolve connection references
	resolveConnectionRefs(configDir, etl.Sources)
	resolveConnectionRefs(configDir, etl.Targets)

	return &etl, nil
}

// resolveConnectionRefs replaces connection_ref references with actual connection details
// from the shared _connections.yaml file.
func resolveConnectionRefs(configDir string, configs []StorageConfig) {
	// Find configs that need resolution
	needsResolve := false
	for _, c := range configs {
		if c.ConnectionRef != "" {
			needsResolve = true
			break
		}
	}
	if !needsResolve {
		return
	}

	// Load connections file
	connFile := filepath.Join(configDir, "_connections.yaml")
	data, err := os.ReadFile(connFile)
	if err != nil {
		return // No connections file, skip resolution
	}

	var connData struct {
		Connections []struct {
			Name       string            `yaml:"name"`
			Type       string            `yaml:"type"`
			Connection map[string]string `yaml:"connection"`
		} `yaml:"connections"`
	}
	if err := yaml.Unmarshal(data, &connData); err != nil {
		return
	}

	connMap := make(map[string]struct {
		Type       string
		Connection map[string]string
	})
	for _, c := range connData.Connections {
		connMap[c.Name] = struct {
			Type       string
			Connection map[string]string
		}{c.Type, c.Connection}
	}

	// Resolve references
	for i := range configs {
		if configs[i].ConnectionRef != "" {
			if resolved, ok := connMap[configs[i].ConnectionRef]; ok {
				configs[i].Type = resolved.Type
				configs[i].Connection = resolved.Connection
			}
		}
	}
}

func (etl *ETL) GetSource(name string) (*StorageConfig, error) {
	return getStorageConfig(name, etl.Sources)
}

func (etl *ETL) GetTarget(name string) (*StorageConfig, error) {
	return getStorageConfig(name, etl.Targets)
}

func (etl *ETL) InjectSourceQueryWithState(source string, query string, pipeline Pipeline) string {
	trackingField, lastValue := pipeline.GetTrackingState(source)
	condition := fmt.Sprintf("%s > '%s'", trackingField, lastValue)
	whereRegex := regexp.MustCompile(`(?i)\bWHERE\b`)

	if whereRegex.MatchString(pipeline.Query) {
		return whereRegex.ReplaceAllString(pipeline.Query, "WHERE "+condition+" AND")
	}

	return query + " WHERE " + condition
}

func track(msg string) (string, time.Time) {
	return msg, time.Now()
}

func duration(msg string, start time.Time) {
	log.Printf("%v: %v\n", msg, time.Since(start))
}

func (etl *ETL) Deduplicate(uniqueFields []string, data []map[string]string, keepLast bool) []map[string]string {
	defer duration(track("dedup"))

	var sb strings.Builder

	if keepLast {
		// Iterate forward, overwriting map entries so the last occurrence wins
		seen := make(map[string]int) // key -> index in results
		var results []map[string]string

		for _, record := range data {
			for _, field := range uniqueFields {
				if value, exists := record[field]; exists {
					sb.WriteString(value)
					sb.WriteString("|")
				}
			}
			key := sb.String()
			sb.Reset()

			if idx, exists := seen[key]; exists {
				results[idx] = record
			} else {
				seen[key] = len(results)
				results = append(results, record)
			}
		}

		return results
	}

	observed := make(map[string]bool)
	var results []map[string]string

	for _, record := range data {
		for _, field := range uniqueFields {
			if value, exists := record[field]; exists {
				sb.WriteString(value)
				sb.WriteString("|")
			}
		}

		if !observed[sb.String()] {
			observed[sb.String()] = true
			results = append(results, record)
		}

		sb.Reset()
	}

	return results
}

func updateQueryWithState(pipeline Pipeline, source string, query string) (string, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimSuffix(query, ";")

	if state, ok := pipeline.State.Sources[source]; ok {
		if state.Field == "" || state.LastValue == "" {
			return query, nil
		}

		condition := fmt.Sprintf("%s > '%s'", state.Field, state.LastValue)

		whereRegex := regexp.MustCompile(`(?i)\bWHERE\b`)
		if whereRegex.MatchString(query) {
			return whereRegex.ReplaceAllString(pipeline.Query, "WHERE "+condition+" AND"), nil
		}

		return query + " WHERE " + condition, nil
	}

	return query, fmt.Errorf("missing state: could not load state for source (%s)", source)
}

func (etl *ETL) Extract(sourceName string, pipeline *Pipeline) ([]map[string]string, error) {
	if err := pipeline.InitState(); err != nil {
		return nil, err
	}

	sc, err := etl.GetSource(sourceName)
	if err != nil {
		return nil, err
	}

	// Resolve relative file paths against the config directory
	conn := sc.Connection
	if fp := conn["filepath"]; fp != "" && !filepath.IsAbs(fp) && !strings.HasPrefix(fp, "~/") {
		conn = make(map[string]string)
		for k, v := range sc.Connection {
			conn[k] = v
		}
		conn["filepath"] = filepath.Join(etl.ConfigDir, fp)
	}

	// Create source via registry
	source, err := NewSource(sc.Type, sc.Name, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create source %s: %w", sourceName, err)
	}

	if err := source.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to source %s: %w", sourceName, err)
	}
	defer source.Close()

	// Build query (DB sources use SQL, others may ignore it)
	query := ""
	if pipeline.Query != "" {
		query, err = getQueryByName(pipeline.Query, etl.Queries)
		if err != nil {
			return nil, err
		}
		query, _ = updateQueryWithState(*pipeline, sourceName, query)
	}

	data, err := source.Extract(query)
	if err != nil {
		return nil, fmt.Errorf("extraction failed for source %s: %w", sourceName, err)
	}

	if len(data) > 0 {
		fmt.Printf("Extracted %d rows from %s\n", len(data), sourceName)
	}
	return data, nil
}

// ExtractFromSource extracts data using a pre-connected source (for persistent connections in Listen mode).
func (etl *ETL) ExtractFromSource(source Source, sourceName string, pipeline *Pipeline) ([]map[string]string, error) {
	if err := pipeline.InitState(); err != nil {
		return nil, err
	}

	// Build query (DB sources use SQL, others may ignore it)
	query := ""
	var err error
	if pipeline.Query != "" {
		query, err = getQueryByName(pipeline.Query, etl.Queries)
		if err != nil {
			return nil, err
		}
		query, _ = updateQueryWithState(*pipeline, sourceName, query)
	}

	data, err := source.Extract(query)
	if err != nil {
		return nil, fmt.Errorf("extraction failed for source %s: %w", sourceName, err)
	}

	if len(data) > 0 {
		fmt.Printf("Extracted %d rows from %s\n", len(data), sourceName)
	}
	return data, nil
}

func (etl *ETL) Load(pipeline Pipeline, targetRef string, data []map[string]string) error {
	targetName, targetTable := pipeline.GetTargetInfo(targetRef)

	fmt.Printf("Writing data to target %s...\n", targetName)

	tc, err := etl.GetTarget(targetName)
	if err != nil {
		return err
	}

	// Resolve relative file paths against the config directory
	conn := tc.Connection
	if fp := conn["filepath"]; fp != "" && !filepath.IsAbs(fp) && !strings.HasPrefix(fp, "~/") {
		conn = make(map[string]string)
		for k, v := range tc.Connection {
			conn[k] = v
		}
		conn["filepath"] = filepath.Join(etl.ConfigDir, fp)
	}

	// Create target via registry
	target, err := NewTarget(tc.Type, tc.Name, conn)
	if err != nil {
		return fmt.Errorf("failed to create target %s: %w", targetName, err)
	}

	if err := target.Connect(); err != nil {
		return fmt.Errorf("failed to connect to target %s: %w", targetName, err)
	}
	defer target.Close()

	if err := target.Load(targetTable, pipeline.Fields, data); err != nil {
		return err
	}

	fmt.Printf("Loaded %d rows into %s.%s\n", len(data), targetName, targetTable)
	return nil
}

// ApplyTransforms runs transform expressions on each row's fields.
func (etl *ETL) ApplyTransforms(pipeline Pipeline, data []map[string]string) ([]map[string]string, error) {
	hasTransforms := false
	for _, f := range pipeline.Fields {
		if f.Transform != "" {
			hasTransforms = true
			break
		}
	}
	if !hasTransforms {
		return data, nil
	}

	for i, row := range data {
		for _, field := range pipeline.Fields {
			if field.Transform == "" {
				continue
			}
			val := row[field.Source]
			transformed, err := transform.Apply(field.Transform, val)
			if err != nil {
				return nil, fmt.Errorf("transform error on field %q row %d: %w", field.Source, i, err)
			}
			row[field.Source] = transformed
		}
	}

	return data, nil
}

func (etl *ETL) Run() error {
	// Cache extracted data so shared sources aren't re-extracted across pipelines
	extractCache := make(map[string][]map[string]string)

	for _, pipeline := range etl.Pipelines {
		pipeline.ConfigDir = etl.ConfigDir
		data := make([]map[string]string, 0)

		var mu sync.Mutex
		var wg sync.WaitGroup
		var extractErrors []error

		for _, sourceName := range pipeline.Sources {
			wg.Add(1)
			go func(source string) {
				defer wg.Done()

				cacheKey := source + "|" + pipeline.Query

				mu.Lock()
				if cached, ok := extractCache[cacheKey]; ok {
					data = append(data, cached...)
					mu.Unlock()
					return
				}
				mu.Unlock()

				result, err := etl.Extract(source, &pipeline)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					extractErrors = append(extractErrors, err)
					return
				}

				if len(result) > 0 {
					pipeline.UpdateTrackingState(source, result[len(result)-1])
				}

				extractCache[cacheKey] = result
				data = append(data, result...)
			}(sourceName)
		}

		wg.Wait()

		if len(extractErrors) > 0 {
			return fmt.Errorf("extraction errors: %w", errors.Join(extractErrors...))
		}

		if len(data) == 0 {
			fmt.Printf("No data extracted for pipeline %s, skipping load.\n", pipeline.Name)
			continue
		}

		if len(pipeline.UniqueFields) > 0 {
			data = etl.Deduplicate(pipeline.UniqueFields, data, pipeline.KeepLast)
		}

		// Apply transforms
		data, err := etl.ApplyTransforms(pipeline, data)
		if err != nil {
			return fmt.Errorf("transform failed for pipeline %s: %w", pipeline.Name, err)
		}

		// Apply functions
		data, err = etl.ApplyFunctions(pipeline, data)
		if err != nil {
			return fmt.Errorf("function failed for pipeline %s: %w", pipeline.Name, err)
		}

		// Load into all targets
		for _, targetRef := range pipeline.GetAllTargets() {
			if err := etl.Load(pipeline, targetRef, data); err != nil {
				return fmt.Errorf("load failed for pipeline %s target %s: %w", pipeline.Name, targetRef, err)
			}
		}

		pipeline.SaveState()
	}

	return nil
}

// Listen runs the ETL in listener mode for event-driven sources.
// It loops continuously, extracting from listener sources on each iteration,
// then running transform+load. Non-listener sources are extracted once and cached.
func (etl *ETL) Listen() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Cache for non-listener source data (extracted once)
	staticCache := make(map[string][]map[string]string)

	// Persistent sources for listener-type sources (keyed by source name)
	persistentSources := make(map[string]Source)
	// Track which sources are listeners
	isListenerSource := make(map[string]bool)

	// Pre-create listener sources
	for _, pipeline := range etl.Pipelines {
		for _, sourceName := range pipeline.Sources {
			if _, exists := persistentSources[sourceName]; exists {
				continue
			}

			sc, err := etl.GetSource(sourceName)
			if err != nil {
				continue
			}

			s, err := NewSource(sc.Type, sc.Name, sc.Connection)
			if err != nil {
				continue
			}

			if _, ok := s.(Listener); ok {
				if err := s.Connect(); err != nil {
					log.Printf("Failed to connect persistent source %s: %v", sourceName, err)
					s.Close()
					continue
				}
				persistentSources[sourceName] = s
				isListenerSource[sourceName] = true
			} else {
				s.Close()
				isListenerSource[sourceName] = false
			}
		}
	}

	// Ensure persistent sources are closed on shutdown
	defer func() {
		for name, s := range persistentSources {
			log.Printf("Closing persistent source %s", name)
			s.Close()
		}
	}()

	fmt.Println("Starting listener mode...")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down listener...")
			return nil
		default:
		}

		for _, pipeline := range etl.Pipelines {
			pipeline.ConfigDir = etl.ConfigDir
			data := make([]map[string]string, 0)

			var mu sync.Mutex
			var wg sync.WaitGroup
			var extractErrors []error

			// Track which persistent sources were used for ACK/NACK
			type sourceExtraction struct {
				sourceName string
				source     Source
			}
			var usedSources []sourceExtraction

			for _, sourceName := range pipeline.Sources {
				wg.Add(1)
				go func(source string) {
					defer wg.Done()

					if !isListenerSource[source] {
						// Non-listener: use cached data
						mu.Lock()
						if cached, ok := staticCache[source]; ok {
							data = append(data, cached...)
							mu.Unlock()
							return
						}
						mu.Unlock()

						// Extract once and cache
						result, err := etl.Extract(source, &pipeline)
						mu.Lock()
						defer mu.Unlock()
						if err != nil {
							extractErrors = append(extractErrors, err)
							return
						}
						staticCache[source] = result
						data = append(data, result...)
						return
					}

					// Listener source: use persistent connection
					s := persistentSources[source]
					if s == nil {
						mu.Lock()
						extractErrors = append(extractErrors, fmt.Errorf("no persistent source for %s", source))
						mu.Unlock()
						return
					}

					// Reconnect if dead
					if aliver, ok := s.(Aliver); ok && !aliver.IsAlive() {
						log.Printf("Reconnecting source %s...", source)
						s.Close()
						if err := s.Connect(); err != nil {
							mu.Lock()
							extractErrors = append(extractErrors, fmt.Errorf("reconnect failed for %s: %w", source, err))
							mu.Unlock()
							return
						}
					}

					result, err := etl.ExtractFromSource(s, source, &pipeline)
					mu.Lock()
					defer mu.Unlock()

					if err != nil {
						extractErrors = append(extractErrors, err)
						return
					}

					usedSources = append(usedSources, sourceExtraction{source, s})
					data = append(data, result...)
				}(sourceName)
			}

			wg.Wait()

			if len(extractErrors) > 0 {
				// NACK any pending deliveries on error
				for _, se := range usedSources {
					if acker, ok := se.source.(Acknowledger); ok {
						acker.NackLast()
					}
				}
				log.Printf("Extraction errors: %v", errors.Join(extractErrors...))
				continue
			}

			if len(data) == 0 {
				continue
			}

			if len(pipeline.UniqueFields) > 0 {
				data = etl.Deduplicate(pipeline.UniqueFields, data, pipeline.KeepLast)
			}

			data, err := etl.ApplyTransforms(pipeline, data)
			if err != nil {
				for _, se := range usedSources {
					if acker, ok := se.source.(Acknowledger); ok {
						acker.NackLast()
					}
				}
				log.Printf("Transform failed for pipeline %s: %v", pipeline.Name, err)
				continue
			}

			data, err = etl.ApplyFunctions(pipeline, data)
			if err != nil {
				for _, se := range usedSources {
					if acker, ok := se.source.(Acknowledger); ok {
						acker.NackLast()
					}
				}
				log.Printf("Function failed for pipeline %s: %v", pipeline.Name, err)
				continue
			}

			loadFailed := false
			for _, targetRef := range pipeline.GetAllTargets() {
				if err := etl.Load(pipeline, targetRef, data); err != nil {
					log.Printf("Load failed for pipeline %s target %s: %v", pipeline.Name, targetRef, err)
					loadFailed = true
				}
			}

			// ACK or NACK based on load success
			for _, se := range usedSources {
				if acker, ok := se.source.(Acknowledger); ok {
					if loadFailed {
						acker.NackLast()
					} else {
						if err := acker.AckLast(); err != nil {
							log.Printf("ACK failed for source %s: %v", se.sourceName, err)
						}
					}
				}
			}

			pipeline.SaveState()
		}
	}
}
