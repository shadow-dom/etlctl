package handler

import (
	"net/http"

	"shadow-dom/etlctl/web/server/service"
)

type RunHandler struct {
	runner *service.RunnerService
}

func NewRunHandler(runner *service.RunnerService) *RunHandler {
	return &RunHandler{runner: runner}
}

func (h *RunHandler) RunETL(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	result, err := h.runner.TestETL(name, false, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *RunHandler) TestETL(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dryRun := r.URL.Query().Get("dryRun") != "false"
	resetState := r.URL.Query().Get("resetState") == "true"
	result, err := h.runner.TestETL(name, dryRun, resetState)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
