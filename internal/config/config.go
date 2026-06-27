package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/natrimmer/claude_commit/internal/filesystem"
	"github.com/natrimmer/claude_commit/internal/ui"
)

// Config represents the application configuration
type Config struct {
	ApiKey string `json:"api_key"`
	Model  string `json:"model"`
}

// DefaultModel is the default Claude model to use
const DefaultModel = "claude-sonnet-4-6"

// AvailableModels lists all supported Claude models
var AvailableModels = []string{
	"claude-opus-4-8",
	"claude-sonnet-4-6",
	"claude-haiku-4-5",
}

// Service handles configuration operations
type Service struct {
	fs      filesystem.FileSystem
	printer ui.Printer
}

// NewService creates a new config service
func NewService(fs filesystem.FileSystem, printer ui.Printer) *Service {
	return &Service{fs: fs, printer: printer}
}

// SaveConfig saves the configuration to disk
func (cs *Service) SaveConfig(apiKey, model string) error {
	// Load existing config if it exists
	existingConfig, _ := cs.LoadConfig()

	// Start with existing config or create new one
	config := Config{
		ApiKey: "",
		Model:  DefaultModel,
	}

	if existingConfig != nil {
		config = *existingConfig
	}

	// Update only the fields that were provided
	if apiKey != "" {
		config.ApiKey = apiKey
	}

	if model != "" {
		config.Model = model
	}

	// Validate that we have an API key (either from existing config or new input)
	if config.ApiKey == "" {
		return fmt.Errorf("API key is required. Use -api-key flag to set it")
	}

	if !isValidModel(config.Model) {
		return fmt.Errorf("invalid model %q. Run 'models' to see available models", config.Model)
	}

	homeDir, err := cs.fs.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".claude-commit")
	err = cs.fs.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	err = cs.fs.WriteFile(configFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	cs.printer.PrintSuccess("Configuration saved successfully")
	cs.printer.Print(ui.Bold + "API Key: " + ui.Reset + MaskAPIKey(config.ApiKey))
	cs.printer.Print(ui.Bold + "Model: " + ui.Reset + config.Model)

	return nil
}

// LoadConfig loads the configuration from disk
func (cs *Service) LoadConfig() (*Config, error) {
	homeDir, err := cs.fs.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	configFile := filepath.Join(homeDir, ".claude-commit", "config.json")
	data, err := cs.fs.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w\nPlease run 'config' first", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}

// ViewConfig displays the current configuration
func (cs *Service) ViewConfig() error {
	config, err := cs.LoadConfig()
	if err != nil {
		return err
	}

	cs.printer.Print(ui.Bold + ui.Cyan + "Current Configuration:" + ui.Reset)
	cs.printer.Print(ui.Bold + "API Key: " + ui.Reset + MaskAPIKey(config.ApiKey))
	cs.printer.Print(ui.Bold + "Model: " + ui.Reset + config.Model)

	return nil
}

// isValidModel reports whether model is in AvailableModels
func isValidModel(model string) bool {
	return slices.Contains(AvailableModels, model)
}

// MaskAPIKey masks an API key for display
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "********"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}
