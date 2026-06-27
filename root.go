package main

import (
	"os"

	"github.com/spf13/cobra"
)

// Version information - set at build time with ldflags.
var (
	version   = "v0.0.0-dev"
	buildDate = "unknown"
	commitSHA = "unknown"
)

var rootCmd = &cobra.Command{
	Use:           "mango",
	Short:         "Generate conventional commit messages with Anthropic's Claude",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	rootCmd.SetVersionTemplate(versionString())
	if err := rootCmd.Execute(); err != nil {
		printError(err.Error())
		os.Exit(1)
	}
}

func versionString() string {
	s := Bold + Magenta + "Mango" + Reset + " " + Dim + version + Reset + "\n"
	if version != "v0.0.0-dev" {
		s += Dim + "Build Date: " + buildDate + Reset + "\n"
		s += Dim + "Commit: " + commitSHA + Reset + "\n"
	}
	return s + Dim + "Generate conventional commit messages with Anthropic's Claude" + Reset + "\n"
}
