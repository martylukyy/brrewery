package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/httputil"
	"github.com/autobrr/brrewery/internal/selfupdate"
)

// UpdateChecker reports the newest published release. *selfupdate.Checker
// satisfies it; tests substitute a fake.
type UpdateChecker interface {
	Status() selfupdate.Status
	Refresh(ctx context.Context) (selfupdate.Status, error)
}

// UpdateRunner launches the self-update job and finishes an installed update
// by restarting the service. *selfupdate.Updater satisfies it; tests
// substitute a fake.
type UpdateRunner interface {
	Start(ctx context.Context) (model.Job, error)
	Restart(ctx context.Context) error
	RestartPending() bool
}

// UpdateHandler exposes the self-update check and trigger. Starting an update
// re-verifies the operator's password, the same gate app installs and sysctl
// changes use.
type UpdateHandler struct {
	checker UpdateChecker
	updater UpdateRunner
	auth    *auth.Service
}

func NewUpdateHandler(checker UpdateChecker, updater UpdateRunner, authService *auth.Service) *UpdateHandler {
	return &UpdateHandler{checker: checker, updater: updater, auth: authService}
}

// Status returns the cached release check; ?refresh=1 queries GitHub first. A
// failed refresh still answers 200 with the stale cache plus its error field,
// so the UI can keep rendering the last known state.
func (h *UpdateHandler) Status(w http.ResponseWriter, r *http.Request) {
	if h.checker == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "Update checker not configured")
		return
	}
	if r.URL.Query().Get("refresh") == "1" {
		_, _ = h.checker.Refresh(r.Context())
	}

	status := h.checker.Status()
	if h.updater != nil {
		status.RestartPending = h.updater.RestartPending()
	}
	httputil.WriteJSON(w, http.StatusOK, status)
}

type startUpdateRequest struct {
	Password string `json:"password"`
}

// Start verifies the operator's password and launches the self-update job.
func (h *UpdateHandler) Start(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var body startUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	password := strings.TrimSpace(body.Password)
	if password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Account password is required to install updates")
		return
	}
	if err := h.auth.VerifyPassword(username, password); err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Incorrect password")
		return
	}

	if h.updater == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "Updater not configured")
		return
	}

	job, err := h.updater.Start(r.Context())
	switch {
	case err == nil:
		httputil.WriteJSON(w, http.StatusAccepted, model.InstallResponse{JobID: job.ID})
	case errors.Is(err, selfupdate.ErrUpdateInProgress):
		httputil.WriteError(w, http.StatusConflict, "An update is already in progress")
	case errors.Is(err, selfupdate.ErrNoUpdate):
		httputil.WriteError(w, http.StatusConflict, "No update available")
	case errors.Is(err, selfupdate.ErrUnsupported):
		httputil.WriteError(w, http.StatusNotImplemented, err.Error())
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to start update: "+err.Error())
	}
}

// Restart finishes an installed update by restarting the brrewery service.
// No password re-check here: the install that armed it was already
// password-gated, and the updater refuses unless that install's marker is on
// disk, so this endpoint cannot bounce the service on its own.
func (h *UpdateHandler) Restart(w http.ResponseWriter, r *http.Request) {
	if h.updater == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "Updater not configured")
		return
	}

	err := h.updater.Restart(r.Context())
	switch {
	case err == nil:
		httputil.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "restarting"})
	case errors.Is(err, selfupdate.ErrNoPendingRestart):
		httputil.WriteError(w, http.StatusConflict, "No installed update is waiting for a restart")
	case errors.Is(err, selfupdate.ErrUpdateInProgress):
		httputil.WriteError(w, http.StatusConflict, "An update is still in progress")
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to restart: "+err.Error())
	}
}
