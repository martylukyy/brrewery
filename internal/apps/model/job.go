package model

import "time"

type JobAction string

const (
	JobActionInstall    JobAction = "install"
	JobActionUpgrade    JobAction = "upgrade"
	JobActionRemove     JobAction = "remove"
	JobActionSelfUpdate JobAction = "self-update"
)

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
)

type Job struct {
	ID         string     `json:"id"`
	AppID      string     `json:"app_id"`
	Action     JobAction  `json:"action"`
	Status     JobStatus  `json:"status"`
	Error      string     `json:"error,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type InstallRequest struct {
	ExtraVars map[string]string `json:"extra_vars,omitempty"`
}

type InstallResponse struct {
	JobID string `json:"job_id"`
}

type JobLogsResponse struct {
	Lines []string `json:"lines"`
}
