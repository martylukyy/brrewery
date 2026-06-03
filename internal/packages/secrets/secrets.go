package secrets

import (
	"errors"
	"strings"

	"github.com/autobrr/brrewery/internal/packages/model"
)

var (
	ErrInstallSecretMissing = errors.New("required install secret missing")
	ErrInstallSecretInvalid = errors.New("invalid install secret")
)

type BrreweryPasswordVerifier interface {
	VerifyPassword(username, password string) error
}

func ValidateInstallSecrets(pkg model.Package, username string, extraVars map[string]string, verifier BrreweryPasswordVerifier) error {
	for _, spec := range pkg.InstallSecrets {
		value := strings.TrimSpace(extraVars[spec.Key])
		if value == "" {
			return ErrInstallSecretMissing
		}
		if spec.VerifyBrreweryPassword {
			if verifier == nil {
				return ErrInstallSecretInvalid
			}
			if err := verifier.VerifyPassword(username, value); err != nil {
				return ErrInstallSecretInvalid
			}
		}
	}
	return nil
}

func RequiredKeys(packages []model.Package) []model.InstallSecret {
	seen := make(map[string]struct{})
	out := make([]model.InstallSecret, 0)
	for _, pkg := range packages {
		for _, spec := range pkg.InstallSecrets {
			if _, ok := seen[spec.Key]; ok {
				continue
			}
			seen[spec.Key] = struct{}{}
			out = append(out, spec)
		}
	}
	return out
}
