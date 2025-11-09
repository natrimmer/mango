package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mock implementations for testing

// MockFileSystem implements FileSystem interface for testing
type MockFileSystem struct {
	homeDir    string
	homeErr    error
	mkdirErr   error
	writeErr   error
	readData   []byte
	readErr    error
	writeFiles map[string][]byte // Track what was written
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		writeFiles: make(map[string][]byte),
	}
}

func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.homeDir, m.homeErr
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return m.mkdirErr
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writeFiles[filename] = data
	return nil
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	return m.readData, m.readErr
}

// MockHTTPClient implements HTTPClient interface for testing
type MockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// MockGitClient implements GitClient interface for testing
type MockGitClient struct {
	stagedDiff  string
	stagedFiles string
	diffErr     error
	filesErr    error
}

func (m *MockGitClient) GetStagedDiff() (string, error) {
	return m.stagedDiff, m.diffErr
}

func (m *MockGitClient) GetStagedFiles() (string, error) {
	return m.stagedFiles, m.filesErr
}

// MockPrinter implements Printer interface for testing
type MockPrinter struct {
	messages []string
}

func (m *MockPrinter) Print(msg string) {
	m.messages = append(m.messages, msg)
}

func (m *MockPrinter) PrintSuccess(msg string) {
	m.messages = append(m.messages, "[SUCCESS] "+msg)
}

func (m *MockPrinter) PrintError(msg string) {
	m.messages = append(m.messages, "[ERROR] "+msg)
}

func (m *MockPrinter) PrintWarning(msg string) {
	m.messages = append(m.messages, "[WARNING] "+msg)
}

func (m *MockPrinter) GetMessages() []string {
	return m.messages
}

func (m *MockPrinter) Reset() {
	m.messages = nil
}

func (m *MockPrinter) ContainsMessage(msg string) bool {
	for _, message := range m.messages {
		if strings.Contains(message, msg) {
			return true
		}
	}
	return false
}

// Helper function to create HTTP response
func createHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// Test MaskAPIKey function
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

// Test ConfigService
func TestConfigService_SaveConfig(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		model          string
		existingConfig *Config
		setupMock      func(*MockFileSystem)
		expectError    bool
		errorMsg       string
		expectedConfig *Config
	}{
		{
			name:   "successful save with both parameters",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
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
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{ApiKey: "old-api-key", Model: "existing-model"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
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
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{ApiKey: "existing-api-key", Model: "old-model"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
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
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("file not found")
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
		{
			name:   "home directory error",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *MockFileSystem) {
				fs.homeErr = errors.New("home dir error")
			},
			expectError: true,
			errorMsg:    "error getting home directory",
		},
		{
			name:   "mkdir error",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.mkdirErr = errors.New("mkdir error")
			},
			expectError: true,
			errorMsg:    "error creating config directory",
		},
		{
			name:   "write file error",
			apiKey: "test-api-key",
			model:  "test-model",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.writeErr = errors.New("write error")
			},
			expectError: true,
			errorMsg:    "error writing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			err := configService.SaveConfig(tt.apiKey, tt.model)

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
				if data, exists := mockFS.writeFiles[expectedPath]; exists {
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

func TestConfigService_LoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockFileSystem)
		expectErr bool
		expected  *Config
		errorMsg  string
	}{
		{
			name: "successful load",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				configJSON := `{"api_key":"test-key","model":"test-model"}`
				fs.readData = []byte(configJSON)
			},
			expectErr: false,
			expected: &Config{
				ApiKey: "test-key",
				Model:  "test-model",
			},
		},
		{
			name: "home directory error",
			setupMock: func(fs *MockFileSystem) {
				fs.homeErr = errors.New("home dir error")
			},
			expectErr: true,
			errorMsg:  "error getting home directory",
		},
		{
			name: "file read error",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("file not found")
			},
			expectErr: true,
			errorMsg:  "error reading config file",
		},
		{
			name: "invalid JSON",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readData = []byte("invalid json")
			},
			expectErr: true,
			errorMsg:  "error parsing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			config, err := configService.LoadConfig()

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

func TestConfigService_ViewConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockFileSystem)
		expectErr bool
		checkMsg  string
	}{
		{
			name: "successful view",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				configJSON := `{"api_key":"test-api-key","model":"test-model"}`
				fs.readData = []byte(configJSON)
			},
			expectErr: false,
			checkMsg:  "Current Configuration:",
		},
		{
			name: "config load error",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("config not found")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			err := configService.ViewConfig()

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

// Test ModelService
func TestModelService_ShowModels(t *testing.T) {
	tests := []struct {
		name         string
		currentModel string
		expectErr    bool
	}{
		{
			name:         "default model selected",
			currentModel: DefaultModel,
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
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}

			// Setup config
			mockFS.homeDir = "/tmp"
			config := Config{ApiKey: "test-key", Model: tt.currentModel}
			configJSON, _ := json.Marshal(config)
			mockFS.readData = configJSON

			configService := NewConfigService(mockFS, mockPrinter)
			modelService := NewModelService(configService, mockPrinter)

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
				if tt.currentModel != DefaultModel {
					if !mockPrinter.ContainsMessage(DefaultModel + " [DEFAULT]") {
						t.Errorf("Expected default model %q to be marked as [DEFAULT]", DefaultModel)
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

// Test AnthropicService
func TestAnthropicService_GenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		prompt      string
		setupMock   func(*MockHTTPClient)
		expectErr   bool
		expectedMsg string
		errorMsg    string
	}{
		{
			name:   "successful generation",
			config: Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				response := AnthropicResponse{
					Content: []struct {
						Text string `json:"text"`
					}{
						{Text: "feat: add new feature"},
					},
				}
				responseJSON, _ := json.Marshal(response)
				client.response = createHTTPResponse(200, string(responseJSON))
			},
			expectErr:   false,
			expectedMsg: "feat: add new feature",
		},
		{
			name:   "HTTP client error",
			config: Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				client.err = errors.New("network error")
			},
			expectErr: true,
			errorMsg:  "error making API call",
		},
		{
			name:   "API error response",
			config: Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				client.response = createHTTPResponse(401, `{"error": "unauthorized"}`)
			},
			expectErr: true,
			errorMsg:  "API error",
		},
		{
			name:   "empty response content",
			config: Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				response := AnthropicResponse{Content: []struct {
					Text string `json:"text"`
				}{}}
				responseJSON, _ := json.Marshal(response)
				client.response = createHTTPResponse(200, string(responseJSON))
			},
			expectErr: true,
			errorMsg:  "empty response from API",
		},
		{
			name:   "invalid JSON response",
			config: Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				client.response = createHTTPResponse(200, "invalid json")
			},
			expectErr: true,
			errorMsg:  "error parsing API response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{}
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockClient)

			service := NewAnthropicService(mockClient, mockPrinter)
			result, err := service.GenerateCommitMessage(tt.config, tt.prompt)

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
				if result != tt.expectedMsg {
					t.Errorf("Expected result %q, got %q", tt.expectedMsg, result)
				}
			}
		})
	}
}

// Test CommitService
func TestCommitService_GenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockFileSystem, *MockGitClient, *MockHTTPClient)
		expectErr      bool
		errorMsg       string
		expectedOutput string
	}{
		{
			name: "successful generation",
			setupMocks: func(fs *MockFileSystem, git *MockGitClient, http *MockHTTPClient) {
				// Config
				fs.homeDir = "/tmp"
				config := Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON

				// Git
				git.stagedDiff = "diff --git a/file.go"
				git.stagedFiles = "file.go"

				// HTTP
				response := AnthropicResponse{
					Content: []struct {
						Text string `json:"text"`
					}{
						{Text: "feat: add new feature"},
					},
				}
				responseJSON, _ := json.Marshal(response)
				http.response = createHTTPResponse(200, string(responseJSON))
			},
			expectErr:      false,
			expectedOutput: "✓ Commit message generated",
		},
		{
			name: "no staged changes",
			setupMocks: func(fs *MockFileSystem, git *MockGitClient, http *MockHTTPClient) {
				// Config
				fs.homeDir = "/tmp"
				config := Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON

				// Git - no staged changes
				git.stagedDiff = ""
				git.stagedFiles = ""
			},
			expectErr: true,
			errorMsg:  "no staged changes found",
		},
		{
			name: "git diff error",
			setupMocks: func(fs *MockFileSystem, git *MockGitClient, http *MockHTTPClient) {
				// Config
				fs.homeDir = "/tmp"
				config := Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON

				// Git error
				git.diffErr = errors.New("git diff error")
			},
			expectErr: true,
			errorMsg:  "git diff error",
		},
		{
			name: "git files error",
			setupMocks: func(fs *MockFileSystem, git *MockGitClient, http *MockHTTPClient) {
				// Config
				fs.homeDir = "/tmp"
				config := Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON

				// Git
				git.stagedDiff = "diff --git a/file.go"
				git.filesErr = errors.New("git files error")
			},
			expectErr: true,
			errorMsg:  "git files error",
		},
		{
			name: "config load error",
			setupMocks: func(fs *MockFileSystem, git *MockGitClient, http *MockHTTPClient) {
				// Config error
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("config not found")
			},
			expectErr: true,
			errorMsg:  "config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockGit := &MockGitClient{}
			mockHTTP := &MockHTTPClient{}
			mockPrinter := &MockPrinter{}

			tt.setupMocks(mockFS, mockGit, mockHTTP)

			configService := NewConfigService(mockFS, mockPrinter)
			anthropicService := NewAnthropicService(mockHTTP, mockPrinter)
			commitService := NewCommitService(configService, anthropicService, mockGit, mockPrinter)

			err := commitService.GenerateCommitMessage("", "", 1)

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
				if !mockPrinter.ContainsMessage(tt.expectedOutput) {
					t.Errorf("Expected output %q not found in messages: %v", tt.expectedOutput, mockPrinter.GetMessages())
				}
			}
		})
	}
}

// Test App integration
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
			model:     "test-model",
			expectErr: false,
		},
		{
			name:           "update only model with existing config",
			apiKey:         "",
			model:          "new-model",
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
			// Create app with real dependencies but mock filesystem
			mockFS := NewMockFileSystem()
			mockFS.homeDir = "/tmp"
			mockPrinter := &MockPrinter{}

			if tt.existingConfig {
				config := Config{ApiKey: "existing-api-key", Model: "existing-model"}
				configJSON, _ := json.Marshal(config)
				mockFS.readData = configJSON
			}

			configService := NewConfigService(mockFS, mockPrinter)
			app := &App{
				configService: configService,
				printer:       mockPrinter,
			}

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

// Test version functionality
func TestApp_ShowVersion(t *testing.T) {
	// Test with default "v0.0.0-dev" version
	mockPrinter := &MockPrinter{}
	app := &App{printer: mockPrinter}

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

// Test config help functionality
func TestApp_ShowConfigHelp(t *testing.T) {
	mockPrinter := &MockPrinter{}
	app := &App{printer: mockPrinter}

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

// Test help functionality
func TestApp_ShowHelp(t *testing.T) {
	mockPrinter := &MockPrinter{}
	app := &App{printer: mockPrinter}

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
	mockPrinter := &MockPrinter{}
	app := &App{printer: mockPrinter}

	app.HandleHelp()

	// Should call ShowHelp which prints help messages
	if len(mockPrinter.GetMessages()) == 0 {
		t.Error("Expected help messages from HandleHelp")
	}
}

// Test prompt building
func TestCommitService_buildPrompt(t *testing.T) {
	tests := []struct {
		name               string
		files              string
		diff               string
		commitType         string
		context            string
		count              int
		expectedElements   []string
		unexpectedElements []string
	}{
		{
			name:       "without type or context specified, count 1",
			files:      "main.go\ntest.go",
			diff:       "diff --git a/main.go",
			commitType: "",
			context:    "",
			count:      1,
			expectedElements: []string{
				"conventional commit message",
				"<type>: <description>",
				"feat:", "fix:", "docs:",
				"imperative mood",
				"Maximum 50 characters",
				"main.go\ntest.go",
				"diff --git a/main.go",
				"Commit message:",
			},
			unexpectedElements: []string{
				"commit type MUST be",
				"Additional context:",
				"different commit message options",
			},
		},
		{
			name:       "with type specified as feat",
			files:      "main.go\ntest.go",
			diff:       "diff --git a/main.go",
			commitType: "feat",
			context:    "",
			count:      1,
			expectedElements: []string{
				"conventional commit message",
				"<type>: <description>",
				"commit type MUST be 'feat'",
				"main.go\ntest.go",
				"diff --git a/main.go",
			},
			unexpectedElements: []string{
				"Additional context:",
			},
		},
		{
			name:       "with type specified as fix",
			files:      "api.go",
			diff:       "diff --git a/api.go",
			commitType: "fix",
			context:    "",
			count:      1,
			expectedElements: []string{
				"commit type MUST be 'fix'",
				"api.go",
			},
		},
		{
			name:       "with context specified",
			files:      "auth.go",
			diff:       "diff --git a/auth.go",
			commitType: "",
			context:    "fixing authentication bug",
			count:      1,
			expectedElements: []string{
				"Additional context: fixing authentication bug",
				"auth.go",
			},
			unexpectedElements: []string{
				"commit type MUST be",
			},
		},
		{
			name:       "with both type and context specified",
			files:      "api.go",
			diff:       "diff --git a/api.go",
			commitType: "fix",
			context:    "resolves issue #123",
			count:      1,
			expectedElements: []string{
				"commit type MUST be 'fix'",
				"Additional context: resolves issue #123",
				"api.go",
			},
		},
		{
			name:       "with count 3",
			files:      "main.go",
			diff:       "diff --git a/main.go",
			commitType: "",
			context:    "",
			count:      3,
			expectedElements: []string{
				"Generate 3 different commit message options",
				"each on a new line",
				"Commit messages (one per line):",
			},
			unexpectedElements: []string{
				"Commit message:",
			},
		},
		{
			name:       "with count 5 and type",
			files:      "api.go",
			diff:       "diff --git a/api.go",
			commitType: "feat",
			context:    "",
			count:      5,
			expectedElements: []string{
				"commit type MUST be 'feat'",
				"Generate 5 different commit message options",
				"Commit messages (one per line):",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &CommitService{}
			prompt := service.buildPrompt(tt.files, tt.diff, tt.commitType, tt.context, tt.count)

			for _, element := range tt.expectedElements {
				if !strings.Contains(prompt, element) {
					t.Errorf("Expected prompt to contain %q", element)
				}
			}

			for _, element := range tt.unexpectedElements {
				if strings.Contains(prompt, element) {
					t.Errorf("Expected prompt NOT to contain %q", element)
				}
			}
		})
	}
}

// Property-based tests for MaskAPIKey
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

// Test constants and global variables
func TestConstants(t *testing.T) {
	// Test that default model is in available models
	found := false
	for _, model := range AvailableModels {
		if model == DefaultModel {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DefaultModel %q should be in AvailableModels", DefaultModel)
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

// Test version variables
func TestVersionVariables(t *testing.T) {
	// Test that version variables are properly defined
	if version == "" {
		t.Error("version variable should not be empty")
	}

	if buildDate == "" {
		t.Error("buildDate variable should not be empty")
	}

	if commitSHA == "" {
		t.Error("commitSHA variable should not be empty")
	}

	// Test SemVer default values
	if version != "v0.0.0-dev" {
		t.Logf("Note: version is set to %q (not default 'v0.0.0-dev')", version)
	}

	if buildDate != "unknown" {
		t.Logf("Note: buildDate is set to %q (not default 'unknown')", buildDate)
	}

	if commitSHA != "unknown" {
		t.Logf("Note: commitSHA is set to %q (not default 'unknown')", commitSHA)
	}

	// Test SemVer format validation
	if !strings.HasPrefix(version, "v") {
		t.Errorf("Version should follow SemVer format and start with 'v', got: %q", version)
	}
}

// Benchmark tests
func BenchmarkMaskAPIKey(b *testing.B) {
	apiKey := "sk-ant-api03-1234567890abcdef1234567890abcdef"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		MaskAPIKey(apiKey)
	}
}

func BenchmarkConfigService_LoadConfig(b *testing.B) {
	mockFS := NewMockFileSystem()
	mockFS.homeDir = "/tmp"
	config := Config{ApiKey: "test-key", Model: "test-model"}
	configJSON, _ := json.Marshal(config)
	mockFS.readData = configJSON

	configService := NewConfigService(mockFS, &MockPrinter{})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := configService.LoadConfig()
		if err != nil {
			b.Fatal(err)
		}
	}
}
