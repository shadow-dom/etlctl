package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"shadow-dom/etlctl/web/server/api"
	"shadow-dom/etlctl/web/server/service"
)

type ETLHandler struct {
	svc *service.ETLService
}

func NewETLHandler(svc *service.ETLService) *ETLHandler {
	return &ETLHandler{svc: svc}
}

func (h *ETLHandler) List(w http.ResponseWriter, r *http.Request) {
	etls, err := h.svc.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, etls)
}

func (h *ETLHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	etl, err := h.svc.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, etl)
}

func (h *ETLHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req api.ETLResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := h.svc.Create(req); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (h *ETLHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req api.ETLResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.svc.Update(name, req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (h *ETLHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.svc.Delete(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ETLHandler) Validate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	result := h.svc.Validate(name)
	writeJSON(w, http.StatusOK, result)
}

func (h *ETLHandler) GetRawYAML(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := filepath.Join(h.svc.ConfigDir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "ETL not found: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (h *ETLHandler) UpdateRawYAML(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := filepath.Join(h.svc.ConfigDir, name+".yaml")

	// Verify the ETL exists
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "ETL not found")
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body: "+err.Error())
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, api.ErrorResponse{Error: msg})
}
