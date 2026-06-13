package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/autobrr/brrewery/internal/httputil"
)

type JobsHandler struct {
	service *appsdomain.Service
}

func NewJobsHandler(service *appsdomain.Service) *JobsHandler {
	return &JobsHandler{service: service}
}

func (h *JobsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, ok := h.service.GetJob(id)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "Job not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, job)
}

func (h *JobsHandler) Logs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	lines, ok := h.service.JobLogs(id)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "Job not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, model.JobLogsResponse{Lines: lines})
}
