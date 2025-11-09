package config

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natrimmer/claude_commit/internal/filesystem"
	"github.com/natrimmer/claude_commit/internal/ui"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal API key",
			input:    "sk-ant-api03-1234567890abcdef",
			expected: "sk-a****cdef",
		},
		{
			name:     "short API key",
			input:    "short",
			expected: "********",
		},
		{
			name:     "exactly 8 chars",
			input:    "12345678",
			expected: "********",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "********",
		},
		{
			name:     "very long API key",
			input:    "sk-ant-api03-very-long-api-key-with-many-characters",
			expected: "sk-a****ters",
		},
		{
			name:     "minimum length plus one",
			input:    "123456789",
			expected: "1234****6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskAPIKey_Properties(t *testing.T) {
	tests := []string{
		"a", "ab", "abcd", "abcdefgh", "abcdefghi",
		"sk-ant-api03-short", "sk-ant-api03-very-long-key-with-many-characters",
		strings.Repeat("x", 100),
	}

	for _, input := range tests {
		t.Run("len_"+string(rune(len(input))), func(t *testing.T) {
			result := MaskAPIKey(input)

			// Properties that should always hold
			if result == "" {
				t.Error("Result should never be empty")
			}

			if len(input) <= 8 {
				if result != "********" {
					t.Error("Short inputs should be fully masked")
				}
			} else {
				// Should contain original prefix and suffix
				if !strings.HasPrefix(result, input[:4]) {
					t.Error("Should preserve first 4 chars")
				}
				if !strings.HasSuffix(result, input[len(input)-4:]) {
					t.Error("Should preserve last 4 chars")
				}
				if !strings.Contains(result, "****") {
					t.Error("Should contain mask characters")
				}
			}
		})
	}
}

func TestService_SaveConfig(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		model          string
		existingConfig *Config
		setupMock      func(*filesystem.MockFileSystem)
		expectError    bool
		errorMsg       string
		expectedConfig *Config
	}{
		{
			name:   "successful save with both parameters",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
			},
			expectError: false,
			expectedConfig: &Config{
				ApiKey: "test-api-key",
				Model:  "test-model",
			},
		},
		{
			name:   "update only API key",
			apiKey: "new-api-key",
			model:  "",
			existingConfig: &Config{
				ApiKey: "old-api-key",
				Model:  "existing-model",
			},
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				config := Config{ApiKey: "old-api-key", Model: "existing-model"}
				configJSON, _ := json.Marshal(config)
				fs.SetReadData(configJSON)
			},
			expectError: false,
			expectedConfig: &Config{
				ApiKey: "new-api-key",
				Model:  "existing-model",
			},
		},
		{
			name:   "update only model",
			apiKey: "",
			model:  "new-model",
			existingConfig: &Config{
				ApiKey: "existing-api-key",
				Model:  "old-model",
			},
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				config := Config{ApiKey: "existing-api-key", Model: "old-model"}
				configJSON, _ := json.Marshal(config)
				fs.SetReadData(configJSON)
			},
			expectError: false,
			expectedConfig: &Config{
				ApiKey: "existing-api-key",
				Model:  "new-model",
			},
		},
		{
			name:   "empty API key with no existing config",
			apiKey: "",
			model:  "test-model",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				fs.SetReadError(errors.New("file not found"))
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
		{
			name:   "home directory error",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeError(errors.New("home dir error"))
			},
			expectError: true,
			errorMsg:    "error getting home directory",
		},
		{
			name:   "mkdir error",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				fs.SetMkdirError(errors.New("mkdir error"))
			},
			expectError: true,
			errorMsg:    "error creating config directory",
		},
		{
			name:   "write file error",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				fs.SetWriteError(errors.New("write error"))
			},
			expectError: true,
			errorMsg:    "error writing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem()
			mockPrinter := ui.NewMockPrinter()
			tt.setupMock(mockFS)

			service := NewService(mockFS, mockPrinter)
			err := service.SaveConfig(tt.apiKey, tt.model)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

				// Check that config was written correctly
				expectedPath := filepath.Join("/tmp", ".claude-commit", "config.json")
				if data := mockFS.GetWrittenFile(expectedPath); data != nil {
					var config Config
					if err := json.Unmarshal(data, &config); err != nil {
						t.Errorf("Failed to unmarshal written config: %v", err)
					} else {
						if tt.expectedConfig != nil {
							if config.ApiKey != tt.expectedConfig.ApiKey {
								t.Errorf("Expected API key %q, got %q", tt.expectedConfig.ApiKey, config.ApiKey)
							}
							if config.Model != tt.expectedConfig.Model {
								t.Errorf("Expected model %q, got %q", tt.expectedConfig.Model, config.Model)
							}
						}
					}
				} else {
					t.Error("Config file was not written")
				}

				// Check that success message was printed
				if !mockPrinter.ContainsMessage("Configuration saved successfully") {
					t.Error("Expected success message to be printed")
				}
			}
		})
	}
}

func TestService_LoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*filesystem.MockFileSystem)
		expectErr bool
		expected  *Config
		errorMsg  string
	}{
		{
			name: "successful load",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				configJSON := `{"api_key":"test-key","model":"test-model"}`
				fs.SetReadData([]byte(configJSON))
			},
			expectErr: false,
			expected: &Config{
				ApiKey: "test-key",
				Model:  "test-model",
			},
		},
		{
			name: "home directory error",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeError(errors.New("home dir error"))
			},
			expectErr: true,
			errorMsg:  "error getting home directory",
		},
		{
			name: "file read error",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				fs.SetReadError(errors.New("file not found"))
			},
			expectErr: true,
			errorMsg:  "error reading config file",
		},
		{
			name: "invalid JSON",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				fs.SetReadData([]byte("invalid json"))
			},
			expectErr: true,
			errorMsg:  "error parsing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem()
			mockPrinter := ui.NewMockPrinter()
			tt.setupMock(mockFS)

			service := NewService(mockFS, mockPrinter)
			config, err := service.LoadConfig()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if config.ApiKey != tt.expected.ApiKey {
					t.Errorf("Expected API key %q, got %q", tt.expected.ApiKey, config.ApiKey)
				}
				if config.Model != tt.expected.Model {
					t.Errorf("Expected model %q, got %q", tt.expected.Model, config.Model)
				}
			}
		})
	}
}

func TestService_ViewConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*filesystem.MockFileSystem)
		expectErr bool
		checkMsg  string
	}{
		{
			name: "successful view",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				configJSON := `{"api_key":"test-api-key","model":"test-model"}`
				fs.SetReadData([]byte(configJSON))
			},
			expectErr: false,
			checkMsg:  "Current Configuration:",
		},
		{
			name: "config load error",
			setupMock: func(fs *filesystem.MockFileSystem) {
				fs.SetHomeDir("/tmp")
				fs.SetReadError(errors.New("config not found"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem()
			mockPrinter := ui.NewMockPrinter()
			tt.setupMock(mockFS)

			service := NewService(mockFS, mockPrinter)
			err := service.ViewConfig()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if !mockPrinter.ContainsMessage(tt.checkMsg) {
					t.Errorf("Expected message %q to be printed", tt.checkMsg)
				}
			}
		})
	}
}

func BenchmarkMaskAPIKey(b *testing.B) {
	apiKey := "sk-ant-api03-1234567890abcdef1234567890abcdef"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		MaskAPIKey(apiKey)
	}
}

func BenchmarkService_LoadConfig(b *testing.B) {
	mockFS := filesystem.NewMockFileSystem()
	mockFS.SetHomeDir("/tmp")
	config := Config{ApiKey: "test-key", Model: "test-model"}
	configJSON, _ := json.Marshal(config)
	mockFS.SetReadData(configJSON)

	service := NewService(mockFS, ui.NewMockPrinter())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.LoadConfig()
		if err != nil {
			b.Fatal(err)
		}
	}
}
