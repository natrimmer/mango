package commit

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/natrimmer/claude_commit/internal/anthropic"
	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/filesystem"
	"github.com/natrimmer/claude_commit/internal/git"
	"github.com/natrimmer/claude_commit/internal/ui"
)

func TestService_GenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*filesystem.MockFileSystem, *git.MockClient, *anthropic.MockHTTPClient)
		expectErr      bool
		errorMsg       string
		expectedOutput string
	}{
		{
			name: "successful generation",
			setupMocks: func(fs *filesystem.MockFileSystem, gitClient *git.MockClient, httpClient *anthropic.MockHTTPClient) {
				// Config
				fs.SetHomeDir("/tmp")
				cfg := config.Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(cfg)
				fs.SetReadData(configJSON)

				// Git
				gitClient.SetStagedDiff("diff --git a/file.go")
				gitClient.SetStagedFiles("file.go")

				// HTTP
				response := anthropic.Response{
					Content: []struct {
						Text string `json:"text"`
					}{
						{Text: "feat: add new feature"},
					},
				}
				responseJSON, _ := json.Marshal(response)
				httpClient.SetResponse(anthropic.CreateHTTPResponse(200, string(responseJSON)))
			},
			expectErr:      false,
			expectedOutput: "✓ Commit message generated",
		},
		{
			name: "no staged changes",
			setupMocks: func(fs *filesystem.MockFileSystem, gitClient *git.MockClient, httpClient *anthropic.MockHTTPClient) {
				// Config
				fs.SetHomeDir("/tmp")
				cfg := config.Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(cfg)
				fs.SetReadData(configJSON)

				// Git - no staged changes
				gitClient.SetStagedDiff("")
				gitClient.SetStagedFiles("")
			},
			expectErr: true,
			errorMsg:  "no staged changes found",
		},
		{
			name: "git diff error",
			setupMocks: func(fs *filesystem.MockFileSystem, gitClient *git.MockClient, httpClient *anthropic.MockHTTPClient) {
				// Config
				fs.SetHomeDir("/tmp")
				cfg := config.Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(cfg)
				fs.SetReadData(configJSON)

				// Git error
				gitClient.SetDiffError(errors.New("git diff error"))
			},
			expectErr: true,
			errorMsg:  "git diff error",
		},
		{
			name: "git files error",
			setupMocks: func(fs *filesystem.MockFileSystem, gitClient *git.MockClient, httpClient *anthropic.MockHTTPClient) {
				// Config
				fs.SetHomeDir("/tmp")
				cfg := config.Config{ApiKey: "test-key", Model: "test-model"}
				configJSON, _ := json.Marshal(cfg)
				fs.SetReadData(configJSON)

				// Git
				gitClient.SetStagedDiff("diff --git a/file.go")
				gitClient.SetFilesError(errors.New("git files error"))
			},
			expectErr: true,
			errorMsg:  "git files error",
		},
		{
			name: "config load error",
			setupMocks: func(fs *filesystem.MockFileSystem, gitClient *git.MockClient, httpClient *anthropic.MockHTTPClient) {
				// Config error
				fs.SetHomeDir("/tmp")
				fs.SetReadError(errors.New("config not found"))
			},
			expectErr: true,
			errorMsg:  "config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem()
			mockGit := git.NewMockClient()
			mockHTTP := anthropic.NewMockHTTPClient()
			mockPrinter := ui.NewMockPrinter()

			tt.setupMocks(mockFS, mockGit, mockHTTP)

			configService := config.NewService(mockFS, mockPrinter)
			anthropicClient := anthropic.NewClient(mockHTTP, mockPrinter)
			commitService := NewService(configService, anthropicClient, mockGit, mockPrinter)

			err := commitService.GenerateCommitMessage("", "", 1, false, false)

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

func TestService_buildPrompt(t *testing.T) {
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
			service := &Service{}
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

func TestService_DryRunAndVerbose(t *testing.T) {
	tests := []struct {
		name           string
		dryRun         bool
		verbose        bool
		expectAPICall  bool
		expectedOutput []string
	}{
		{
			name:          "dry-run mode shows prompt, no API call",
			dryRun:        true,
			verbose:       false,
			expectAPICall: false,
			expectedOutput: []string{
				"Prompt being sent to Claude:",
				"Dry run mode - API not called",
			},
		},
		{
			name:          "verbose mode shows prompt and response",
			dryRun:        false,
			verbose:       true,
			expectAPICall: true,
			expectedOutput: []string{
				"Prompt being sent to Claude:",
				"Raw API Response:",
			},
		},
		{
			name:          "both flags: dry-run takes precedence",
			dryRun:        true,
			verbose:       true,
			expectAPICall: false,
			expectedOutput: []string{
				"Prompt being sent to Claude:",
				"Dry run mode - API not called",
			},
		},
		{
			name:          "normal mode: no extra output",
			dryRun:        false,
			verbose:       false,
			expectAPICall: true,
			expectedOutput: []string{
				"✓ Commit message generated",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := filesystem.NewMockFileSystem()
			mockGit := git.NewMockClient()
			mockHTTP := anthropic.NewMockHTTPClient()
			mockPrinter := ui.NewMockPrinter()

			// Setup
			mockFS.SetHomeDir("/tmp")
			cfg := config.Config{ApiKey: "test-key", Model: "test-model"}
			configJSON, _ := json.Marshal(cfg)
			mockFS.SetReadData(configJSON)

			mockGit.SetStagedDiff("diff --git a/file.go")
			mockGit.SetStagedFiles("file.go")

			response := anthropic.Response{
				Content: []struct {
					Text string `json:"text"`
				}{
					{Text: "feat: add new feature"},
				},
			}
			responseJSON, _ := json.Marshal(response)
			mockHTTP.SetResponse(anthropic.CreateHTTPResponse(200, string(responseJSON)))

			configService := config.NewService(mockFS, mockPrinter)
			anthropicClient := anthropic.NewClient(mockHTTP, mockPrinter)
			commitService := NewService(configService, anthropicClient, mockGit, mockPrinter)

			err := commitService.GenerateCommitMessage("", "", 1, tt.dryRun, tt.verbose)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !mockPrinter.ContainsMessage(expected) {
					t.Errorf("Expected output to contain %q, messages: %v", expected, mockPrinter.GetMessages())
				}
			}

			// Note: We can't easily verify if API was called without exposing mock internals
			// The test already verifies behavior through output checking
		})
	}
}
