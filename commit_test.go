package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestParseSelection(t *testing.T) {
	cases := []struct {
		in        string
		max, want int
	}{
		{"1", 3, 1}, {"3", 3, 3}, {" 2 ", 3, 2},
		{"0", 3, -1}, {"4", 3, -1}, {"", 3, -1}, {"x", 3, -1},
	}
	for _, c := range cases {
		if got := parseSelection(c.in, c.max); got != c.want {
			t.Errorf("parseSelection(%q, %d) = %d, want %d", c.in, c.max, got, c.want)
		}
	}
}

func TestYes(t *testing.T) {
	for _, s := range []string{"", "y", "Y", "yes", " Yes "} {
		if !yes(s) {
			t.Errorf("yes(%q) should be true", s)
		}
	}
	for _, s := range []string{"n", "no", "x"} {
		if yes(s) {
			t.Errorf("yes(%q) should be false", s)
		}
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
		{
			"skips thinking block", 200,
			`{"content":[{"type":"thinking","thinking":"","text":""},{"type":"text","text":"fix: handle nil"}]}`,
			"fix: handle nil", "",
		},
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

// stubClaude installs a shell script standing in for the claude CLI, and points
// claudeBin at it for the duration of the test.
func stubClaude(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script stub needs a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}
	old := claudeBin
	claudeBin = path
	t.Cleanup(func() { claudeBin = old })
}

func TestGenerateViaClaudeCode(t *testing.T) {
	// Echo the flags and the stdin prompt back so both can be asserted on.
	stubClaude(t, `printf '%s\n' "$*"; cat`)

	got, err := generateCommitMessage(Config{Provider: providerClaudeCode, Model: "claude-opus-4-8"}, "the prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"-p", "--tools", "--model claude-opus-4-8", "--output-format text", "the prompt"} {
		if !strings.Contains(got, want) {
			t.Errorf("invocation missing %q, got %q", want, got)
		}
	}
}

func TestGenerateViaClaudeCode_Errors(t *testing.T) {
	t.Run("non-zero exit", func(t *testing.T) {
		stubClaude(t, `echo "credit balance too low" >&2; exit 1`)
		_, err := generateCommitMessage(Config{Provider: providerClaudeCode, Model: defaultModel}, "p")
		if err == nil || !strings.Contains(err.Error(), "credit balance too low") {
			t.Fatalf("want stderr in error, got %v", err)
		}
	})

	t.Run("empty output", func(t *testing.T) {
		stubClaude(t, `exit 0`)
		_, err := generateCommitMessage(Config{Provider: providerClaudeCode, Model: defaultModel}, "p")
		if err == nil || !strings.Contains(err.Error(), "empty response") {
			t.Fatalf("want empty response error, got %v", err)
		}
	})

	t.Run("not installed", func(t *testing.T) {
		old := claudeBin
		claudeBin = filepath.Join(t.TempDir(), "no-such-claude")
		defer func() { claudeBin = old }()

		_, err := generateCommitMessage(Config{Provider: providerClaudeCode, Model: defaultModel}, "p")
		if err == nil || !strings.Contains(err.Error(), "mango config set --provider api") {
			t.Fatalf("want actionable not-found error, got %v", err)
		}
	})
}

// Guard the request shape the API depends on.
func TestGenerateCommitMessage_RequestBody(t *testing.T) {
	var seen map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decoding into a non-nil map merges keys, so start fresh each request.
		seen = nil
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
	if _, ok := seen["thinking"]; ok {
		t.Errorf("thinking sent for a model that doesn't think by default")
	}

	// Models that think by default must be told not to, or the reasoning eats
	// the max_tokens budget the commit message is supposed to fit in.
	// Each model accepts a different off switch; the wrong one is a 400.
	for model, want := range map[string]string{
		"claude-opus-5":     "disabled",
		"claude-sonnet-5":   "disabled",
		"claude-haiku-5-5":  "disabled",
		"claude-sonnet-5-5": "between_tools",
	} {
		_, _ = generateCommitMessage(Config{ApiKey: "secret", Model: model}, "hello")
		thinking, ok := seen["thinking"].(map[string]any)
		if !ok || thinking["type"] != want {
			t.Errorf("%s: thinking = %v, want type %q", model, seen["thinking"], want)
		}
	}

	// Models that always think reject any thinking setting but "adaptive", so
	// they get low effort and more room instead.
	for _, model := range []string{"claude-opus-5-5", "claude-fable-5-1"} {
		_, _ = generateCommitMessage(Config{ApiKey: "secret", Model: model}, "hello")
		if _, ok := seen["thinking"]; ok {
			t.Errorf("%s: thinking sent to a model that rejects it: %v", model, seen["thinking"])
		}
		oc, _ := seen["output_config"].(map[string]any)
		if oc["effort"] != "low" {
			t.Errorf("%s: output_config = %v, want effort low", model, seen["output_config"])
		}
		if seen["max_tokens"] != float64(4096) {
			t.Errorf("%s: max_tokens = %v, want 4096", model, seen["max_tokens"])
		}
	}
}
