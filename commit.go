package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const divider = "─────────────────────────────────────────"

// conventionalTypes are the valid commit types accepted by --type.
var conventionalTypes = []string{
	"feat", "fix", "docs", "style", "refactor",
	"perf", "test", "chore", "ci", "build", "revert",
}

// apiURL is a var so tests can point it at a stub server.
var apiURL = "https://api.anthropic.com/v1/messages"

func stagedDiff() (string, error) {
	out, err := exec.Command("git", "diff", "--staged").Output()
	if err != nil {
		return "", fmt.Errorf("error running git diff: %w", err)
	}
	return string(out), nil
}

func stagedFiles() (string, error) {
	out, err := exec.Command("git", "diff", "--staged", "--name-only").Output()
	if err != nil {
		return "", fmt.Errorf("error getting changed files: %w", err)
	}
	return string(out), nil
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func generateCommitMessage(cfg Config, prompt string) (string, error) {
	reqBody, err := json.Marshal(map[string]any{
		"model":      cfg.Model,
		"max_tokens": 1024, // ceiling billed on actual output; generous so --count doesn't truncate
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.ApiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making API call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	var out anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("error parsing API response: %w", err)
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	return out.Content[0].Text, nil
}

func buildPrompt(files, diff, commitType, context string, count int) string {
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

func runCommit(commitType, context string, count int, dryRun, verbose bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	diff, err := stagedDiff()
	if err != nil {
		return err
	}
	files, err := stagedFiles()
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		return fmt.Errorf("no staged changes found. Use git add to stage changes")
	}

	prompt := buildPrompt(files, diff, commitType, context, count)

	if verbose || dryRun {
		fmt.Println(Bold + Cyan + "Prompt being sent to Claude:" + Reset)
		fmt.Println(Dim + divider + Reset)
		fmt.Println(prompt)
		fmt.Println(Dim + divider + Reset)
		fmt.Println()
	}
	if dryRun {
		printWarning("Dry run mode - API not called")
		return nil
	}

	fmt.Println(Dim + "Analyzing git diff with Claude..." + Reset)

	msg, err := generateCommitMessage(*cfg, prompt)
	if err != nil {
		return err
	}
	msg = strings.TrimSpace(msg)

	if verbose {
		fmt.Println(Bold + Cyan + "Raw API Response:" + Reset)
		fmt.Println(Dim + divider + Reset)
		fmt.Println(msg)
		fmt.Println(Dim + divider + Reset)
		fmt.Println()
	}

	var messages []string
	if count > 1 {
		for line := range strings.SplitSeq(msg, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				messages = append(messages, line)
			}
		}
	} else {
		messages = []string{msg}
	}

	// Piped/CI: no TTY to prompt on, so print runnable commands (old behavior).
	if !isInteractive() {
		printCommands(messages)
		return nil
	}

	chosen := selectMessage(messages)
	if chosen == "" {
		printWarning("Aborted — nothing committed")
		return nil
	}
	return runGitCommit(chosen)
}

func printCommands(messages []string) {
	if len(messages) > 1 {
		printSuccess("Commit message options generated")
	} else {
		printSuccess("Commit message generated")
	}
	fmt.Println()
	for i, m := range messages {
		if len(messages) > 1 {
			fmt.Printf("%s%d.%s %s\n", Bold, i+1, Reset, fmt.Sprintf("git commit -m %q", m))
		} else {
			fmt.Println(Bold + fmt.Sprintf("git commit -m %q", m) + Reset)
		}
	}
}

// selectMessage shows the message(s) and returns the one to commit, or "" to abort.
func selectMessage(messages []string) string {
	fmt.Println()
	if len(messages) == 1 {
		fmt.Println(Bold + messages[0] + Reset)
		fmt.Println()
		if yes(promptLine("Commit with this message? [Y/n] ")) {
			return messages[0]
		}
		return ""
	}
	for i, m := range messages {
		fmt.Printf("%s%d.%s %s\n", Bold, i+1, Reset, m)
	}
	fmt.Println()
	if n := parseSelection(promptLine(fmt.Sprintf("Select [1-%d] to commit, or Enter to skip: ", len(messages))), len(messages)); n > 0 {
		return messages[n-1]
	}
	return ""
}

func runGitCommit(msg string) error {
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	return nil
}

// isInteractive reports whether stdin is a terminal we can prompt on.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func promptLine(prompt string) string {
	fmt.Print(Bold + prompt + Reset)
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		return sc.Text()
	}
	return ""
}

func yes(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "" || s == "y" || s == "yes"
}

// parseSelection returns a 1-based choice in [1,max], or -1 if invalid.
func parseSelection(in string, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(in))
	if err != nil || n < 1 || n > max {
		return -1
	}
	return n
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message from staged changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		commitType, _ := cmd.Flags().GetString("type")
		context, _ := cmd.Flags().GetString("context")
		count, _ := cmd.Flags().GetInt("count")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		verbose, _ := cmd.Flags().GetBool("verbose")
		if commitType != "" && !slices.Contains(conventionalTypes, commitType) {
			return fmt.Errorf("invalid type %q. Valid types: %s", commitType, strings.Join(conventionalTypes, ", "))
		}
		return runCommit(commitType, context, count, dryRun, verbose)
	},
}

func init() {
	commitCmd.Flags().StringP("type", "t", "", "Commit type (feat, fix, docs, etc.)")
	commitCmd.Flags().StringP("context", "c", "", "Additional context to guide generation")
	commitCmd.Flags().IntP("count", "n", 1, "Number of commit message options to generate")
	commitCmd.Flags().Bool("dry-run", false, "Show prompt without calling the API")
	commitCmd.Flags().BoolP("verbose", "v", false, "Show prompt and full API interaction")
	rootCmd.AddCommand(commitCmd)
}
