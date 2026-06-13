package secrets

import (
	"errors"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/model"
)

var (
	ErrInstallSecretMissing = errors.New("required install secret missing")
	ErrInstallSecretInvalid = errors.New("invalid install secret")
)

type BrreweryPasswordVerifier interface {
	VerifyPassword(username, password string) error
}

func ValidateInstallSecrets(app model.App, username string, extraVars map[string]string, verifier BrreweryPasswordVerifier) error {
	for _, spec := range app.InstallSecrets {
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

func RequiredKeys(apps []model.App) []model.InstallSecret {
	seen := make(map[string]struct{})
	out := make([]model.InstallSecret, 0)
	for _, app := range apps {
		for _, spec := range app.InstallSecrets {
			if _, ok := seen[spec.Key]; ok {
				continue
			}
			seen[spec.Key] = struct{}{}
			out = append(out, spec)
		}
	}
	return out
}
