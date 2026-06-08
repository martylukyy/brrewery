package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/httputil"
	pkgdomain "github.com/autobrr/brrewery/internal/packages"
	"github.com/autobrr/brrewery/internal/packages/catalog"
	"github.com/autobrr/brrewery/internal/packages/model"
	"github.com/autobrr/brrewery/internal/packages/qbittorrent"
	"github.com/autobrr/brrewery/internal/packages/secrets"
)

type PackagesHandler struct {
	service *pkgdomain.Service
	auth    *auth.Service
}

func NewPackagesHandler(service *pkgdomain.Service, authService *auth.Service) *PackagesHandler {
	return &PackagesHandler{service: service, auth: authService}
}

type PackageListResponse struct {
	Packages []model.PackageStatus `json:"packages"`
}

func (h *PackagesHandler) List(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, PackageListResponse{Packages: h.service.List(username)})
}

func (h *PackagesHandler) Get(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	pkg, ok := h.service.Get(id, username)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "Package not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, pkg)
}

func (h *PackagesHandler) Install(w http.ResponseWriter, r *http.Request) {
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

	pkg, ok := catalog.ByID(id)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "Package not found")
		return
	}
	if err := secrets.ValidateInstallSecrets(pkg, username, body.ExtraVars, h.auth); err != nil {
		writePackageJobError(w, err)
		return
	}
	if !writeQbittorrentValidation(w, pkg.ID, body.ExtraVars) {
		return
	}

	job, err := h.service.StartInstall(r.Context(), id, username, body.ExtraVars)
	if err != nil {
		writePackageJobError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusAccepted, model.InstallResponse{JobID: job.ID})
}

func (h *PackagesHandler) Upgrade(w http.ResponseWriter, r *http.Request) {
	h.startPackageJob(w, r, true, h.service.StartUpgrade)
}

func (h *PackagesHandler) Remove(w http.ResponseWriter, r *http.Request) {
	h.startPackageJob(w, r, false, h.service.StartRemove)
}

type packageJobStarter func(context.Context, string, string, map[string]string) (model.Job, error)

func (h *PackagesHandler) startPackageJob(w http.ResponseWriter, r *http.Request, validateOptions bool, start packageJobStarter) {
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
		writePackageJobError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusAccepted, model.InstallResponse{JobID: job.ID})
}

// writeQbittorrentValidation validates qBittorrent install options and reports
// whether processing may continue. On failure it writes the HTTP error.
func writeQbittorrentValidation(w http.ResponseWriter, packageID string, extraVars map[string]string) bool {
	err := qbittorrent.Validate(packageID, extraVars)
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

func writePackageJobError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pkgdomain.ErrPackageNotFound):
		httputil.WriteError(w, http.StatusNotFound, "Package not found")
	case errors.Is(err, pkgdomain.ErrAlreadyInstalled):
		httputil.WriteError(w, http.StatusConflict, "Package already installed")
	case errors.Is(err, pkgdomain.ErrNotInstalled):
		httputil.WriteError(w, http.StatusConflict, "Package not installed")
	case errors.Is(err, pkgdomain.ErrDependenciesNotMet):
		httputil.WriteError(w, http.StatusConflict, "Package dependencies not satisfied")
	case errors.Is(err, pkgdomain.ErrPlaybookMissing):
		httputil.WriteError(w, http.StatusInternalServerError, "Playbook not available")
	case errors.Is(err, pkgdomain.ErrInstallUserMissing):
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
	case errors.Is(err, secrets.ErrInstallSecretMissing):
		httputil.WriteError(w, http.StatusBadRequest, "Required install credentials missing")
	case errors.Is(err, secrets.ErrInstallSecretInvalid):
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid install credentials")
	default:
		httputil.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
