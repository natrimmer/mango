package model

import (
	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/ui"
)

// AvailableModels lists all supported Claude models
var AvailableModels = []string{
	"claude-opus-4-0",
	"claude-sonnet-4-0",
	"claude-3-7-sonnet-latest",
	"claude-3-5-sonnet-latest",
	"claude-3-5-haiku-latest",
	"claude-3-opus-latest",
}

// Service handles model-related operations
type Service struct {
	configService *config.Service
	printer       ui.Printer
}

// NewService creates a new model service
func NewService(configService *config.Service, printer ui.Printer) *Service {
	return &Service{
		configService: configService,
		printer:       printer,
	}
}

// ShowModels displays all available models with the current selection highlighted
func (ms *Service) ShowModels() error {
	cfg, err := ms.configService.LoadConfig()
	if err != nil {
		return err
	}

	ms.printer.Print(ui.Bold + ui.Cyan + "Available Models:" + ui.Reset)
	for _, model := range AvailableModels {
		switch model {
		case cfg.Model:
			ms.printer.Print(ui.Bold + ui.Green + model + " [CURRENT]" + ui.Reset)
		case config.DefaultModel:
			ms.printer.Print(ui.Bold + model + " [DEFAULT]" + ui.Reset)
		default:
			ms.printer.Print(ui.Bold + model + ui.Reset)
		}
	}

	return nil
}
