package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/alexedwards/scs/v2"
)

type Service struct {
	store   *FileStore
	session *scs.SessionManager
}

func NewService(store *FileStore, session *scs.SessionManager) *Service {
	return &Service{store: store, session: session}
}

func (s *Service) HasUsers() (bool, error) {
	return s.store.HasUsers()
}

func (s *Service) CreateAdmin(username, password string) (*User, error) {
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	id, err := newUserID()
	if err != nil {
		return nil, err
	}

	user := User{
		ID:           id,
		Username:     username,
		PasswordHash: hash,
		TenantID:     "",
	}

	if err := s.store.CreateAdmin(user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*User, error) {
	user, err := s.store.GetByUsername(username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidPassword
		}
		return nil, err
	}

	if !CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidPassword
	}

	if err := s.session.RenewToken(ctx); err != nil {
		return nil, fmt.Errorf("renew session: %w", err)
	}

	s.session.Put(ctx, SessionKey(), true)
	s.session.Put(ctx, "user_id", user.ID)
	s.session.Put(ctx, "username", user.Username)

	return user, nil
}

func (s *Service) VerifyPassword(username, password string) error {
	user, err := s.store.GetByUsername(username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrInvalidPassword
		}
		return err
	}
	if !CheckPassword(user.PasswordHash, password) {
		return ErrInvalidPassword
	}
	return nil
}

func (s *Service) Logout(ctx context.Context) error {
	return s.session.Destroy(ctx)
}

func (s *Service) IsAuthenticated(ctx context.Context) bool {
	return s.session.GetBool(ctx, SessionKey())
}

func (s *Service) Username(ctx context.Context) (string, bool) {
	username := s.session.GetString(ctx, "username")
	return username, username != ""
}

func newUserID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
