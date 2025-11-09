package filesystem

import (
	"errors"
	"testing"
)

func TestRealFileSystem(t *testing.T) {
	// Basic smoke test to ensure RealFileSystem implements the interface
	var _ FileSystem = &RealFileSystem{}

	fs := NewRealFileSystem()
	if fs == nil {
		t.Error("NewRealFileSystem should not return nil")
	}
}

func TestMockFileSystem(t *testing.T) {
	mock := NewMockFileSystem()
	mock.SetHomeDir("/test/home")
	mock.SetReadData([]byte("test data"))

	// Test UserHomeDir
	home, err := mock.UserHomeDir()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if home != "/test/home" {
		t.Errorf("Expected /test/home, got %s", home)
	}

	// Test ReadFile
	data, err := mock.ReadFile("test.txt")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if string(data) != "test data" {
		t.Errorf("Expected 'test data', got %s", string(data))
	}

	// Test WriteFile
	err = mock.WriteFile("output.txt", []byte("written"), 0644)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if string(mock.GetWrittenFile("output.txt")) != "written" {
		t.Error("WriteFile did not store data correctly")
	}

	// Test error cases
	mock.SetHomeError(errors.New("home error"))
	_, err = mock.UserHomeDir()
	if err == nil {
		t.Error("Expected error from UserHomeDir")
	}

	mock.SetWriteError(errors.New("write error"))
	err = mock.WriteFile("fail.txt", []byte("data"), 0644)
	if err == nil {
		t.Error("Expected error from WriteFile")
	}
}
