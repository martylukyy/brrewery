package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPasswordEndpoint(t *testing.T) {
	t.Parallel()

	client, baseURL := newLoggedInClient(t)
	endpoint := baseURL + "/api/v1/auth/verify-password"

	t.Run("correct password returns 204", func(t *testing.T) {
		res := postJSON(t, client, endpoint, map[string]any{"password": "password123"})
		defer res.Body.Close()
		require.Equal(t, http.StatusNoContent, res.StatusCode)
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		res := postJSON(t, client, endpoint, map[string]any{"password": "wrong-password"})
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("missing password returns 400", func(t *testing.T) {
		res := postJSON(t, client, endpoint, map[string]any{})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}
