package app

import (
	"github.com/natrimmer/claude_commit/internal/commit"
	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/model"
	"github.com/natrimmer/claude_commit/internal/ui"
)

// Version information - can be set at build time with ldflags
var (
	Version   = "v0.0.0-dev" // Default SemVer version for development builds
	BuildDate = "unknown"    // Build date
	CommitSHA = "unknown"    // Git commit SHA
)

// App coordinates all services and handles command routing
type App struct {
	configService *config.Service
	modelService  *model.Service
	commitService *commit.Service
	printer       ui.Printer
}

// New creates a new application instance
func New(configService *config.Service, modelService *model.Service, commitService *commit.Service, printer ui.Printer) *App {
	return &App{
		configService: configService,
		modelService:  modelService,
		commitService: commitService,
		printer:       printer,
	}
}

// HandleConfig handles the config command
func (app *App) HandleConfig(apiKey, model string) error {
	return app.configService.SaveConfig(apiKey, model)
}

// HandleView handles the view command
func (app *App) HandleView() error {
	return app.configService.ViewConfig()
}

// HandleModels handles the models command
func (app *App) HandleModels() error {
	return app.modelService.ShowModels()
}

// HandleHelp handles the help command
func (app *App) HandleHelp() {
	app.ShowHelp()
}

// HandleCommit handles the commit command
func (app *App) HandleCommit(commitType, context string, count int, dryRun, verbose bool) error {
	return app.commitService.GenerateCommitMessage(commitType, context, count, dryRun, verbose)
}

// ShowVersion displays version information
func (app *App) ShowVersion() {
	app.printer.Print(ui.Bold + ui.Magenta + "Claude Commit" + ui.Reset + " " + ui.Dim + Version + ui.Reset)
	if Version != "v0.0.0-dev" {
		app.printer.Print(ui.Dim + "Build Date: " + BuildDate + ui.Reset)
		app.printer.Print(ui.Dim + "Commit: " + CommitSHA + ui.Reset)
	}
	app.printer.Print(ui.Dim + "Generate conventional commit messages with Anthropic's Claude" + ui.Reset)
}

// ShowConfigHelp displays help for the config command
func (app *App) ShowConfigHelp() {
	app.printer.Print(ui.Bold + ui.Magenta + "Claude Commit Config" + ui.Reset)
	app.printer.Print("Configure API key and model settings")
	app.printer.Print("")
	app.printer.Print(ui.Bold + "Usage:" + ui.Reset)
	app.printer.Print("  claude_commit config [flags]")
	app.printer.Print("")
	app.printer.Print(ui.Bold + "Flags:" + ui.Reset)
	app.printer.Print("  -api-key string   Anthropic API key")
	app.printer.Print("  -model string     Anthropic model to use")
	app.printer.Print("")
	app.printer.Print(ui.Bold + "Examples:" + ui.Reset)
	app.printer.Print("  # Initial setup (API key required)")
	app.printer.Print("  claude_commit config -api-key \"sk-ant-api03-...\" -model \"claude-sonnet-4-6\"")
	app.printer.Print("")
	app.printer.Print("  # Update only API key")
	app.printer.Print("  claude_commit config -api-key \"sk-ant-api03-...\"")
	app.printer.Print("")
	app.printer.Print("  # Update only model")
	app.printer.Print("  claude_commit config -model \"claude-haiku-4-5\"")
	app.printer.Print("")
	app.printer.Print("Use 'claude_commit view' to see current configuration")
	app.printer.Print("Use 'claude_commit models' to see available models")
}

// ShowHelp displays general help information
func (app *App) ShowHelp() {
	app.printer.Print(ui.Bold + ui.Magenta + "Claude Commit" + ui.Reset + " " + ui.Dim + Version + ui.Reset)
	app.printer.Print(ui.Dim + ui.Magenta + "Generate conventional commit messages with Anthropic's Claude" + ui.Reset)
	app.printer.Print("")
	app.printer.Print(ui.Bold + "Commands:" + ui.Reset)
	app.printer.Print("  config    Configure API key and model")
	app.printer.Print("  view      View current configuration")
	app.printer.Print("  models    List available models")
	app.printer.Print("  commit    Generate commit message")
	app.printer.Print("  help      Show this help message")
	app.printer.Print("")
	app.printer.Print(ui.Bold + "Flags:" + ui.Reset)
	app.printer.Print("  --version, -v    Show version information")
	app.printer.Print("  --help, -h       Show this help message")

	// Show usage examples
	app.printer.Print("\n" + ui.Bold + "Examples:" + ui.Reset)
	app.printer.Print("  claude_commit config -api-key \"your-api-key\" -model \"claude-sonnet-4-6\"")
	app.printer.Print("  claude_commit config -api-key \"your-api-key\"  # Set only API key")
	app.printer.Print("  claude_commit config -model \"claude-haiku-4-5\"  # Update only model")
	app.printer.Print("  claude_commit view")
	app.printer.Print("  claude_commit models")
	app.printer.Print("  claude_commit commit")
	app.printer.Print("  claude_commit commit --type feat  # Force commit type")
	app.printer.Print("  claude_commit commit --context \"fixing authentication bug\"  # Add context")
	app.printer.Print("  claude_commit commit --count 3  # Generate 3 options")
	app.printer.Print("  claude_commit commit --dry-run  # Show prompt without API call")
	app.printer.Print("  claude_commit commit --verbose  # Show prompt and API interaction")
	app.printer.Print("  claude_commit commit -v  # Short form of --verbose")
	app.printer.Print("  claude_commit commit --type fix --context \"resolves issue #123\"  # Combine flags")
	app.printer.Print("  claude_commit --version")

	// Show conventional commit info
	app.printer.Print("\n" + ui.Bold + "Commit Types:" + ui.Reset)
	app.printer.Print("  feat:     A new feature")
	app.printer.Print("  fix:      A bug fix")
	app.printer.Print("  docs:     Documentation changes")
	app.printer.Print("  style:    Code style changes (formatting, etc.)")
	app.printer.Print("  refactor: Code refactoring without changes to functionality")
	app.printer.Print("  perf:     Performance improvements")
	app.printer.Print("  test:     Adding or updating tests")
	app.printer.Print("  chore:    Maintenance tasks, dependency updates, etc.")
	app.printer.Print("  ci:       Continuous integration changes")
	app.printer.Print("  build:    Changes that affect the build system or external dependencies")
	app.printer.Print("  revert:   Reverts a previous commit")
}
