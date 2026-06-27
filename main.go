package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/natrimmer/claude_commit/internal/anthropic"
	"github.com/natrimmer/claude_commit/internal/app"
	"github.com/natrimmer/claude_commit/internal/commit"
	"github.com/natrimmer/claude_commit/internal/config"
	"github.com/natrimmer/claude_commit/internal/filesystem"
	"github.com/natrimmer/claude_commit/internal/git"
	"github.com/natrimmer/claude_commit/internal/model"
	"github.com/natrimmer/claude_commit/internal/ui"
)

// Version information - set at build time with ldflags
var (
	version   = "v0.0.0-dev" // Default SemVer version for development builds
	buildDate = "unknown"    // Build date
	commitSHA = "unknown"    // Git commit SHA
)

func main() {
	// Set version information in app package
	app.Version = version
	app.BuildDate = buildDate
	app.CommitSHA = commitSHA

	// Initialize dependencies
	fs := filesystem.NewRealFileSystem()
	httpClient := &http.Client{}
	gitClient := git.NewRealClient()
	printer := ui.NewConsolePrinter()

	// Initialize services
	configService := config.NewService(fs, printer)
	anthropicClient := anthropic.NewClient(httpClient, printer)
	modelService := model.NewService(configService, printer)
	commitService := commit.NewService(configService, anthropicClient, gitClient, printer)

	// Create app
	application := app.New(configService, modelService, commitService, printer)

	// Handle global flags first
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-v":
			application.ShowVersion()
			return
		case "--help", "-h":
			application.ShowHelp()
			return
		}
	}

	// Define command flag sets
	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	apiKey := configCmd.String("api-key", "", "Anthropic API key")
	modelFlag := configCmd.String("model", "", "Anthropic model to use")

	commitCmd := flag.NewFlagSet("commit", flag.ExitOnError)
	commitType := commitCmd.String("type", "", "Commit type (feat, fix, docs, etc.)")
	commitContext := commitCmd.String("context", "", "Additional context to guide commit message generation")
	commitCount := commitCmd.Int("count", 1, "Number of commit message options to generate")
	commitDryRun := commitCmd.Bool("dry-run", false, "Show prompt without calling API")
	commitVerbose := commitCmd.Bool("verbose", false, "Show prompt and full API interaction")
	commitVerboseShort := commitCmd.Bool("v", false, "Show prompt and full API interaction (short form)")

	viewCmd := flag.NewFlagSet("view", flag.ExitOnError)
	modelsCmd := flag.NewFlagSet("models", flag.ExitOnError)
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)

	// If no arguments provided, show help
	if len(os.Args) < 2 {
		application.ShowHelp()
		return
	}

	var err error

	// Parse command and execute
	switch os.Args[1] {
	case "config":
		// If no arguments after 'config', show help
		if len(os.Args) == 2 {
			application.ShowConfigHelp()
			return
		}
		err = configCmd.Parse(os.Args[2:])
		if err != nil {
			printer.PrintError(fmt.Sprintf("Error parsing config arguments: %v", err))
			os.Exit(1)
		}
		err = application.HandleConfig(*apiKey, *modelFlag)
	case "view":
		err = viewCmd.Parse(os.Args[2:])
		if err != nil {
			printer.PrintError(fmt.Sprintf("Error parsing view arguments: %v", err))
			os.Exit(1)
		}
		err = application.HandleView()
	case "models":
		err = modelsCmd.Parse(os.Args[2:])
		if err != nil {
			printer.PrintError(fmt.Sprintf("Error parsing models arguments: %v", err))
			os.Exit(1)
		}
		err = application.HandleModels()
	case "commit":
		err = commitCmd.Parse(os.Args[2:])
		if err != nil {
			printer.PrintError(fmt.Sprintf("Error parsing commit arguments: %v", err))
			os.Exit(1)
		}
		// Combine verbose flags (--verbose or -v)
		verbose := *commitVerbose || *commitVerboseShort
		err = application.HandleCommit(*commitType, *commitContext, *commitCount, *commitDryRun, verbose)
	case "help":
		err = helpCmd.Parse(os.Args[2:])
		if err != nil {
			printer.PrintError(fmt.Sprintf("Error parsing help arguments: %v", err))
			os.Exit(1)
		}
		application.HandleHelp()
		return // Help doesn't return an error
	default:
		printer.PrintError(fmt.Sprintf("Unknown command '%s'. Use 'help' to see available commands.", os.Args[1]))
		os.Exit(1)
	}

	if err != nil {
		printer.PrintError(err.Error())
		os.Exit(1)
	}
}
