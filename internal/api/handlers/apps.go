package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/apps/catalog"
	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/autobrr/brrewery/internal/apps/qbittorrent"
	"github.com/autobrr/brrewery/internal/apps/secrets"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/httputil"
)

type AppsHandler struct {
	service *appsdomain.Service
	auth    *auth.Service
}

func NewAppsHandler(service *appsdomain.Service, authService *auth.Service) *AppsHandler {
	return &AppsHandler{service: service, auth: authService}
}

type AppListResponse struct {
	Apps []model.AppStatus `json:"apps"`
}

func (h *AppsHandler) List(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, AppListResponse{Apps: h.service.List(username)})
}

func (h *AppsHandler) Get(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	app, ok := h.service.Get(id, username)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "App not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, app)
}

func (h *AppsHandler) Install(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var body model.InstallRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			httputil.WriteError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	}

	app, ok := catalog.ByID(id)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "App not found")
		return
	}
	if err := secrets.ValidateInstallSecrets(app, username, body.ExtraVars, h.auth); err != nil {
		writeAppJobError(w, err)
		return
	}
	if !writeQbittorrentValidation(w, app.ID, body.ExtraVars) {
		return
	}

	job, err := h.service.StartInstall(r.Context(), id, username, body.ExtraVars)
	if err != nil {
		writeAppJobError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusAccepted, model.InstallResponse{JobID: job.ID})
}

type setServiceRequest struct {
	Enabled  bool   `json:"enabled"`
	Password string `json:"password"`
}

// SetService starts & enables or stops & disables an installed app's systemd
// service. The operator's account password is required (and verified) as a
// confirmation gate before the privileged transition runs.
func (h *AppsHandler) SetService(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var body setServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	password := strings.TrimSpace(body.Password)
	if password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Account password is required")
		return
	}
	if err := h.auth.VerifyPassword(username, password); err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Incorrect password")
		return
	}

	status, err := h.service.SetServiceEnabled(r.Context(), id, username, body.Enabled)
	if err != nil {
		writeAppJobError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, status)
}

func (h *AppsHandler) Upgrade(w http.ResponseWriter, r *http.Request) {
	h.startAppJob(w, r, true, h.service.StartUpgrade)
}

func (h *AppsHandler) Remove(w http.ResponseWriter, r *http.Request) {
	h.startAppJob(w, r, false, h.service.StartRemove)
}

type appJobStarter func(context.Context, string, string, map[string]string) (model.Job, error)

func (h *AppsHandler) startAppJob(w http.ResponseWriter, r *http.Request, validateOptions bool, start appJobStarter) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	var body model.InstallRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			httputil.WriteError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	}

	if validateOptions && !writeQbittorrentValidation(w, id, body.ExtraVars) {
		return
	}

	job, err := start(r.Context(), id, username, body.ExtraVars)
	if err != nil {
		writeAppJobError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusAccepted, model.InstallResponse{JobID: job.ID})
}

// writeQbittorrentValidation validates qBittorrent install options and reports
// whether processing may continue. On failure it writes the HTTP error.
func writeQbittorrentValidation(w http.ResponseWriter, appID string, extraVars map[string]string) bool {
	err := qbittorrent.Validate(appID, extraVars)
	switch {
	case err == nil:
		return true
	case errors.Is(err, qbittorrent.ErrManifestUnavailable):
		httputil.WriteError(w, http.StatusInternalServerError, "qBittorrent build manifest unavailable")
	default:
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	}
	return false
}

func writeAppJobError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, appsdomain.ErrAppNotFound):
		httputil.WriteError(w, http.StatusNotFound, "App not found")
	case errors.Is(err, appsdomain.ErrAlreadyInstalled):
		httputil.WriteError(w, http.StatusConflict, "App already installed")
	case errors.Is(err, appsdomain.ErrNotInstalled):
		httputil.WriteError(w, http.StatusConflict, "App not installed")
	case errors.Is(err, appsdomain.ErrNoService):
		httputil.WriteError(w, http.StatusConflict, "App has no controllable service")
	case errors.Is(err, appsdomain.ErrDependenciesNotMet):
		httputil.WriteError(w, http.StatusConflict, "App dependencies not satisfied")
	case errors.Is(err, appsdomain.ErrPlaybookMissing):
		httputil.WriteError(w, http.StatusInternalServerError, "Playbook not available")
	case errors.Is(err, appsdomain.ErrInstallUserMissing):
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
	case errors.Is(err, secrets.ErrInstallSecretMissing):
		httputil.WriteError(w, http.StatusBadRequest, "Required install credentials missing")
	case errors.Is(err, secrets.ErrInstallSecretInvalid):
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid install credentials")
	default:
		httputil.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
