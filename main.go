package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version information - can be set at build time with ldflags
var (
	version   = "v0.0.0-dev" // Default SemVer version for development builds
	buildDate = "unknown"    // Build date
	commitSHA = "unknown"    // Git commit SHA
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// Domain types
type Config struct {
	ApiKey string `json:"api_key"`
	Model  string `json:"model"`
}

type AnthropicRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Interfaces for dependency injection
type FileSystem interface {
	UserHomeDir() (string, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type GitClient interface {
	GetStagedDiff() (string, error)
	GetStagedFiles() (string, error)
}

type Printer interface {
	Print(msg string)
	PrintSuccess(msg string)
	PrintError(msg string)
	PrintWarning(msg string)
}

// Real implementations
type RealFileSystem struct{}

func (fs *RealFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *RealFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (fs *RealFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

type RealGitClient struct{}

func (gc *RealGitClient) GetStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--staged")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running git diff: %w", err)
	}
	return out.String(), nil
}

func (gc *RealGitClient) GetStagedFiles() (string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error getting changed files: %w", err)
	}
	return out.String(), nil
}

type ConsolePrinter struct{}

func (p *ConsolePrinter) Print(msg string) {
	fmt.Println(msg)
}

func (p *ConsolePrinter) PrintSuccess(msg string) {
	fmt.Println(Green + msg + Reset)
}

func (p *ConsolePrinter) PrintError(msg string) {
	fmt.Println(Red + msg + Reset)
}

func (p *ConsolePrinter) PrintWarning(msg string) {
	fmt.Println(Yellow + msg + Reset)
}

// Services
type ConfigService struct {
	fs      FileSystem
	printer Printer
}

func NewConfigService(fs FileSystem, printer Printer) *ConfigService {
	return &ConfigService{fs: fs, printer: printer}
}

func (cs *ConfigService) SaveConfig(apiKey, model string) error {
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
	cs.printer.Print(Bold + "API Key: " + Reset + MaskAPIKey(config.ApiKey))
	cs.printer.Print(Bold + "Model: " + Reset + config.Model)

	return nil
}

func (cs *ConfigService) LoadConfig() (*Config, error) {
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

func (cs *ConfigService) ViewConfig() error {
	config, err := cs.LoadConfig()
	if err != nil {
		return err
	}

	cs.printer.Print(Bold + Cyan + "Current Configuration:" + Reset)
	cs.printer.Print(Bold + "API Key: " + Reset + MaskAPIKey(config.ApiKey))
	cs.printer.Print(Bold + "Model: " + Reset + config.Model)

	return nil
}

type ModelService struct {
	configService *ConfigService
	printer       Printer
}

func NewModelService(configService *ConfigService, printer Printer) *ModelService {
	return &ModelService{
		configService: configService,
		printer:       printer,
	}
}

var AvailableModels = []string{
	"claude-opus-4-0",
	"claude-sonnet-4-0",
	"claude-3-7-sonnet-latest",
	"claude-3-5-sonnet-latest",
	"claude-3-5-haiku-latest",
	"claude-3-opus-latest",
}

const DefaultModel = "claude-3-7-sonnet-latest"

func (ms *ModelService) ShowModels() error {
	config, err := ms.configService.LoadConfig()
	if err != nil {
		return err
	}

	ms.printer.Print(Bold + Cyan + "Available Models:" + Reset)
	for _, model := range AvailableModels {
		switch model {
		case config.Model:
			ms.printer.Print(Bold + Green + model + " [CURRENT]" + Reset)
		case DefaultModel:
			ms.printer.Print(Bold + model + " [DEFAULT]" + Reset)
		default:
			ms.printer.Print(Bold + model + Reset)
		}
	}

	return nil
}

type AnthropicService struct {
	client  HTTPClient
	printer Printer
}

func NewAnthropicService(client HTTPClient, printer Printer) *AnthropicService {
	return &AnthropicService{
		client:  client,
		printer: printer,
	}
}

func (as *AnthropicService) GenerateCommitMessage(config Config, prompt string) (string, error) {
	requestBody := AnthropicRequest{
		Model: config.Model,
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
	req.Header.Set("x-api-key", config.ApiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := as.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making API call: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			as.printer.PrintError(fmt.Sprintf("Error closing response body: %v", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	var anthropicResp AnthropicResponse
	err = json.NewDecoder(resp.Body).Decode(&anthropicResp)
	if err != nil {
		return "", fmt.Errorf("error parsing API response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return anthropicResp.Content[0].Text, nil
}

type CommitService struct {
	configService    *ConfigService
	anthropicService *AnthropicService
	gitClient        GitClient
	printer          Printer
}

func NewCommitService(configService *ConfigService, anthropicService *AnthropicService, gitClient GitClient, printer Printer) *CommitService {
	return &CommitService{
		configService:    configService,
		anthropicService: anthropicService,
		gitClient:        gitClient,
		printer:          printer,
	}
}

func (cs *CommitService) GenerateCommitMessage(commitType, context string, count int) error {
	config, err := cs.configService.LoadConfig()
	if err != nil {
		return err
	}

	diff, err := cs.gitClient.GetStagedDiff()
	if err != nil {
		return err
	}

	files, err := cs.gitClient.GetStagedFiles()
	if err != nil {
		return err
	}

	if strings.TrimSpace(diff) == "" {
		return fmt.Errorf("no staged changes found. Use git add to stage changes")
	}

	cs.printer.Print(Dim + "⚙️  Analyzing git diff with Claude AI..." + Reset)

	prompt := cs.buildPrompt(files, diff, commitType, context, count)

	commitMsg, err := cs.anthropicService.GenerateCommitMessage(*config, prompt)
	if err != nil {
		return err
	}

	commitMsg = strings.TrimSpace(commitMsg)

	if count > 1 {
		// Multiple messages - display them numbered
		cs.printer.PrintSuccess("✓ Commit message options generated")
		cs.printer.Print("")
		messages := strings.Split(commitMsg, "\n")
		for i, msg := range messages {
			msg = strings.TrimSpace(msg)
			if msg != "" {
				cs.printer.Print(fmt.Sprintf("%s%d.%s %s", Bold, i+1, Reset, msg))
			}
		}
	} else {
		// Single message - display git command
		gitCommand := fmt.Sprintf("git commit -m \"%s\"", commitMsg)
		cs.printer.PrintSuccess("✓ Commit message generated")
		cs.printer.Print("")
		cs.printer.Print(Bold + gitCommand + Reset)
	}

	return nil
}

func (cs *CommitService) buildPrompt(files, diff, commitType, context string, count int) string {
	typeInstruction := ""
	if commitType != "" {
		typeInstruction = fmt.Sprintf("\nIMPORTANT: The commit type MUST be '%s'.", commitType)
	}

	contextInstruction := ""
	if context != "" {
		contextInstruction = fmt.Sprintf("\n\nAdditional context: %s", context)
	}

	countInstruction := ""
	outputFormat := "Commit message:"
	if count > 1 {
		countInstruction = fmt.Sprintf("\nGenerate %d different commit message options, each on a new line.", count)
		outputFormat = "Commit messages (one per line):"
	}

	return fmt.Sprintf(`Generate a conventional commit message based on the following git diff.

IMPORTANT: Return ONLY the commit message(s), nothing else. No explanations, no analysis, no additional text.%s%s

The message should follow this format: <type>: <description>

Types include:
- feat: A new feature
- fix: A bug fix
- docs: Documentation changes
- style: Code style changes (formatting, etc.)
- refactor: Code refactoring without changes to functionality
- perf: Performance improvements
- test: Adding or updating tests
- chore: Maintenance tasks, dependency updates, etc.
- ci: Continuous integration changes
- build: Changes that affect the build system or external dependencies
- revert: Reverts a previous commit

Guidelines:
1. Use the imperative mood ("add feature" not "Added feature")
2. All lowercase characters
3. No period at the end
4. Be concise but descriptive (what was changed and why)
5. Maximum 50 characters
6. Return ONLY the commit message(s), no other text%s

Here are the files changed:
%s

Here is the git diff:
%s

%s`, typeInstruction, countInstruction, contextInstruction, files, diff, outputFormat)
}

// Utility functions
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "********"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// App struct to hold all dependencies
type App struct {
	configService    *ConfigService
	modelService     *ModelService
	commitService    *CommitService
	anthropicService *AnthropicService
	printer          Printer
}

func NewApp() *App {
	// Real dependencies
	fs := &RealFileSystem{}
	httpClient := &http.Client{}
	gitClient := &RealGitClient{}
	printer := &ConsolePrinter{}

	// Services
	configService := NewConfigService(fs, printer)
	anthropicService := NewAnthropicService(httpClient, printer)
	modelService := NewModelService(configService, printer)
	commitService := NewCommitService(configService, anthropicService, gitClient, printer)

	return &App{
		configService:    configService,
		modelService:     modelService,
		commitService:    commitService,
		anthropicService: anthropicService,
		printer:          printer,
	}
}

// Command handlers
func (app *App) HandleConfig(apiKey, model string) error {
	return app.configService.SaveConfig(apiKey, model)
}

func (app *App) HandleView() error {
	return app.configService.ViewConfig()
}

func (app *App) HandleModels() error {
	return app.modelService.ShowModels()
}

func (app *App) HandleHelp() {
	app.ShowHelp()
}

func (app *App) HandleCommit(commitType, context string, count int) error {
	return app.commitService.GenerateCommitMessage(commitType, context, count)
}

func (app *App) ShowVersion() {
	app.printer.Print(Bold + Magenta + "Claude Commit" + Reset + " " + Dim + version + Reset)
	if version != "v0.0.0-dev" {
		app.printer.Print(Dim + "Build Date: " + buildDate + Reset)
		app.printer.Print(Dim + "Commit: " + commitSHA + Reset)
	}
	app.printer.Print(Dim + "Generate conventional commit messages with Anthropic's Claude" + Reset)
}

func (app *App) ShowConfigHelp() {
	app.printer.Print(Bold + Magenta + "Claude Commit Config" + Reset)
	app.printer.Print("Configure API key and model settings")
	app.printer.Print("")
	app.printer.Print(Bold + "Usage:" + Reset)
	app.printer.Print("  claude_commit config [flags]")
	app.printer.Print("")
	app.printer.Print(Bold + "Flags:" + Reset)
	app.printer.Print("  -api-key string   Anthropic API key")
	app.printer.Print("  -model string     Anthropic model to use")
	app.printer.Print("")
	app.printer.Print(Bold + "Examples:" + Reset)
	app.printer.Print("  # Initial setup (API key required)")
	app.printer.Print("  claude_commit config -api-key \"sk-ant-api03-...\" -model \"claude-3-7-sonnet-latest\"")
	app.printer.Print("")
	app.printer.Print("  # Update only API key")
	app.printer.Print("  claude_commit config -api-key \"sk-ant-api03-...\"")
	app.printer.Print("")
	app.printer.Print("  # Update only model")
	app.printer.Print("  claude_commit config -model \"claude-3-5-sonnet-latest\"")
	app.printer.Print("")
	app.printer.Print("Use 'claude_commit view' to see current configuration")
	app.printer.Print("Use 'claude_commit models' to see available models")
}

func (app *App) ShowHelp() {
	app.printer.Print(Bold + Magenta + "Claude Commit" + Reset + " " + Dim + version + Reset)
	app.printer.Print(Dim + Magenta + "Generate conventional commit messages with Anthropic's Claude" + Reset)
	app.printer.Print("")
	app.printer.Print(Bold + "Commands:" + Reset)
	app.printer.Print("  config    Configure API key and model")
	app.printer.Print("  view      View current configuration")
	app.printer.Print("  models    List available models")
	app.printer.Print("  commit    Generate commit message")
	app.printer.Print("  help      Show this help message")
	app.printer.Print("")
	app.printer.Print(Bold + "Flags:" + Reset)
	app.printer.Print("  --version, -v    Show version information")
	app.printer.Print("  --help, -h       Show this help message")

	// Show usage examples
	app.printer.Print("\n" + Bold + "Examples:" + Reset)
	app.printer.Print("  claude_commit config -api-key \"your-api-key\" -model \"claude-3-7-sonnet-latest\"")
	app.printer.Print("  claude_commit config -api-key \"your-api-key\"  # Set only API key")
	app.printer.Print("  claude_commit config -model \"claude-3-5-sonnet-latest\"  # Update only model")
	app.printer.Print("  claude_commit view")
	app.printer.Print("  claude_commit models")
	app.printer.Print("  claude_commit commit")
	app.printer.Print("  claude_commit commit --type feat  # Force commit type")
	app.printer.Print("  claude_commit commit --context \"fixing authentication bug\"  # Add context")
	app.printer.Print("  claude_commit commit --count 3  # Generate 3 options")
	app.printer.Print("  claude_commit commit --type fix --context \"resolves issue #123\"  # Combine flags")
	app.printer.Print("  claude_commit --version")

	// Show conventional commit info
	app.printer.Print("\n" + Bold + "Commit Types:" + Reset)
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

func main() {
	app := NewApp()

	// Handle global flags first
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-v":
			app.ShowVersion()
			return
		case "--help", "-h":
			app.ShowHelp()
			return
		}
	}

	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	apiKey := configCmd.String("api-key", "", "Anthropic API key")
	model := configCmd.String("model", DefaultModel, "Anthropic model to use")

	commitCmd := flag.NewFlagSet("commit", flag.ExitOnError)
	commitType := commitCmd.String("type", "", "Commit type (feat, fix, docs, etc.)")
	commitContext := commitCmd.String("context", "", "Additional context to guide commit message generation")
	commitCount := commitCmd.Int("count", 1, "Number of commit message options to generate")
	viewCmd := flag.NewFlagSet("view", flag.ExitOnError)
	modelsCmd := flag.NewFlagSet("models", flag.ExitOnError)
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)

	// If no arguments provided, show help instead of error
	if len(os.Args) < 2 {
		app.ShowHelp()
		return
	}

	var err error

	switch os.Args[1] {
	case "config":
		// If no arguments after 'config', show help
		if len(os.Args) == 2 {
			app.ShowConfigHelp()
			return
		}
		err = configCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing config arguments: %v", err))
			os.Exit(1)
		}
		err = app.HandleConfig(*apiKey, *model)
	case "view":
		err = viewCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing view arguments: %v", err))
			os.Exit(1)
		}
		err = app.HandleView()
	case "models":
		err = modelsCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing models arguments: %v", err))
			os.Exit(1)
		}
		err = app.HandleModels()
	case "commit":
		err = commitCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing commit arguments: %v", err))
			os.Exit(1)
		}
		err = app.HandleCommit(*commitType, *commitContext, *commitCount)
	case "help":
		err = helpCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing help arguments: %v", err))
			os.Exit(1)
		}
		app.HandleHelp()
		return // Help doesn't return an error
	default:
		app.printer.PrintError(fmt.Sprintf("Unknown command '%s'. Use 'help' to see available commands.", os.Args[1]))
		os.Exit(1)
	}

	if err != nil {
		app.printer.PrintError(err.Error())
		os.Exit(1)
	}
}
