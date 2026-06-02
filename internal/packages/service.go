package packages

import (
	"github.com/autobrr/brrewery/internal/packages/catalog"
	"github.com/autobrr/brrewery/internal/packages/detect"
	"github.com/autobrr/brrewery/internal/packages/model"
)

type Service struct {
	evaluator *detect.Evaluator
}

func NewService() *Service {
	return &Service{evaluator: detect.NewEvaluator()}
}

func (s *Service) List() []model.PackageStatus {
	all := catalog.All()
	out := make([]model.PackageStatus, 0, len(all))
	for i := range all {
		out = append(out, s.statusFor(&all[i]))
	}
	return out
}

func (s *Service) Get(id string) (model.PackageStatus, bool) {
	pkg, ok := catalog.ByID(id)
	if !ok {
		return model.PackageStatus{}, false
	}
	return s.statusFor(&pkg), true
}

func (s *Service) statusFor(pkg *model.Package) model.PackageStatus {
	installed := s.evaluator.Installed(&pkg.Detection)
	depsOK := s.evaluator.DependenciesSatisfied(pkg.Dependencies, catalog.DetectionSpec)
	return model.PackageStatus{
		Package:               *pkg,
		Installed:             installed,
		DependenciesSatisfied: depsOK,
	}
}
