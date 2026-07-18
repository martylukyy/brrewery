package auth

import (
	"testing"
	"time"

	"github.com/autobrr/brrewery/internal/buildinfo"
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

func TestNewSessionManager_secureCookieByDefaultInReleaseBuilds(t *testing.T) {
	// Not parallel: mutates the package-level buildinfo.Version.
	orig := buildinfo.Version
	buildinfo.Version = "1.2.3"
	t.Cleanup(func() { buildinfo.Version = orig })

	manager := NewSessionManager(nil)
	if !manager.Cookie.Secure {
		t.Fatal("Cookie.Secure = false for a release build, want true")
	}
}

func TestNewSessionManager_devBuildRelaxesSecureCookie(t *testing.T) {
	orig := buildinfo.Version
	buildinfo.Version = "0.0.0-dev"
	t.Cleanup(func() { buildinfo.Version = orig })

	manager := NewSessionManager(nil)
	if manager.Cookie.Secure {
		t.Fatal("Cookie.Secure = true for a dev build, want false")
	}
}

func TestSecureCookies_insecureEnvOverride(t *testing.T) {
	orig := buildinfo.Version
	buildinfo.Version = "1.2.3"
	t.Cleanup(func() { buildinfo.Version = orig })
	t.Setenv("BRREWERY_INSECURE_COOKIES", "1")

	if secureCookies() {
		t.Fatal("secureCookies() = true with BRREWERY_INSECURE_COOKIES=1, want false")
	}
}
