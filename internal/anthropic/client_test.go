package anthropic

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/ui"
)

func TestClient_GenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		prompt      string
		setupMock   func(*MockHTTPClient)
		expectErr   bool
		expectedMsg string
		errorMsg    string
	}{
		{
			name:   "successful generation",
			config: config.Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				response := Response{
					Content: []struct {
						Text string `json:"text"`
					}{
						{Text: "feat: add new feature"},
					},
				}
				responseJSON, _ := json.Marshal(response)
				client.SetResponse(CreateHTTPResponse(200, string(responseJSON)))
			},
			expectErr:   false,
			expectedMsg: "feat: add new feature",
		},
		{
			name:   "HTTP client error",
			config: config.Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				client.SetError(errors.New("network error"))
			},
			expectErr: true,
			errorMsg:  "error making API call",
		},
		{
			name:   "API error response",
			config: config.Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				client.SetResponse(CreateHTTPResponse(401, `{"error": "unauthorized"}`))
			},
			expectErr: true,
			errorMsg:  "API error",
		},
		{
			name:   "empty response content",
			config: config.Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				response := Response{Content: []struct {
					Text string `json:"text"`
				}{}}
				responseJSON, _ := json.Marshal(response)
				client.SetResponse(CreateHTTPResponse(200, string(responseJSON)))
			},
			expectErr: true,
			errorMsg:  "empty response from API",
		},
		{
			name:   "invalid JSON response",
			config: config.Config{ApiKey: "test-key", Model: "test-model"},
			prompt: "test prompt",
			setupMock: func(client *MockHTTPClient) {
				client.SetResponse(CreateHTTPResponse(200, "invalid json"))
			},
			expectErr: true,
			errorMsg:  "error parsing API response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := NewMockHTTPClient()
			mockPrinter := ui.NewMockPrinter()
			tt.setupMock(mockHTTP)

			client := NewClient(mockHTTP, mockPrinter)
			result, err := client.GenerateCommitMessage(tt.config, tt.prompt)

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
