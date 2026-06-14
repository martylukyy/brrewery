package handlers_test

import (
	"encoding/json"
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

func TestMeEndpoint(t *testing.T) {
	t.Parallel()

	client, baseURL := newLoggedInClient(t)
	endpoint := baseURL + "/api/v1/auth/me"

	t.Run("returns the signed-in username", func(t *testing.T) {
		res, err := client.Get(endpoint)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var body struct {
			Username string `json:"username"`
		}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
		require.Equal(t, "admin", body.Username)
	})

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		res, err := http.Get(endpoint)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})
}
