package auth

import (
	"testing"
	"time"
)

func TestNewSessionManager_lifetime(t *testing.T) {
	t.Parallel()

	manager := NewSessionManager(nil)
	if manager.Lifetime != 365*24*time.Hour {
		t.Fatalf("Lifetime = %v, want 8760h", manager.Lifetime)
	}
}

func TestNewSessionManager_cookieName(t *testing.T) {
	t.Parallel()

	manager := NewSessionManager(nil)
	if manager.Cookie.Name != "brrewery_session" {
		t.Fatalf("Cookie.Name = %q, want %q", manager.Cookie.Name, "brrewery_session")
	}
}
