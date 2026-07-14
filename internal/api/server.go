package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/autobrr/brrewery/internal/api/handlers"
	"github.com/autobrr/brrewery/internal/api/middleware"
	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/httputil"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
	webapp "github.com/autobrr/brrewery/internal/web"
)

type Server struct {
	logger         zerolog.Logger
	authService    *auth.Service
	sessionManager *scs.SessionManager
	apps           *appsdomain.Service
	system         *system.Collector
	vnstat         *vnstat.Collector
	sysctlRunner   handlers.PlaybookRunner
	updateChecker  handlers.UpdateChecker
	updateStarter  handlers.UpdateStarter
	embedFS        fs.FS
}

func NewServer(
	logger *zerolog.Logger,
	authService *auth.Service,
	sessionManager *scs.SessionManager,
	appsService *appsdomain.Service,
	systemCollector *system.Collector,
	vnstatCollector *vnstat.Collector,
	sysctlRunner handlers.PlaybookRunner,
	updateChecker handlers.UpdateChecker,
	updateStarter handlers.UpdateStarter,
	embedFS fs.FS,
) *Server {
	return &Server{
		logger:         *logger,
		authService:    authService,
		sessionManager: sessionManager,
		apps:           appsService,
		system:         systemCollector,
		vnstat:         vnstatCollector,
		sysctlRunner:   sysctlRunner,
		updateChecker:  updateChecker,
		updateStarter:  updateStarter,
		embedFS:        embedFS,
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(s.sessionManager.LoadAndSave)
	r.Use(secureCookieMiddleware)

	health := handlers.NewHealthHandler()
	r.Get("/health", health.Health)

	r.Route("/api/v1", func(r chi.Router) {
		authHandler := handlers.NewAuthHandler(s.authService)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(s.authService))
			r.Post("/auth/logout", authHandler.Logout)
			r.Post("/auth/verify-password", authHandler.VerifyPassword)
			r.Get("/auth/me", authHandler.Me)

			version := handlers.NewVersionHandler()
			r.Get("/version", version.Version)

			apps := handlers.NewAppsHandler(s.apps, s.authService)
			r.Get("/apps", apps.List)
			r.Get("/apps/{id}", apps.Get)
			r.Post("/apps/{id}/install", apps.Install)
			r.Post("/apps/{id}/upgrade", apps.Upgrade)
			r.Post("/apps/{id}/remove", apps.Remove)
			r.Post("/apps/{id}/service", apps.SetService)

			jobsHandler := handlers.NewJobsHandler(s.apps)
			r.Get("/jobs/{id}", jobsHandler.Get)
			r.Get("/jobs/{id}/logs", jobsHandler.Logs)

			sys := handlers.NewSystemHandler(s.system)
			r.Get("/system", sys.Get)

			sysctl := handlers.NewSysctlHandler(s.sysctlRunner, s.authService)
			r.Get("/system/sysctl", sysctl.Get)
			r.Post("/system/sysctl", sysctl.Apply)

			update := handlers.NewUpdateHandler(s.updateChecker, s.updateStarter, s.authService)
			r.Get("/update", update.Status)
			r.Post("/update", update.Start)

			vn := handlers.NewVnstatHandler(s.vnstat)
			r.Get("/traffic/vnstat", vn.Get)
		})
	})

	if s.embedFS != nil {
		webHandler := webapp.NewHandler(s.embedFS)
		spa := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api/") {
				httputil.WriteError(w, http.StatusNotFound, "Not found")
				return
			}
			webHandler.ServeSPA(w, req)
		})
		// Register GET and HEAD explicitly: chi does not route HEAD to a GET
		// handler, so without this an unknown path answers HEAD with a 405
		// instead of the 404 it returns for GET.
		r.Method(http.MethodGet, "/*", spa)
		r.Method(http.MethodHead, "/*", spa)
	}

	return r
}

func secureCookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			r = r.WithContext(r.Context())
		}
		next.ServeHTTP(w, r)
	})
}
