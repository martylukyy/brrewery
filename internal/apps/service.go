package apps

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/ansible"
	"github.com/autobrr/brrewery/internal/apps/catalog"
	"github.com/autobrr/brrewery/internal/apps/deluge"
	"github.com/autobrr/brrewery/internal/apps/detect"
	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/jobs"
	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/autobrr/brrewery/internal/apps/qbittorrent"
	"github.com/autobrr/brrewery/internal/apps/rtorrent"
)

var (
	ErrAppNotFound        = errors.New("app not found")
	ErrAlreadyInstalled   = errors.New("app already installed")
	ErrNotInstalled       = errors.New("app not installed")
	ErrDependenciesNotMet = errors.New("app dependencies not satisfied")
	ErrPlaybookMissing    = errors.New("playbook not found")
	ErrInstallUserMissing = errors.New("install user is required")
	ErrNoService          = errors.New("app has no controllable service")
)

type PlaybookRunner interface {
	Run(ctx context.Context, req ansible.RunRequest) error
}

type Service struct {
	evaluator  *detect.Evaluator
	runner     PlaybookRunner
	jobs       *jobs.Store
	controller serviceController
}

func NewService() *Service {
	return NewServiceWithDeps(detect.NewEvaluator(), nil, jobs.NewStore())
}

func NewServiceWithDeps(evaluator *detect.Evaluator, runner PlaybookRunner, store *jobs.Store) *Service {
	if store == nil {
		store = jobs.NewStore()
	}
	return &Service{
		evaluator:  evaluator,
		runner:     runner,
		jobs:       store,
		controller: systemctlController{},
	}
}

func (s *Service) List(username string) []model.AppStatus {
	all := catalog.All()
	out := make([]model.AppStatus, 0, len(all))
	for i := range all {
		out = append(out, s.statusFor(&all[i], username))
	}
	return out
}

func (s *Service) Get(id, username string) (model.AppStatus, bool) {
	app, ok := catalog.ByID(id)
	if !ok {
		return model.AppStatus{}, false
	}
	return s.statusFor(&app, username), true
}

func (s *Service) GetJob(id string) (model.Job, bool) {
	return s.jobs.Get(id)
}

func (s *Service) JobLogs(id string) ([]string, bool) {
	return s.jobs.Logs(id)
}

func (s *Service) StartInstall(ctx context.Context, id, username string, extraVars map[string]string) (model.Job, error) {
	return s.startJob(ctx, model.JobActionInstall, id, username, extraVars)
}

func (s *Service) StartUpgrade(ctx context.Context, id, username string, extraVars map[string]string) (model.Job, error) {
	return s.startJob(ctx, model.JobActionUpgrade, id, username, extraVars)
}

func (s *Service) StartRemove(ctx context.Context, id, username string, extraVars map[string]string) (model.Job, error) {
	return s.startJob(ctx, model.JobActionRemove, id, username, extraVars)
}

func (s *Service) startJob(
	ctx context.Context,
	action model.JobAction,
	id, username string,
	extraVars map[string]string,
) (model.Job, error) {
	if s.runner == nil {
		return model.Job{}, errors.New("app runner not configured")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return model.Job{}, ErrInstallUserMissing
	}

	app, ok := catalog.ByID(id)
	if !ok {
		return model.Job{}, ErrAppNotFound
	}

	status := s.statusFor(&app, username)
	switch action {
	case model.JobActionInstall:
		if status.Installed {
			return model.Job{}, ErrAlreadyInstalled
		}
		if !status.DependenciesSatisfied {
			return model.Job{}, ErrDependenciesNotMet
		}
	case model.JobActionUpgrade, model.JobActionRemove:
		if !status.Installed {
			return model.Job{}, ErrNotInstalled
		}
	}

	playbookPath := strings.TrimSpace(playbookForAction(&app, action))
	if playbookPath == "" {
		return model.Job{}, ErrPlaybookMissing
	}
	if _, err := os.Stat(playbookPath); err != nil {
		return model.Job{}, ErrPlaybookMissing
	}

	job := s.jobs.Create(id, action)
	jobID := job.ID
	vars := extravars.ForInstall(username, extraVars)
	if err := enrichAppVars(ctx, id, action, vars); err != nil {
		s.jobs.SetStatus(jobID, model.JobStatusFailed, err.Error())
		return model.Job{}, err
	}

	go s.runJob(context.Background(), jobID, action, id, username, playbookPath, vars)

	return job, nil
}

func (s *Service) runJob(
	ctx context.Context,
	jobID string,
	action model.JobAction,
	appID, username, playbookPath string,
	extraVars map[string]string,
) {
	s.jobs.MarkRunning(jobID)

	err := s.runner.Run(ctx, ansible.RunRequest{
		PlaybookPath: playbookPath,
		ExtraVars:    extraVars,
		OnOutput: func(line string) {
			s.jobs.AppendLog(jobID, line)
		},
	})
	if err != nil {
		s.jobs.SetStatus(jobID, model.JobStatusFailed, err.Error())
		return
	}

	app, ok := catalog.ByID(appID)
	if !ok {
		s.jobs.SetStatus(jobID, model.JobStatusFailed, "app not found after job")
		return
	}

	detectUser := username
	if u := strings.TrimSpace(extraVars[extravars.BrreweryUser]); u != "" {
		detectUser = u
	}
	installed := s.statusFor(&app, detectUser).Installed
	switch action {
	case model.JobActionInstall, model.JobActionUpgrade:
		if !installed {
			s.jobs.SetStatus(jobID, model.JobStatusFailed, string(action)+" finished but app was not detected on the system")
			return
		}
	case model.JobActionRemove:
		if installed {
			s.jobs.SetStatus(jobID, model.JobStatusFailed, "remove finished but app is still detected on the system")
			return
		}
	}

	s.jobs.SetStatus(jobID, model.JobStatusSucceeded, "")
}

func playbookForAction(app *model.App, action model.JobAction) string {
	switch action {
	case model.JobActionInstall:
		return app.Playbooks.Install
	case model.JobActionUpgrade:
		return app.Playbooks.Upgrade
	case model.JobActionRemove:
		return app.Playbooks.Remove
	default:
		return ""
	}
}

func enrichAppVars(ctx context.Context, appID string, action model.JobAction, vars map[string]string) error {
	if action != model.JobActionInstall && action != model.JobActionUpgrade {
		return nil
	}
	switch appID {
	case qbittorrent.AppID:
		return qbittorrent.EnrichAnsibleVars(ctx, vars, nil)
	case rtorrent.AppID:
		return rtorrent.EnrichAnsibleVars(ctx, vars, nil)
	case deluge.AppID:
		return deluge.EnrichAnsibleVars(ctx, vars, nil)
	default:
		return nil
	}
}

// SetServiceEnabled starts & enables (on) or stops & disables (off) the
// installed app's systemd unit(s), returning the refreshed service state. The
// caller is responsible for authenticating the operator; brrewery runs the
// transition directly as root.
func (s *Service) SetServiceEnabled(ctx context.Context, id, username string, on bool) (model.ServiceStatus, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return model.ServiceStatus{}, ErrInstallUserMissing
	}

	app, ok := catalog.ByID(id)
	if !ok {
		return model.ServiceStatus{}, ErrAppNotFound
	}

	status := s.statusFor(&app, username)
	if !status.Installed {
		return model.ServiceStatus{}, ErrNotInstalled
	}
	if status.Service == nil {
		return model.ServiceStatus{}, ErrNoService
	}

	if err := s.controller.SetEnabled(ctx, status.Service.Units, on); err != nil {
		return model.ServiceStatus{}, err
	}

	refreshed, _ := s.evaluator.ServiceStatus(&app.Detection, username)
	return refreshed, nil
}

func (s *Service) statusFor(app *model.App, username string) model.AppStatus {
	installed := s.evaluator.InstalledForUser(&app.Detection, username)
	depsOK := s.evaluator.DependenciesSatisfied(username, app.Dependencies, catalog.DetectionSpec)
	status := model.AppStatus{
		App:                   *app,
		Installed:             installed,
		DependenciesSatisfied: depsOK,
	}
	// Only installed apps expose a live service toggle; surfacing it for an
	// uninstalled app would query units that aren't there yet.
	if installed {
		if svc, ok := s.evaluator.ServiceStatus(&app.Detection, username); ok {
			status.Service = &svc
		}
	}
	return status
}
