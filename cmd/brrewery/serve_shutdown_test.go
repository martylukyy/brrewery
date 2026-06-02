package main

import (
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestShutdownHTTPServer_graceful(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		_ = srv.Serve(ln) //nolint:errcheck // tested via shutdown
	}()

	sigCh := make(chan os.Signal, 2)
	logger := zerolog.New(io.Discard)

	if err := shutdownHTTPServer(srv, sigCh, &logger); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 200*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		t.Fatal("listener still accepting connections after shutdown")
	}
}

func TestShutdownHTTPServer_forceOnSecondSignal(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	block := make(chan struct{})
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/block" {
				<-block
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		_ = srv.Serve(ln)
	}()

	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		_, _ = client.Get("http://" + ln.Addr().String() + "/block")
	}()

	time.Sleep(50 * time.Millisecond)

	sigCh := make(chan os.Signal, 2)
	sigCh <- syscall.SIGINT

	logger := zerolog.New(io.Discard)

	done := make(chan error, 1)
	go func() {
		done <- shutdownHTTPServer(srv, sigCh, &logger)
	}()

	time.Sleep(50 * time.Millisecond)
	sigCh <- syscall.SIGINT

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		close(block)
		t.Fatal("shutdown did not complete after forced signal")
	}
}
