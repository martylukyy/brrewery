package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autobrr/brrewery/internal/paths"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrNoUsers         = errors.New("no users configured")
	ErrInvalidPassword = errors.New("invalid password")
)

type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	if path == "" {
		path = paths.UserStorePath
	}
	return &FileStore{path: path}
}

func (s *FileStore) load() (*UserStore, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserStore{Users: []User{}}, nil
		}
		return nil, fmt.Errorf("read user store: %w", err)
	}

	var store UserStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parse user store: %w", err)
	}
	return &store, nil
}

func (s *FileStore) save(store *UserStore) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return fmt.Errorf("create user store dir: %w", err)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal user store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write user store: %w", err)
	}
	return nil
}

func (s *FileStore) HasUsers() (bool, error) {
	store, err := s.load()
	if err != nil {
		return false, err
	}
	return len(store.Users) > 0, nil
}

func (s *FileStore) GetByUsername(username string) (*User, error) {
	store, err := s.load()
	if err != nil {
		return nil, err
	}

	for i := range store.Users {
		if store.Users[i].Username == username {
			return &store.Users[i], nil
		}
	}
	return nil, ErrUserNotFound
}

func (s *FileStore) CreateAdmin(user User) error {
	store, err := s.load()
	if err != nil {
		return err
	}

	if len(store.Users) > 0 {
		return ErrUserExists
	}

	store.Users = append(store.Users, user)
	return s.save(store)
}
