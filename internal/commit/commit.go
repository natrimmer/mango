package commit

import (
	"fmt"
	"strings"

	"github.com/natrimmer/claude_commit/internal/anthropic"
	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/git"
	"github.com/natrimmer/claude_commit/internal/ui"
)

// Service handles commit message generation
type Service struct {
	configService   *config.Service
	anthropicClient *anthropic.Client
	gitClient       git.Client
	printer         ui.Printer
}

// NewService creates a new commit service
func NewService(configService *config.Service, anthropicClient *anthropic.Client, gitClient git.Client, printer ui.Printer) *Service {
	return &Service{
		configService:   configService,
		anthropicClient: anthropicClient,
		gitClient:       gitClient,
		printer:         printer,
	}
}

// GenerateCommitMessage generates a commit message based on staged changes
func (cs *Service) GenerateCommitMessage(commitType, context string, count int, dryRun, verbose bool) error {
	cfg, err := cs.configService.LoadConfig()
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

	prompt := cs.buildPrompt(files, diff, commitType, context, count)

	// If verbose or dry-run, show the prompt
	if verbose || dryRun {
		cs.printer.Print(ui.Bold + ui.Cyan + "Prompt being sent to Claude:" + ui.Reset)
		cs.printer.Print(ui.Dim + "─────────────────────────────────────────" + ui.Reset)
		cs.printer.Print(prompt)
		cs.printer.Print(ui.Dim + "─────────────────────────────────────────" + ui.Reset)
		cs.printer.Print("")
	}

	// If dry-run, stop here without calling the API
	if dryRun {
		cs.printer.PrintWarning("⚠️  Dry run mode - API not called")
		return nil
	}

	if !dryRun {
		cs.printer.Print(ui.Dim + "⚙️  Analyzing git diff with Claude AI..." + ui.Reset)
	}

	commitMsg, err := cs.anthropicClient.GenerateCommitMessage(*cfg, prompt)
	if err != nil {
		return err
	}

	commitMsg = strings.TrimSpace(commitMsg)

	// If verbose, show the raw response
	if verbose {
		cs.printer.Print(ui.Bold + ui.Cyan + "Raw API Response:" + ui.Reset)
		cs.printer.Print(ui.Dim + "─────────────────────────────────────────" + ui.Reset)
		cs.printer.Print(commitMsg)
		cs.printer.Print(ui.Dim + "─────────────────────────────────────────" + ui.Reset)
		cs.printer.Print("")
	}

	if count > 1 {
		// Multiple messages - display them numbered
		cs.printer.PrintSuccess("✓ Commit message options generated")
		cs.printer.Print("")
		messages := strings.Split(commitMsg, "\n")
		for i, msg := range messages {
			msg = strings.TrimSpace(msg)
			if msg != "" {
				cs.printer.Print(fmt.Sprintf("%s%d.%s %s", ui.Bold, i+1, ui.Reset, msg))
			}
		}
	} else {
		// Single message - display git command
		gitCommand := fmt.Sprintf("git commit -m \"%s\"", commitMsg)
		cs.printer.PrintSuccess("✓ Commit message generated")
		cs.printer.Print("")
		cs.printer.Print(ui.Bold + gitCommand + ui.Reset)
	}

	return nil
}

func (cs *Service) buildPrompt(files, diff, commitType, context string, count int) string {
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
