package anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/ui"
)

// HTTPClient defines the interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Request represents an Anthropic API request
type Request struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents an Anthropic API response
type Response struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Client handles communication with the Anthropic API
type Client struct {
	httpClient HTTPClient
	printer    ui.Printer
}

// NewClient creates a new Anthropic API client
func NewClient(httpClient HTTPClient, printer ui.Printer) *Client {
	return &Client{
		httpClient: httpClient,
		printer:    printer,
	}
}

// GenerateCommitMessage generates a commit message using the Anthropic API
func (c *Client) GenerateCommitMessage(cfg config.Config, prompt string) (string, error) {
	requestBody := Request{
		Model: cfg.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 50,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.ApiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making API call: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.printer.PrintError(fmt.Sprintf("Error closing response body: %v", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	var anthropicResp Response
	err = json.NewDecoder(resp.Body).Decode(&anthropicResp)
	if err != nil {
		return "", fmt.Errorf("error parsing API response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return anthropicResp.Content[0].Text, nil
}
