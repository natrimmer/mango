package git

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Client defines the interface for git operations
type Client interface {
	GetStagedDiff() (string, error)
	GetStagedFiles() (string, error)
}

// RealClient implements Client using git commands
type RealClient struct{}

func NewRealClient() *RealClient {
	return &RealClient{}
}

func (gc *RealClient) GetStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--staged")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running git diff: %w", err)
	}
	return out.String(), nil
}

func (gc *RealClient) GetStagedFiles() (string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error getting changed files: %w", err)
	}
	return out.String(), nil
}
