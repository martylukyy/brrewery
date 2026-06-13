package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/autobrr/brrewery/internal/apps/model"
)

type Store struct {
	mu      sync.RWMutex
	jobs    map[string]*jobRecord
	dir     string
	persist bool
}

type jobRecord struct {
	job  model.Job
	logs []string
}

type persistedJob struct {
	Job  model.Job `json:"job"`
	Logs []string  `json:"logs"`
}

// NewStore returns an in-memory job store. Use NewStoreAt with a non-empty dir to persist jobs.
func NewStore() *Store {
	return NewStoreAt("")
}

// NewStoreAt creates a job store. When dir is non-empty, jobs survive process restarts.
func NewStoreAt(dir string) *Store {
	store := &Store{
		jobs: make(map[string]*jobRecord),
		dir:  dir,
	}
	if dir != "" {
		store.persist = true
		store.loadFromDisk()
	}
	return store
}

func (s *Store) Create(appID string, action model.JobAction) model.Job {
	job := model.Job{
		ID:        newJobID(),
		AppID:     appID,
		Action:    action,
		Status:    model.JobStatusQueued,
		StartedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	s.jobs[job.ID] = &jobRecord{job: job}
	s.persistLocked()
	s.mu.Unlock()

	return job
}

func (s *Store) Get(id string) (model.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.jobs[id]
	if !ok {
		return model.Job{}, false
	}
	return record.job, true
}

func (s *Store) Logs(id string) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	out := make([]string, len(record.logs))
	copy(out, record.logs)
	return out, true
}

func (s *Store) SetStatus(id string, status model.JobStatus, errMsg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.jobs[id]
	if !ok {
		return false
	}

	record.job.Status = status
	record.job.Error = errMsg
	if status == model.JobStatusSucceeded || status == model.JobStatusFailed {
		now := time.Now().UTC()
		record.job.FinishedAt = &now
	}
	s.persistLocked()
	return true
}

func (s *Store) AppendLog(id, line string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.jobs[id]
	if !ok {
		return false
	}
	record.logs = append(record.logs, line)
	s.persistLocked()
	return true
}

func (s *Store) MarkRunning(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.jobs[id]
	if !ok {
		return false
	}
	record.job.Status = model.JobStatusRunning
	s.persistLocked()
	return true
}

func (s *Store) persistLocked() {
	if !s.persist {
		return
	}
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return
	}

	for id, record := range s.jobs {
		payload, err := json.Marshal(persistedJob{
			Job:  record.job,
			Logs: record.logs,
		})
		if err != nil {
			continue
		}
		path := filepath.Join(s.dir, id+".json")
		_ = os.WriteFile(path, payload, 0o600)
	}
}

func (s *Store) loadFromDisk() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var persisted persistedJob
		if err := json.Unmarshal(data, &persisted); err != nil {
			continue
		}
		if persisted.Job.ID == "" {
			continue
		}
		s.jobs[persisted.Job.ID] = &jobRecord{
			job:  persisted.Job,
			logs: append([]string(nil), persisted.Logs...),
		}
	}
}

func newJobID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}

func OpenStore(dir string) (*Store, error) {
	if dir == "" {
		return NewStore(), nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create jobs dir: %w", err)
	}
	return NewStoreAt(dir), nil
}
