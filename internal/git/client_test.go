package git

import (
	"errors"
	"testing"
)

func TestRealClient(t *testing.T) {
	// Basic smoke test to ensure RealClient implements the interface
	var _ Client = &RealClient{}

	client := NewRealClient()
	if client == nil {
		t.Error("NewRealClient should not return nil")
	}
}

func TestMockClient(t *testing.T) {
	mock := NewMockClient()
	mock.SetStagedDiff("diff content")
	mock.SetStagedFiles("file1.go\nfile2.go")

	// Test GetStagedDiff
	diff, err := mock.GetStagedDiff()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if diff != "diff content" {
		t.Errorf("Expected 'diff content', got %s", diff)
	}

	// Test GetStagedFiles
	files, err := mock.GetStagedFiles()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if files != "file1.go\nfile2.go" {
		t.Errorf("Expected 'file1.go\\nfile2.go', got %s", files)
	}

	// Test error cases
	mock.SetDiffError(errors.New("diff error"))
	_, err = mock.GetStagedDiff()
	if err == nil {
		t.Error("Expected error from GetStagedDiff")
	}

	mock.SetFilesError(errors.New("files error"))
	_, err = mock.GetStagedFiles()
	if err == nil {
		t.Error("Expected error from GetStagedFiles")
	}
}
