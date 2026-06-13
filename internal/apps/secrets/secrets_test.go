package secrets_test

import (
	"errors"
	"testing"

	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/autobrr/brrewery/internal/apps/secrets"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubVerifier struct {
	err error
}

func (s stubVerifier) VerifyPassword(_, _ string) error {
	return s.err
}

func TestValidateInstallSecrets(t *testing.T) {
	t.Parallel()

	app := model.App{
		InstallSecrets: []model.InstallSecret{{
			Key:                    "brrewery_user_password",
			VerifyBrreweryPassword: true,
		}},
	}

	t.Run("missing secret", func(t *testing.T) {
		t.Parallel()
		err := secrets.ValidateInstallSecrets(app, "admin", nil, stubVerifier{})
		require.ErrorIs(t, err, secrets.ErrInstallSecretMissing)
	})

	t.Run("invalid password", func(t *testing.T) {
		t.Parallel()
		err := secrets.ValidateInstallSecrets(app, "admin", map[string]string{
			"brrewery_user_password": "wrong",
		}, stubVerifier{err: auth.ErrInvalidPassword})
		require.ErrorIs(t, err, secrets.ErrInstallSecretInvalid)
	})

	t.Run("valid password", func(t *testing.T) {
		t.Parallel()
		err := secrets.ValidateInstallSecrets(app, "admin", map[string]string{
			"brrewery_user_password": "password123",
		}, stubVerifier{})
		require.NoError(t, err)
	})
}

func TestRequiredKeys(t *testing.T) {
	t.Parallel()

	specs := secrets.RequiredKeys([]model.App{
		{InstallSecrets: []model.InstallSecret{{Key: "brrewery_user_password", Label: "Password"}}},
		{InstallSecrets: []model.InstallSecret{{Key: "brrewery_user_password", Label: "Password"}}},
	})
	require.Len(t, specs, 1)
	assert.Equal(t, "brrewery_user_password", specs[0].Key)
}

func TestValidateInstallSecrets_NoSecretsConfigured(t *testing.T) {
	t.Parallel()

	err := secrets.ValidateInstallSecrets(model.App{}, "admin", nil, stubVerifier{err: errors.New("unused")})
	require.NoError(t, err)
}
