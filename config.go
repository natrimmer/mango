package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

const defaultModel = "claude-sonnet-5-5"

// Providers are the two ways mango can reach a model: a direct API call with a
// stored key, or the locally installed `claude` CLI in headless mode.
const (
	providerAPI        = "api"
	providerClaudeCode = "claude-code"
	defaultProvider    = providerAPI
)

var availableProviders = []string{providerAPI, providerClaudeCode}

var availableModels = []string{
	"claude-fable-5-1",
	"claude-opus-5-5",
	"claude-opus-5",
	"claude-opus-4-8",
	"claude-sonnet-5-5",
	"claude-sonnet-5",
	"claude-sonnet-4-6",
	"claude-haiku-5-5",
	"claude-haiku-4-5",
}

// Config represents the application configuration.
type Config struct {
	Provider string `json:"provider"`
	ApiKey   string `json:"api_key"`
	Model    string `json:"model"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}
	return filepath.Join(home, ".mango", "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w\nRun: mango config set --api-key \"sk-ant-...\"\nOr, to use the claude CLI: mango config set --provider claude-code", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}
	// Configs written before providers existed carry no provider field.
	if cfg.Provider == "" {
		cfg.Provider = defaultProvider
	}
	return &cfg, nil
}

// saveConfig merges the provided fields over any existing config and writes it.
func saveConfig(provider, apiKey, model string) error {
	cfg := Config{Provider: defaultProvider, Model: defaultModel}
	if existing, err := loadConfig(); err == nil {
		cfg = *existing
	}
	if provider != "" {
		cfg.Provider = provider
	}
	if apiKey != "" {
		cfg.ApiKey = apiKey
	}
	if model != "" {
		cfg.Model = model
	}

	if !slices.Contains(availableProviders, cfg.Provider) {
		return fmt.Errorf("invalid provider %q. Valid providers: %s", cfg.Provider, strings.Join(availableProviders, ", "))
	}
	// The claude CLI carries its own credentials, so no key to require.
	if cfg.Provider == providerAPI && cfg.ApiKey == "" {
		return fmt.Errorf("API key is required. Use --api-key flag to set it")
	}
	if !slices.Contains(availableModels, cfg.Model) {
		return fmt.Errorf("invalid model %q. Run 'mango config models' to see available models", cfg.Model)
	}

	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	printSuccess("Configuration saved successfully")
	printConfig(cfg)
	return nil
}

// printConfig renders a config, hiding the key line where none is used.
func printConfig(cfg Config) {
	fmt.Println(Bold + "Provider: " + Reset + cfg.Provider)
	if cfg.Provider == providerAPI {
		fmt.Println(Bold + "API Key: " + Reset + maskAPIKey(cfg.ApiKey))
	}
	fmt.Println(Bold + "Model: " + Reset + cfg.Model)
}

// maskAPIKey masks an API key for display.
func maskAPIKey(k string) string {
	if len(k) <= 8 {
		return "********"
	}
	return k[:4] + "****" + k[len(k)-4:]
}

// configCmd is a parent grouping setup under one noun; bare `mango config`
// prints help listing the subcommands.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage provider, API key and model",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set provider, API key and/or model",
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		apiKey, _ := cmd.Flags().GetString("api-key")
		model, _ := cmd.Flags().GetString("model")
		if provider == "" && apiKey == "" && model == "" {
			return cmd.Help()
		}
		return saveConfig(provider, apiKey, model)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		fmt.Println(Bold + Cyan + "Current Configuration:" + Reset)
		printConfig(*cfg)
		return nil
	},
}

var configModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Config is optional here — listing static models needs no API key.
		// If it loads, mark the active one.
		current := ""
		if cfg, err := loadConfig(); err == nil {
			current = cfg.Model
		}
		fmt.Println(Bold + Cyan + "Available Models:" + Reset)
		for _, m := range availableModels {
			switch m {
			case current:
				fmt.Println(Bold + Green + m + " [CURRENT]" + Reset)
			case defaultModel:
				fmt.Println(Bold + m + " [DEFAULT]" + Reset)
			default:
				fmt.Println(Bold + m + Reset)
			}
		}
		return nil
	},
}

func init() {
	configSetCmd.Flags().String("provider", "", "How to reach the model: api or claude-code")
	configSetCmd.Flags().String("api-key", "", "API key for the model provider")
	configSetCmd.Flags().String("model", "", "Model to use")
	configCmd.AddCommand(configSetCmd, configShowCmd, configModelsCmd)
	rootCmd.AddCommand(configCmd)
}
