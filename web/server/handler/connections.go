package handler

import (
	"encoding/json"
	"net/http"

	"shadow-dom/etlctl/web/server/service"
)

type ConnectionHandler struct {
	svc *service.ConnectionService
}

func NewConnectionHandler(svc *service.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{svc: svc}
}

func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) {
	conns, err := h.svc.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conns)
}

func (h *ConnectionHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	conn, err := h.svc.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conn)
}

func (h *ConnectionHandler) Save(w http.ResponseWriter, r *http.Request) {
	var conn service.Connection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if conn.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := h.svc.Save(conn); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conn)
}

func (h *ConnectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.svc.Delete(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
