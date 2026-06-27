package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/natrimmer/claude_commit/internal/anthropic"
	"github.com/natrimmer/claude_commit/internal/commit"
	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/filesystem"
	"github.com/natrimmer/claude_commit/internal/git"
	"github.com/natrimmer/claude_commit/internal/model"
	"github.com/natrimmer/claude_commit/internal/ui"
)

func TestApp_HandleConfig(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		model          string
		existingConfig bool
		expectErr      bool
	}{
		{
			name:      "successful config with both parameters",
			apiKey:    "test-api-key",
			model:     "claude-sonnet-4-6",
			expectErr: false,
		},
		{
			name:           "update only model with existing config",
			apiKey:         "",
			model:          "claude-haiku-4-5",
			existingConfig: true,
			expectErr:      false,
		},
		{
			name:      "empty api key without existing config",
			apiKey:    "",
			model:     "test-model",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem()
			mockFS.SetHomeDir("/tmp")
			mockPrinter := ui.NewMockPrinter()

			if tt.existingConfig {
				cfg := config.Config{ApiKey: "existing-api-key", Model: "existing-model"}
				configJSON, _ := json.Marshal(cfg)
				mockFS.SetReadData(configJSON)
			}

			configService := config.NewService(mockFS, mockPrinter)
			modelService := model.NewService(configService, mockPrinter)

			app := New(configService, modelService, nil, mockPrinter)

			err := app.HandleConfig(tt.apiKey, tt.model)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestApp_ShowVersion(t *testing.T) {
	// Test with default "v0.0.0-dev" version
	mockPrinter := ui.NewMockPrinter()
	app := New(nil, nil, nil, mockPrinter)

	app.ShowVersion()

	messages := mockPrinter.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected version messages, got none")
	}

	// Should contain "Claude Commit" and version
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "Claude Commit") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'Claude Commit' in version output")
	}

	// Should contain the SemVer version
	foundVersion := false
	for _, msg := range messages {
		if strings.Contains(msg, "v0.0.0-dev") {
			foundVersion = true
			break
		}
	}
	if !foundVersion {
		t.Error("Expected SemVer version 'v0.0.0-dev' in version output")
	}
}

func TestApp_ShowConfigHelp(t *testing.T) {
	mockPrinter := ui.NewMockPrinter()
	app := New(nil, nil, nil, mockPrinter)

	app.ShowConfigHelp()

	messages := mockPrinter.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected config help messages, got none")
	}

	// Check for expected content
	expectedContent := []string{
		"Claude Commit Config",
		"Configure API key and model settings",
		"Usage:",
		"claude_commit config [flags]",
		"Flags:",
		"-api-key string",
		"-model string",
		"Examples:",
		"Initial setup",
		"Update only API key",
		"Update only model",
		"claude_commit view",
		"claude_commit models",
	}

	for _, expected := range expectedContent {
		if !mockPrinter.ContainsMessage(expected) {
			t.Errorf("Expected config help to contain: %q", expected)
		}
	}
}

func TestApp_ShowHelp(t *testing.T) {
	mockPrinter := ui.NewMockPrinter()
	app := New(nil, nil, nil, mockPrinter)

	app.ShowHelp()

	messages := mockPrinter.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected help messages, got none")
	}

	// Check for expected sections
	expectedSections := []string{
		"Claude Commit",
		"Commands:",
		"config",
		"view",
		"models",
		"commit",
		"help",
		"Flags:",
		"--version",
		"--help",
		"Examples:",
		"Commit Types:",
	}

	for _, expected := range expectedSections {
		if !mockPrinter.ContainsMessage(expected) {
			t.Errorf("Expected help to contain section: %q", expected)
		}
	}
}

func TestApp_HandleHelp(t *testing.T) {
	mockPrinter := ui.NewMockPrinter()
	app := New(nil, nil, nil, mockPrinter)

	app.HandleHelp()

	// Should call ShowHelp which prints help messages
	if len(mockPrinter.GetMessages()) == 0 {
		t.Error("Expected help messages from HandleHelp")
	}
}

func TestApp_Integration(t *testing.T) {
	// Test that all services work together
	mockFS := filesystem.NewMockFileSystem()
	mockFS.SetHomeDir("/tmp")
	mockPrinter := ui.NewMockPrinter()
	mockGit := git.NewMockClient()
	mockHTTP := anthropic.NewMockHTTPClient()

	// Setup config
	cfg := config.Config{ApiKey: "test-key", Model: "test-model"}
	configJSON, _ := json.Marshal(cfg)
	mockFS.SetReadData(configJSON)

	// Create services
	configService := config.NewService(mockFS, mockPrinter)
	modelService := model.NewService(configService, mockPrinter)
	anthropicClient := anthropic.NewClient(mockHTTP, mockPrinter)
	commitService := commit.NewService(configService, anthropicClient, mockGit, mockPrinter)

	app := New(configService, modelService, commitService, mockPrinter)

	// Test view command
	err := app.HandleView()
	if err != nil {
		t.Errorf("HandleView failed: %v", err)
	}

	// Test models command
	err = app.HandleModels()
	if err != nil {
		t.Errorf("HandleModels failed: %v", err)
	}

	// Test that app is properly initialized
	if app.configService == nil {
		t.Error("App configService is nil")
	}
	if app.modelService == nil {
		t.Error("App modelService is nil")
	}
	if app.commitService == nil {
		t.Error("App commitService is nil")
	}
	if app.printer == nil {
		t.Error("App printer is nil")
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables are properly defined
	if Version == "" {
		t.Error("Version variable should not be empty")
	}

	if BuildDate == "" {
		t.Error("BuildDate variable should not be empty")
	}

	if CommitSHA == "" {
		t.Error("CommitSHA variable should not be empty")
	}

	// Test SemVer default values
	if Version != "v0.0.0-dev" {
		t.Logf("Note: Version is set to %q (not default 'v0.0.0-dev')", Version)
	}

	if BuildDate != "unknown" {
		t.Logf("Note: BuildDate is set to %q (not default 'unknown')", BuildDate)
	}

	if CommitSHA != "unknown" {
		t.Logf("Note: CommitSHA is set to %q (not default 'unknown')", CommitSHA)
	}

	// Test SemVer format validation
	if !strings.HasPrefix(Version, "v") {
		t.Errorf("Version should follow SemVer format and start with 'v', got: %q", Version)
	}
}
