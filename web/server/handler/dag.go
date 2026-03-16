package handler

import (
	"encoding/json"
	"net/http"

	"shadow-dom/etlctl/web/server/api"
	"shadow-dom/etlctl/web/server/service"
)

type DAGHandler struct {
	svc *service.DAGService
}

func NewDAGHandler(svc *service.DAGService) *DAGHandler {
	return &DAGHandler{svc: svc}
}

func (h *DAGHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	meta, err := h.svc.Get(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *DAGHandler) Save(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var meta api.DAGMeta
	if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.svc.Save(name, &meta); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
