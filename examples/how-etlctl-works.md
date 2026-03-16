# How etlctl Works

A schema-driven ETL platform that lets you define, run, and deploy data pipelines entirely through YAML.

---

## Overview

etlctl has three execution modes:

1. **Interpreted** (`run` / `listen`) — executes YAML definitions directly
2. **Compiled** (`generate`) — generates a standalone Go binary from YAML
3. **Deployed** (`deploy`) — produces Kubernetes manifests for containerized execution

---

## Pipeline Lifecycle

```
  YAML Definition
       │
       ▼
  ┌──────────┐     ┌──────────┐     ┌──────────┐
  │ validate  │     │ generate │     │  deploy   │
  │           │     │          │     │           │
  │ check     │     │ YAML →   │     │ YAML →   │
  │ structure │     │ Go code  │     │ K8s YAML  │
  └──────────┘     └─────┬────┘     └─────┬────┘
                         │                │
                    go build         kubectl apply
                         │                │
                         ▼                ▼
                   Standalone       CronJob or
                     Binary         Deployment
       │
       ▼
  ┌──────────────────────────────────┐
  │         run  or  listen          │
  │                                  │
  │  Extract → Dedup → Transform    │
  │     → Functions → Load → State  │
  └──────────────────────────────────┘
```

---

## Execution: What Happens at Runtime

When you run `etlctl run <name>` or `etlctl listen <name>`, here's the exact sequence:

### 1. Load Configuration

```
CreateETL(configDir, name)
  ├─ Read YAML file
  ├─ Unmarshal into ETL struct
  └─ Resolve connection references (_connections.yaml)
```

No connections are opened at this stage. The YAML is parsed into an in-memory struct.

### 2. For Each Pipeline

Pipelines execute sequentially. Within each pipeline:

#### Extract (parallel across sources)

```
FOR each source (in parallel goroutines):
  ├─ Load pipeline state from disk (etls/state/<pipeline>_state.yaml)
  ├─ Resolve source config from YAML
  ├─ Create Source via registry → NewSource(type, name, connection)
  ├─ source.Connect()
  ├─ Build query (apply state-based WHERE clause for incremental runs)
  ├─ source.Extract(query) → []map[string]string
  ├─ Update tracking state with last record's key value
  ├─ Cache result (avoid re-extracting if shared across pipelines)
  └─ source.Close()
```

All data flows as `[]map[string]string` — a list of key-value rows. This uniform format lets any source type feed into any target type.

#### Deduplicate (optional)

If `uniqueFields` is set on the pipeline:
- Groups rows by the specified field(s)
- `keepLast: true` keeps the last occurrence, `false` keeps the first

#### Transform (per-field, per-row)

For each row, for each field with a `transform` expression:

```yaml
fields:
  - source: customer
    target: customer_name
    transform: "trim|title"        # chainable with pipes
  - source: amount
    target: amount
    transform: "to_float|multiply:1.1|round:2"
```

Built-in transforms:

| Category | Transforms |
|----------|-----------|
| String | `uppercase`, `lowercase`, `trim`, `title` |
| Type | `to_int`, `to_float`, `to_bool` |
| Math | `multiply:N`, `divide:N`, `add:N`, `round:N` |
| String | `replace:old->new`, `prefix:X`, `suffix:X`, `truncate:N`, `default:fallback` |
| Date | `date_format:2006-01-02` |

Custom transforms can be registered with `transform.Register()`.

#### Functions (custom Go logic)

Functions operate on the entire dataset (not per-field):

```go
type PipelineFunc func(data []map[string]string) ([]map[string]string, error)
```

Functions must be compiled into the binary and registered via `etl.RegisterFunction()`. The YAML `functions` section declares them, but execution requires the Go function to exist at runtime.

#### Load (multi-target)

```
FOR each target in pipeline.targets:
  ├─ Create Target via registry → NewTarget(type, name, connection)
  ├─ target.Connect()
  ├─ target.Load(tableName, fieldMappings, data)
  └─ target.Close()
```

A single pipeline can write to multiple targets simultaneously — JSON file, SQLite, CSV, and an API endpoint all from the same extracted data.

#### Save State

Writes `etls/state/<pipeline>_state.yaml` with the last-seen value of the tracking field. Next run uses this to build incremental WHERE clauses, fetching only new data.

---

## Run vs Listen

| | `run` | `listen` |
|---|---|---|
| Execution | Once, then exits | Continuous loop until SIGTERM |
| Sources | All extracted once | Listeners block waiting for events; non-listeners cached |
| Use case | Batch/scheduled jobs | Event-driven (RabbitMQ, webhooks) |
| K8s mode | CronJob | Deployment (Service) |

In `listen` mode, sources that implement the `Listener` interface (RabbitMQ, Webhook) block until data arrives. Non-listener sources (CSV, JSON, API) are extracted once and cached across iterations.

---

## The Registry Pattern

Sources and targets are pluggable. Each type registers a factory function at init time:

```go
// In sources/csv.go
func init() {
    etl.RegisterSource("csv", func(name string, config map[string]string) (etl.Source, error) {
        return &CSVSource{name: name, filepath: config["filepath"]}, nil
    })
}
```

The blank import in the CLI entrypoint triggers all registrations:

```go
import (
    _ "shadow-dom/etlctl/pkg/util/etl/sources"  // registers csv, json, api, rabbitmq, webhook
    _ "shadow-dom/etlctl/pkg/util/etl/targets"   // registers csv, json, api
)
```

At runtime, `NewSource("rabbitmq", "my_queue", config)` looks up the factory and creates the concrete type. This means adding a new source/target type is just:

1. Implement the `Source` or `Target` interface
2. Register it in an `init()` function
3. Import the package

### Source Interface

```go
type Source interface {
    Name() string
    Type() string
    Connect() error
    Extract(query string) ([]map[string]string, error)
    Close() error
}
```

### Target Interface

```go
type Target interface {
    Name() string
    Type() string
    Connect() error
    Load(tableName string, fields []FieldMapping, data []map[string]string) error
    Close() error
}
```

### Registered Types

**Sources:** csv, json, api, sqlite3, postgres, rabbitmq, webhook
**Targets:** csv, json, api, sqlite3, postgres, sqlserver

---

## Code Generation (`generate`)

`etlctl generate <name>` reads the YAML and renders a Go source file using an embedded template (`pkg/builder/etl.tmpl`):

```go
package main

import (
    "shadow-dom/etlctl/pkg/util/etl"
    _ "shadow-dom/etlctl/pkg/util/etl/sources"
    _ "shadow-dom/etlctl/pkg/util/etl/targets"
)

func main() {
    e := etl.ETL{
        Name: "my-pipeline",
        Sources: []etl.StorageConfig{
            {Name: "src", Type: "csv", Connection: map[string]string{"filepath": "data.csv"}},
        },
        // ... all YAML values embedded as Go literals
    }
    e.Run()
}
```

The generated file is a standalone Go program. Compile it with `go build` for a single binary with no YAML dependency at runtime.

---

## Deployment (`deploy`)

`etlctl deploy <name>` generates Kubernetes manifests:

### What Gets Generated

```
deploy/
├── Dockerfile           # Multi-stage build
├── configmap.yaml       # ETL YAML embedded as a ConfigMap
├── secret.yaml          # Template for connection credentials
└── cronjob.yaml         # or deployment.yaml (based on mode)
```

### Dockerfile (multi-stage)

```dockerfile
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache gcc musl-dev      # needed for sqlite3 (cgo)
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o etlctl ./pkg/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/etlctl .
COPY etls/ /etc/etlctl/configs/
ENTRYPOINT ["./etlctl"]
```

What goes into the image:
- The compiled `etlctl` binary (with all source/target types linked in)
- The `etls/` directory (all YAML configs + data files)
- Alpine base with CA certificates (for HTTPS sources/targets)
- CGO enabled for sqlite3 support via `musl-dev`

### CronJob Mode (default)

For batch pipelines that run on a schedule:

```yaml
apiVersion: batch/v1
kind: CronJob
spec:
  schedule: "*/5 * * * *"           # from deploy.schedule
  jobTemplate:
    spec:
      containers:
        - command: ["./etlctl", "run", "<name>", "--config-dir", "/etc/etlctl/configs"]
          volumeMounts:
            - name: config
              mountPath: /etc/etlctl/configs
          envFrom:
            - secretRef:
                name: <name>-secrets
      volumes:
        - name: config
          configMap:
            name: <name>-config
```

### Service Mode

For event-driven pipelines (RabbitMQ, webhooks):

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 1
  containers:
    - command: ["./etlctl", "listen", "<name>", "--config-dir", "/etc/etlctl/configs"]
  restartPolicy: Always
```

### Deploy Config

Set in the YAML or override with CLI flags:

```yaml
deploy:
  schedule: "*/5 * * * *"    # cron expression (cronjob mode)
  mode: "cronjob|service"     # execution mode
  namespace: "default"        # K8s namespace
  image: "etlctl:latest"      # container image
```

---

## Incremental State Tracking

Pipelines can track their progress for incremental extraction:

```yaml
pipelines:
  - name: sync_orders
    tracking:
      field: "updated_at"
      storage_type: "file"
```

**State file** (`etls/state/sync_orders_state.yaml`):

```yaml
sources:
  order_db:
    field: "updated_at"
    last_value: "2026-03-15 10:30:00"
storage_type: file
```

On the next run, the query gets a WHERE clause appended: `WHERE updated_at > '2026-03-15 10:30:00'`, so only new/updated records are processed.

---

## Interpreted vs Compiled

| Aspect | Interpreted (`run`/`listen`) | Compiled (`generate`) |
|--------|------------------------------|----------------------|
| YAML at runtime | Required | Embedded in binary |
| Startup | Parse YAML each time | Pre-compiled, instant |
| Modification | Edit YAML, re-run | Regenerate + recompile |
| Custom functions | Must be pre-registered in Go | Can add to generated code |
| Deployment | Binary + YAML configs | Single binary |
| Best for | Development, testing | Production, CI/CD |

---

## Full File Map

```
pkg/
├── main.go                          # CLI entrypoint
├── cmd/
│   ├── root.go                      # Cobra root command, --config-dir
│   ├── run.go                       # etlctl run
│   ├── listen.go                    # etlctl listen
│   ├── generate.go                  # etlctl generate
│   ├── deploy.go                    # etlctl deploy
│   ├── validate.go                  # etlctl validate
│   ├── list.go                      # etlctl list
│   ├── init_cmd.go                  # etlctl init (scaffold)
│   └── serve.go                     # etlctl serve (web UI)
├── util/etl/
│   ├── etl.go                       # Core: Run(), Listen(), Extract(), Load()
│   ├── core.go                      # High-level entry: Run(dir,name), Listen(dir,name)
│   ├── pipeline.go                  # Pipeline struct, state management
│   ├── storage.go                   # StorageConfig (YAML representation)
│   ├── source.go                    # Source interface
│   ├── target.go                    # Target interface
│   ├── registry.go                  # Factory registry for sources/targets
│   ├── function.go                  # Function registration and execution
│   ├── builtins.go                  # Built-in registered functions
│   ├── validate.go                  # Structural validation
│   ├── schema.go                    # Auto-mapping and introspection
│   ├── dbstorage.go                 # Database source/target (sqlite3, postgres, sqlserver)
│   ├── transform/
│   │   ├── transform.go             # Transform engine (Apply, chain parsing)
│   │   └── builtins.go              # Built-in transforms
│   ├── sources/
│   │   ├── csv.go, json.go, api.go  # File/HTTP sources
│   │   ├── rabbitmq.go              # Event-driven source (listener)
│   │   └── webhook.go               # HTTP listener source
│   └── targets/
│       ├── csv.go, json.go          # File targets
│       └── api.go                   # HTTP POST target
├── builder/
│   ├── builder.go                   # Code generation logic
│   └── etl.tmpl                     # Go template for generated code
└── k8s/
    ├── generator.go                 # K8s manifest generation
    └── templates/
        ├── Dockerfile.tmpl
        ├── configmap.tmpl
        ├── secret.tmpl
        ├── cronjob.tmpl
        └── deployment.tmpl
```
