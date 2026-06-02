package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/autobrr/brrewery/internal/httputil"
	pkgdomain "github.com/autobrr/brrewery/internal/packages"
	"github.com/autobrr/brrewery/internal/packages/model"
)

type PackagesHandler struct {
	service *pkgdomain.Service
}

func NewPackagesHandler(service *pkgdomain.Service) *PackagesHandler {
	return &PackagesHandler{service: service}
}

type PackageListResponse struct {
	Packages []model.PackageStatus `json:"packages"`
}

func (h *PackagesHandler) List(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, PackageListResponse{Packages: h.service.List()})
}

func (h *PackagesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pkg, ok := h.service.Get(id)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "Package not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, pkg)
}
