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
	Username string `json:"username"`
	Password string `json:"password"`
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

	user, err := h.auth.Login(r.Context(), req.Username, req.Password)
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
