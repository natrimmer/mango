package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/filesystem"
	"github.com/natrimmer/claude_commit/internal/ui"
)

func TestService_ShowModels(t *testing.T) {
	tests := []struct {
		name         string
		currentModel string
		expectErr    bool
	}{
		{
			name:         "default model selected",
			currentModel: config.DefaultModel,
			expectErr:    false,
		},
		{
			name:         "non-default model selected",
			currentModel: "claude-opus-4-0",
			expectErr:    false,
		},
		{
			name:         "haiku model selected",
			currentModel: "claude-3-5-haiku-latest",
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem()
			mockPrinter := ui.NewMockPrinter()

			// Setup config
			mockFS.SetHomeDir("/tmp")
			cfg := config.Config{ApiKey: "test-key", Model: tt.currentModel}
			configJSON, _ := json.Marshal(cfg)
			mockFS.SetReadData(configJSON)

			configService := config.NewService(mockFS, mockPrinter)
			modelService := NewService(configService, mockPrinter)

			err := modelService.ShowModels()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

				// Check that the correct messages are printed
				if !mockPrinter.ContainsMessage("Available Models:") {
					t.Error("Expected 'Available Models:' message")
				}

				if !mockPrinter.ContainsMessage(tt.currentModel + " [CURRENT]") {
					t.Errorf("Expected current model %q to be marked as [CURRENT]", tt.currentModel)
				}

				// If current model is not default, default should be shown
				if tt.currentModel != config.DefaultModel {
					if !mockPrinter.ContainsMessage(config.DefaultModel + " [DEFAULT]") {
						t.Errorf("Expected default model %q to be marked as [DEFAULT]", config.DefaultModel)
					}
				}

				// Check that all models are listed
				for _, model := range AvailableModels {
					found := false
					for _, msg := range mockPrinter.GetMessages() {
						if strings.Contains(msg, model) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected model %q to be listed", model)
					}
				}
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Test that default model is in available models
	found := false
	for _, model := range AvailableModels {
		if model == config.DefaultModel {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DefaultModel %q should be in AvailableModels", config.DefaultModel)
	}

	// Test that available models list has expected models
	expectedModels := []string{
		"claude-opus-4-0",
		"claude-sonnet-4-0",
		"claude-3-7-sonnet-latest",
		"claude-3-5-sonnet-latest",
		"claude-3-5-haiku-latest",
		"claude-3-opus-latest",
	}

	if len(AvailableModels) != len(expectedModels) {
		t.Errorf("Expected %d available models, got %d", len(expectedModels), len(AvailableModels))
	}

	for _, expected := range expectedModels {
		found := false
		for _, available := range AvailableModels {
			if available == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model %q not found in AvailableModels", expected)
		}
	}
}
