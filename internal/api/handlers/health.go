package handlers

import (
	"net/http"

	"github.com/autobrr/brrewery/internal/httputil"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

type HealthResponse struct {
	Status string `json:"status"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}
