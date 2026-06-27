package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	p := buildPrompt("a.go\n", "diff", "fix", "issue #1", 3)
	for _, want := range []string{
		"The commit type MUST be 'fix'",
		"Additional context: issue #1",
		"Generate 3 different commit message options",
		"Commit messages (one per line):",
		"a.go",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q", want)
		}
	}

	// Single-message prompt omits the multi-option instruction.
	if strings.Contains(buildPrompt("a.go", "diff", "", "", 1), "different commit message options") {
		t.Error("single prompt should not mention options")
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr string
	}{
		{"ok", 200, `{"content":[{"text":"feat: add thing"}]}`, "feat: add thing", ""},
		{"api error", 401, `{"error":"unauthorized"}`, "", "API error"},
		{"empty content", 200, `{"content":[]}`, "", "empty response"},
		{"bad json", 200, "not json", "", "error parsing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			old := apiURL
			apiURL = srv.URL
			defer func() { apiURL = old }()

			got, err := generateCommitMessage(Config{ApiKey: "k", Model: "claude-sonnet-4-6"}, "prompt")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Guard the request shape the API depends on.
func TestGenerateCommitMessage_RequestBody(t *testing.T) {
	var seen map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&seen)
		if r.Header.Get("x-api-key") != "secret" {
			t.Errorf("missing api key header")
		}
		_, _ = w.Write([]byte(`{"content":[{"text":"ok"}]}`))
	}))
	defer srv.Close()

	old := apiURL
	apiURL = srv.URL
	defer func() { apiURL = old }()

	_, _ = generateCommitMessage(Config{ApiKey: "secret", Model: "claude-opus-4-8"}, "hello")
	if seen["model"] != "claude-opus-4-8" {
		t.Errorf("model not sent: %v", seen["model"])
	}
}
