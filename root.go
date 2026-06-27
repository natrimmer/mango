package main

import (
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// tagline is the product one-liner, kept provider-neutral so it survives
// adding API providers beyond Claude.
const tagline = "Generate conventional commit messages with AI"

// Version information - set at build time with ldflags.
var (
	version   = "v0.0.0-dev"
	buildDate = "unknown"
	commitSHA = "unknown"
)

// When installed via `go install ...@version`, ldflags aren't applied, so fall
// back to the module version the Go toolchain embeds.
func init() {
	if version != "v0.0.0-dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}

var rootCmd = &cobra.Command{
	Use:           "mango",
	Short:         tagline,
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
	s := Bold + Magenta + "mango" + Reset + " " + Dim + version + Reset + "\n"
	if version != "v0.0.0-dev" {
		s += Dim + "Build Date: " + buildDate + Reset + "\n"
		s += Dim + "Commit: " + commitSHA + Reset + "\n"
	}
	return s + Dim + tagline + Reset + "\n"
}
