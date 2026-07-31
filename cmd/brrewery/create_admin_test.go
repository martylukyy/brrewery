package main

import (
	"os"
	"strings"
	"testing"
)

// withStdin points the prompt helpers at scripted input. Prompts go to
// os.Stdout, which is muted so failures stay readable.
func withStdin(t *testing.T, input string) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	go func() {
		defer writer.Close()
		_, _ = writer.WriteString(input)
	}()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}

	origStdin, origStdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = reader, devNull
	t.Cleanup(func() {
		os.Stdin, os.Stdout = origStdin, origStdout
		reader.Close()
		devNull.Close()
	})
}

func TestValidateNewUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{name: "simple", username: "brewer"},
		{name: "digits and dash", username: "brewer-01"},
		{name: "leading underscore", username: "_svc"},
		{name: "max length", username: "a" + strings.Repeat("b", 30)},
		{name: "empty", username: "", wantErr: true},
		{name: "leading digit", username: "1brewer", wantErr: true},
		{name: "uppercase", username: "Brewer", wantErr: true},
		{name: "embedded space", username: "brew er", wantErr: true},
		{name: "dot breaks sudoers drop-in naming", username: "brew.er", wantErr: true},
		{name: "root is rejected like any taken name only by lookup", username: "root"},
		{name: "too long", username: "a" + strings.Repeat("b", 31), wantErr: true},
		{name: "shell metacharacter", username: "brewer;id", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateNewUsername(tt.username)
			if tt.wantErr && err == nil {
				t.Fatalf("validateNewUsername(%q) = nil, want error", tt.username)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateNewUsername(%q) = %v, want nil", tt.username, err)
			}
		})
	}
}

func TestPromptNewUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "accepts first valid name", input: "brewer\n", want: "brewer"},
		{name: "retries past invalid names", input: "1bad\nAlso Bad\nbrewer\n", want: "brewer"},
		{name: "rejects an existing account", input: "root\nbrewer\n", want: "brewer"},
		{name: "gives up after three tries", input: "1bad\n2bad\n3bad\n", wantErr: true},
		{name: "fails on closed input", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdin(t, tt.input)

			got, err := promptNewUsername()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("promptNewUsername() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("promptNewUsername() error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("promptNewUsername() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPromptNewPassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "accepts a matching pair", input: "hunter2hunter2\nhunter2hunter2\n", want: "hunter2hunter2"},
		{name: "retries after a mismatch", input: "hunter2hunter2\ntypo\ngoodpassword\ngoodpassword\n", want: "goodpassword"},
		{name: "rejects short passwords", input: "short\nshort\nlongenough\nlongenough\n", want: "longenough"},
		{name: "gives up after three tries", input: "a\na\nb\nb\nc\nc\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdin(t, tt.input)

			got, err := promptNewPassword("brewer")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("promptNewPassword() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("promptNewPassword() error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("promptNewPassword() = %q, want %q", got, tt.want)
			}
		})
	}
}
