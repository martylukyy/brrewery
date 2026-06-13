package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/httputil"
)

type AuthHandler struct {
	auth *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{auth: authService}
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type LoginResponse struct {
	Username string `json:"username"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	user, err := h.auth.Login(r.Context(), req.Username, req.Password, req.RememberMe)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			httputil.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "Login failed")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, LoginResponse{Username: user.Username})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Logout(r.Context()); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Logout failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
}

type VerifyPasswordRequest struct {
	Password string `json:"password"`
}

// VerifyPassword checks a candidate password against the session user's account
// without any side effects. It backs the install password prompt's inline check:
// the Linux user, sudo (become) and brrewery dashboard passwords are all the same
// value, so verifying against the brrewery account confirms the operator entered
// the right credential before the install starts.
func (h *AuthHandler) VerifyPassword(w http.ResponseWriter, r *http.Request) {
	var req VerifyPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Password is required")
		return
	}

	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.auth.VerifyPassword(username, req.Password); err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			httputil.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "Password verification failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
