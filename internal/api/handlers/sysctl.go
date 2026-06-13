package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/ansible"
	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/httputil"
	"github.com/autobrr/brrewery/internal/paths"
	"github.com/autobrr/brrewery/internal/system"
)

// PlaybookRunner executes an Ansible playbook. *ansible.Runner satisfies it;
// tests substitute a fake to capture the run request.
type PlaybookRunner interface {
	Run(ctx context.Context, req ansible.RunRequest) error
}

// SysctlHandler reads the curated kernel-tuning catalog and applies changes by
// running a privileged Ansible playbook, escalating with the operator's password
// the same way app installs do.
type SysctlHandler struct {
	runner       PlaybookRunner
	auth         *auth.Service
	playbookPath string
}

func NewSysctlHandler(runner PlaybookRunner, authService *auth.Service) *SysctlHandler {
	return &SysctlHandler{
		runner:       runner,
		auth:         authService,
		playbookPath: filepath.Join(paths.ResolveAnsibleRoot(), "playbooks", "system", "sysctl.yml"),
	}
}

// Get returns the tuning catalog with each parameter's current live value.
func (h *SysctlHandler) Get(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, system.ReadSysctl())
}

type applySysctlRequest struct {
	Values   map[string]string `json:"values"`
	Password string            `json:"password"`
}

// Apply validates the requested values against the catalog, verifies the
// operator's password, then writes and reloads the brrewery-managed sysctl
// drop-in via Ansible. On success it returns the refreshed report so the UI
// reflects the new live values.
func (h *SysctlHandler) Apply(w http.ResponseWriter, r *http.Request) {
	username, ok := h.auth.Username(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var body applySysctlRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !system.ReadSysctl().Writable {
		httputil.WriteError(w, http.StatusNotImplemented, "Tuning sysctl parameters is not supported on this platform")
		return
	}

	sanitized, err := system.ValidateSysctl(body.Values)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	password := strings.TrimSpace(body.Password)
	if password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "Account password is required to apply changes")
		return
	}
	if err := h.auth.VerifyPassword(username, password); err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Incorrect password")
		return
	}

	if h.runner == nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Playbook runner not configured")
		return
	}

	var output []string
	runErr := h.runner.Run(r.Context(), ansible.RunRequest{
		PlaybookPath: h.playbookPath,
		ExtraVars: map[string]string{
			// Kept off argv and out of the extra-vars JSON by the runner, which
			// passes it via --become-password-file.
			extravars.BecomePassword: password,
			"sysctl_conf_content":    system.SysctlConfContent(sanitized),
			"sysctl_conf_path":       system.SysctlConfPath,
		},
		OnOutput: func(line string) {
			if line = strings.TrimSpace(line); line != "" {
				output = append(output, line)
			}
		},
	})
	if runErr != nil {
		httputil.WriteError(w, http.StatusInternalServerError, sysctlApplyErrorMessage(runErr, output))
		return
	}

	httputil.WriteJSON(w, http.StatusOK, system.ReadSysctl())
}

// sysctlApplyErrorMessage prefers the last line of playbook output, which is the
// failing task's message and far more useful than the runner's wrapped "exit
// status N", falling back to the raw error when there was no output.
func sysctlApplyErrorMessage(err error, output []string) string {
	if len(output) > 0 {
		return "Failed to apply sysctl settings: " + output[len(output)-1]
	}
	return "Failed to apply sysctl settings: " + err.Error()
}
