package auth

import (
	"testing"
	"time"
)

func TestNewSessionManager_lifetime(t *testing.T) {
	t.Parallel()

	manager := NewSessionManager(nil)
	if manager.Lifetime != 24*time.Hour {
		t.Fatalf("Lifetime = %v, want 24h", manager.Lifetime)
	}
}
