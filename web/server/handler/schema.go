package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/user"
	"path/filepath"

	"shadow-dom/etlctl/pkg/util/etl"
	"shadow-dom/etlctl/web/server/api"
)

type SchemaHandler struct{}

func NewSchemaHandler() *SchemaHandler {
	return &SchemaHandler{}
}

func (h *SchemaHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	var req api.SchemaIntrospectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	introspector, err := h.getIntrospector(req.Type, req.Role, req.Connection)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer h.closeIntrospector(introspector)

	tables, err := introspector.ListTables()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tables: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, api.SchemaIntrospectResponse{Tables: tables})
}

func (h *SchemaHandler) Describe(w http.ResponseWriter, r *http.Request) {
	var req api.SchemaDescribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	introspector, err := h.getIntrospector(req.Type, req.Role, req.Connection)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer h.closeIntrospector(introspector)

	schema, err := introspector.DescribeTable(req.Table)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to describe table: "+err.Error())
		return
	}

	resp := api.SchemaDescribeResponse{Table: schema.Table}
	for _, col := range schema.Columns {
		resp.Columns = append(resp.Columns, api.SchemaColumn{
			Name:     col.Name,
			DataType: col.DataType,
			Nullable: col.Nullable,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *SchemaHandler) AutoMap(w http.ResponseWriter, r *http.Request) {
	var req api.SchemaAutoMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	srcIntrospector, err := h.getIntrospector(req.Source.Type, "source", req.Source.Connection)
	if err != nil {
		writeError(w, http.StatusBadRequest, "source: "+err.Error())
		return
	}
	defer h.closeIntrospector(srcIntrospector)

	tgtIntrospector, err := h.getIntrospector(req.Target.Type, "target", req.Target.Connection)
	if err != nil {
		writeError(w, http.StatusBadRequest, "target: "+err.Error())
		return
	}
	defer h.closeIntrospector(tgtIntrospector)

	// For flat sources (csv, json), the table name may be empty — that's valid.
	// For DB sources, if no table is specified, try listing tables and use the first one.
	srcTable := req.Source.Table
	if srcTable == "" && dbTypes[req.Source.Type] {
		tables, err := srcIntrospector.ListTables()
		if err == nil && len(tables) > 0 {
			srcTable = tables[0]
		}
	}

	srcSchema, err := srcIntrospector.DescribeTable(srcTable)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to describe source table: "+err.Error())
		return
	}

	tgtTable := req.Target.Table
	if tgtTable == "" && dbTypes[req.Target.Type] {
		tables, err := tgtIntrospector.ListTables()
		if err == nil && len(tables) > 0 {
			tgtTable = tables[0]
		}
	}

	tgtSchema, err := tgtIntrospector.DescribeTable(tgtTable)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to describe target table: "+err.Error())
		return
	}

	result := etl.AutoMap(srcSchema, tgtSchema)

	resp := api.SchemaAutoMapResponse{
		UnmatchedSource: result.UnmatchedSource,
		UnmatchedTarget: result.UnmatchedTarget,
	}
	for _, f := range result.Fields {
		resp.Fields = append(resp.Fields, api.FieldMapping{
			Source: f.Source,
			Target: f.Target,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// dbTypes that require Connect() for introspection (they need a live DB connection).
var dbTypes = map[string]bool{
	"sqlite3":   true,
	"postgres":  true,
	"sqlserver": true,
}

// expandFilePath resolves ~ to the user's home directory.
func expandFilePath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		usr, err := user.Current()
		if err == nil {
			return filepath.Join(usr.HomeDir, path[2:])
		}
	}
	return path
}

// normalizeConnection expands file paths in connection config.
func normalizeConnection(connection map[string]string) map[string]string {
	out := make(map[string]string, len(connection))
	for k, v := range connection {
		if k == "filepath" && v != "" {
			out[k] = expandFilePath(v)
		} else {
			out[k] = v
		}
	}
	return out
}

// getIntrospector creates a source or target and checks if it implements SchemaIntrospector.
// For DB types, Connect() is called to establish the DB connection.
// For file types (csv, json), Connect() is skipped since introspection reads files directly.
func (h *SchemaHandler) getIntrospector(typeName, role string, connection map[string]string) (etl.SchemaIntrospector, error) {
	connection = normalizeConnection(connection)

	var obj any
	var err error

	if role == "target" {
		obj, err = etl.NewTarget(typeName, "_introspect", connection)
	} else {
		obj, err = etl.NewSource(typeName, "_introspect", connection)
	}
	if err != nil {
		return nil, err
	}

	// Only call Connect() for DB types that need a live connection for introspection.
	if dbTypes[typeName] {
		type connector interface {
			Connect() error
		}
		if cc, ok := obj.(connector); ok {
			if err := cc.Connect(); err != nil {
				return nil, fmt.Errorf("connection failed: %w", err)
			}
		}
	}

	introspector, ok := obj.(etl.SchemaIntrospector)
	if !ok {
		h.closeObj(obj)
		return nil, fmt.Errorf("type %q does not support schema introspection", typeName)
	}

	return introspector, nil
}

func (h *SchemaHandler) closeIntrospector(introspector etl.SchemaIntrospector) {
	h.closeObj(introspector)
}

func (h *SchemaHandler) closeObj(obj any) {
	type closer interface {
		Close() error
	}
	if c, ok := obj.(closer); ok {
		c.Close()
	}
}
