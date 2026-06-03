package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/autobrr/brrewery/internal/httputil"
	pkgdomain "github.com/autobrr/brrewery/internal/packages"
	"github.com/autobrr/brrewery/internal/packages/model"
)

type JobsHandler struct {
	service *pkgdomain.Service
}

func NewJobsHandler(service *pkgdomain.Service) *JobsHandler {
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
