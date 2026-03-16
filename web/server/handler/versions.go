package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"shadow-dom/etlctl/web/server/service"
)

type VersionHandler struct {
	svc *service.VersionService
}

func NewVersionHandler(svc *service.VersionService) *VersionHandler {
	return &VersionHandler{svc: svc}
}

func (h *VersionHandler) List(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versions, err := h.svc.ListVersions(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, versions)
}

func (h *VersionHandler) Save(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Label   string `json:"label"`
		IsDraft bool   `json:"isDraft"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	ver, err := h.svc.SaveVersion(name, req.Label, req.IsDraft)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ver)
}

func (h *VersionHandler) Restore(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionStr := r.PathValue("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid version number")
		return
	}
	if err := h.svc.RestoreVersion(name, version); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
