package packages

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/autobrr/brrewery/internal/packages/ansible"
	"github.com/autobrr/brrewery/internal/packages/catalog"
	"github.com/autobrr/brrewery/internal/packages/detect"
	"github.com/autobrr/brrewery/internal/packages/extravars"
	"github.com/autobrr/brrewery/internal/packages/jobs"
	"github.com/autobrr/brrewery/internal/packages/model"
	"github.com/autobrr/brrewery/internal/packages/qbittorrent"
)

var (
	ErrPackageNotFound    = errors.New("package not found")
	ErrAlreadyInstalled   = errors.New("package already installed")
	ErrNotInstalled       = errors.New("package not installed")
	ErrDependenciesNotMet = errors.New("package dependencies not satisfied")
	ErrPlaybookMissing    = errors.New("playbook not found")
	ErrInstallUserMissing = errors.New("install user is required")
)

type PlaybookRunner interface {
	Run(ctx context.Context, req ansible.RunRequest) error
}

type Service struct {
	evaluator *detect.Evaluator
	runner    PlaybookRunner
	jobs      *jobs.Store
}

func NewService() *Service {
	return NewServiceWithDeps(detect.NewEvaluator(), nil, jobs.NewStore())
}

func NewServiceWithDeps(evaluator *detect.Evaluator, runner PlaybookRunner, store *jobs.Store) *Service {
	if store == nil {
		store = jobs.NewStore()
	}
	return &Service{
		evaluator: evaluator,
		runner:    runner,
		jobs:      store,
	}
}

func (s *Service) List(username string) []model.PackageStatus {
	all := catalog.All()
	out := make([]model.PackageStatus, 0, len(all))
	for i := range all {
		out = append(out, s.statusFor(&all[i], username))
	}
	return out
}

func (s *Service) Get(id, username string) (model.PackageStatus, bool) {
	pkg, ok := catalog.ByID(id)
	if !ok {
		return model.PackageStatus{}, false
	}
	return s.statusFor(&pkg, username), true
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
		return model.Job{}, errors.New("package runner not configured")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return model.Job{}, ErrInstallUserMissing
	}

	pkg, ok := catalog.ByID(id)
	if !ok {
		return model.Job{}, ErrPackageNotFound
	}

	status := s.statusFor(&pkg, username)
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

	playbookPath := strings.TrimSpace(playbookForAction(&pkg, action))
	if playbookPath == "" {
		return model.Job{}, ErrPlaybookMissing
	}
	if _, err := os.Stat(playbookPath); err != nil {
		return model.Job{}, ErrPlaybookMissing
	}

	job := s.jobs.Create(id, action)
	jobID := job.ID
	vars := extravars.ForInstall(username, extraVars)
	if err := enrichPackageVars(ctx, id, action, vars); err != nil {
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
	packageID, username, playbookPath string,
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

	pkg, ok := catalog.ByID(packageID)
	if !ok {
		s.jobs.SetStatus(jobID, model.JobStatusFailed, "package not found after job")
		return
	}

	detectUser := username
	if u := strings.TrimSpace(extraVars[extravars.BrreweryUser]); u != "" {
		detectUser = u
	}
	installed := s.statusFor(&pkg, detectUser).Installed
	switch action {
	case model.JobActionInstall, model.JobActionUpgrade:
		if !installed {
			s.jobs.SetStatus(jobID, model.JobStatusFailed, string(action)+" finished but package was not detected on the system")
			return
		}
	case model.JobActionRemove:
		if installed {
			s.jobs.SetStatus(jobID, model.JobStatusFailed, "remove finished but package is still detected on the system")
			return
		}
	}

	s.jobs.SetStatus(jobID, model.JobStatusSucceeded, "")
}

func playbookForAction(pkg *model.Package, action model.JobAction) string {
	switch action {
	case model.JobActionInstall:
		return pkg.Playbooks.Install
	case model.JobActionUpgrade:
		return pkg.Playbooks.Upgrade
	case model.JobActionRemove:
		return pkg.Playbooks.Remove
	default:
		return ""
	}
}

func enrichPackageVars(ctx context.Context, packageID string, action model.JobAction, vars map[string]string) error {
	if packageID != qbittorrent.PackageID {
		return nil
	}
	switch action {
	case model.JobActionInstall, model.JobActionUpgrade:
		return qbittorrent.EnrichAnsibleVars(ctx, vars, nil, nil, nil, nil, nil)
	default:
		return nil
	}
}

func (s *Service) statusFor(pkg *model.Package, username string) model.PackageStatus {
	installed := s.evaluator.InstalledForUser(&pkg.Detection, username)
	depsOK := s.evaluator.DependenciesSatisfied(username, pkg.Dependencies, catalog.DetectionSpec)
	return model.PackageStatus{
		Package:                 *pkg,
		Installed:               installed,
		DependenciesSatisfied: depsOK,
	}
}
