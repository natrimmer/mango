package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
)

const defaultModel = "claude-sonnet-4-6"

var availableModels = []string{
	"claude-opus-4-8",
	"claude-sonnet-4-6",
	"claude-haiku-4-5",
}

// Config represents the application configuration.
type Config struct {
	ApiKey string `json:"api_key"`
	Model  string `json:"model"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}
	return filepath.Join(home, ".claude-commit", "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w\nPlease run 'config' first", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}
	return &cfg, nil
}

// saveConfig merges the provided fields over any existing config and writes it.
func saveConfig(apiKey, model string) error {
	cfg := Config{Model: defaultModel}
	if existing, err := loadConfig(); err == nil {
		cfg = *existing
	}
	if apiKey != "" {
		cfg.ApiKey = apiKey
	}
	if model != "" {
		cfg.Model = model
	}

	if cfg.ApiKey == "" {
		return fmt.Errorf("API key is required. Use --api-key flag to set it")
	}
	if !slices.Contains(availableModels, cfg.Model) {
		return fmt.Errorf("invalid model %q. Run 'models' to see available models", cfg.Model)
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
	fmt.Println(Bold + "API Key: " + Reset + maskAPIKey(cfg.ApiKey))
	fmt.Println(Bold + "Model: " + Reset + cfg.Model)
	return nil
}

// maskAPIKey masks an API key for display.
func maskAPIKey(k string) string {
	if len(k) <= 8 {
		return "********"
	}
	return k[:4] + "****" + k[len(k)-4:]
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure API key and model",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, _ := cmd.Flags().GetString("api-key")
		model, _ := cmd.Flags().GetString("model")
		if apiKey == "" && model == "" {
			return cmd.Help()
		}
		return saveConfig(apiKey, model)
	},
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		fmt.Println(Bold + Cyan + "Current Configuration:" + Reset)
		fmt.Println(Bold + "API Key: " + Reset + maskAPIKey(cfg.ApiKey))
		fmt.Println(Bold + "Model: " + Reset + cfg.Model)
		return nil
	},
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		fmt.Println(Bold + Cyan + "Available Models:" + Reset)
		for _, m := range availableModels {
			switch m {
			case cfg.Model:
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
	configCmd.Flags().String("api-key", "", "Anthropic API key")
	configCmd.Flags().String("model", "", "Anthropic model to use")
	rootCmd.AddCommand(configCmd, viewCmd, modelsCmd)
}
