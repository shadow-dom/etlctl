package api

// ETLResponse is the JSON representation of an ETL config.
type ETLResponse struct {
	Name      string           `json:"name"`
	Sources   []StorageConfig  `json:"sources"`
	Targets   []StorageConfig  `json:"targets"`
	Pipelines []PipelineConfig `json:"pipelines"`
	Queries   []QueryConfig    `json:"queries"`
	Functions []FunctionConfig `json:"functions,omitempty"`
	Deploy    *DeployConfig    `json:"deploy,omitempty"`
}

type FunctionConfig struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type StorageConfig struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Connection map[string]string `json:"connection"`
}

type PipelineConfig struct {
	Name         string         `json:"name"`
	Sources      []string       `json:"sources"`
	Targets      []string       `json:"targets"`
	Fields       []FieldMapping `json:"fields"`
	Query        string         `json:"query,omitempty"`
	Functions    []string       `json:"functions,omitempty"`
	UniqueFields []string       `json:"uniqueFields,omitempty"`
	KeepLast     bool           `json:"keepLast,omitempty"`
	Tracking     *TrackingSpec  `json:"tracking,omitempty"`
}

type FieldMapping struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Transform string `json:"transform,omitempty"`
}

type TrackingSpec struct {
	Field       string `json:"field"`
	StorageType string `json:"storageType"`
}

type QueryConfig struct {
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

type DeployConfig struct {
	Schedule  string `json:"schedule,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Image     string `json:"image,omitempty"`
}

// DAG metadata for frontend layout persistence.
type DAGMeta struct {
	Nodes []DAGNode `json:"nodes"`
	Edges []DAGEdge `json:"edges"`
}

type DAGNode struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"` // "source", "transform", "target", "query", "function"
	Label  string         `json:"label"`
	X      float64        `json:"x"`
	Y      float64        `json:"y"`
	Config map[string]any `json:"config"`
}

type DAGEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// Validation response.
type ValidateResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

// Test step result.
type TestStepResult struct {
	Step       string              `json:"step"`
	Node       string              `json:"node"`
	RowCount   int                 `json:"rowCount"`
	Sample     []map[string]string `json:"sample"`
	Error      string              `json:"error,omitempty"`
	DurationMs int64               `json:"durationMs"`
}

type TestResponse struct {
	Steps   []TestStepResult `json:"steps"`
	Success bool             `json:"success"`
}

// Registry responses.
type RegistryResponse struct {
	Types []string `json:"types"`
}

type TransformRegistryResponse struct {
	Simple       []string `json:"simple"`
	Parameterized []string `json:"parameterized"`
}

type TransformPreviewRequest struct {
	Expression string `json:"expression"`
	Value      string `json:"value"`
}

type TransformPreviewResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// Schema introspection types.
type SchemaIntrospectRequest struct {
	Type       string            `json:"type"`
	Connection map[string]string `json:"connection"`
	Role       string            `json:"role"` // "source" or "target"
}

type SchemaIntrospectResponse struct {
	Tables []string `json:"tables"`
}

type SchemaDescribeRequest struct {
	Type       string            `json:"type"`
	Connection map[string]string `json:"connection"`
	Role       string            `json:"role"`
	Table      string            `json:"table"`
}

type SchemaDescribeResponse struct {
	Table   string           `json:"table"`
	Columns []SchemaColumn   `json:"columns"`
}

type SchemaColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
}

type SchemaAutoMapRequest struct {
	Source SchemaEndpoint `json:"source"`
	Target SchemaEndpoint `json:"target"`
}

type SchemaEndpoint struct {
	Type       string            `json:"type"`
	Connection map[string]string `json:"connection"`
	Table      string            `json:"table"`
}

type SchemaAutoMapResponse struct {
	Fields          []FieldMapping `json:"fields"`
	UnmatchedSource []string       `json:"unmatched_source"`
	UnmatchedTarget []string       `json:"unmatched_target"`
}

// ErrorResponse for API errors.
type ErrorResponse struct {
	Error string `json:"error"`
}
