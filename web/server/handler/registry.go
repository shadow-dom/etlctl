package handler

import (
	"encoding/json"
	"net/http"

	"shadow-dom/etlctl/pkg/util/etl"
	"shadow-dom/etlctl/pkg/util/etl/transform"
	"shadow-dom/etlctl/web/server/api"
)

type RegistryHandler struct{}

func NewRegistryHandler() *RegistryHandler {
	return &RegistryHandler{}
}

func (h *RegistryHandler) SourceTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.RegistryResponse{
		Types: etl.RegisteredSourceTypes(),
	})
}

func (h *RegistryHandler) TargetTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.RegistryResponse{
		Types: etl.RegisteredTargetTypes(),
	})
}

func (h *RegistryHandler) RegisteredFunctions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.RegistryResponse{
		Types: etl.RegisteredFunctions(),
	})
}

func (h *RegistryHandler) Transforms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.TransformRegistryResponse{
		Simple:        transform.RegisteredTransforms(),
		Parameterized: transform.RegisteredParamTransforms(),
	})
}

func (h *RegistryHandler) PreviewTransform(w http.ResponseWriter, r *http.Request) {
	var req api.TransformPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := transform.Apply(req.Expression, req.Value)
	resp := api.TransformPreviewResponse{Result: result}
	if err != nil {
		resp.Error = err.Error()
	}

	writeJSON(w, http.StatusOK, resp)
}
