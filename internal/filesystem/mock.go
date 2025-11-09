package filesystem

import "os"

// MockFileSystem implements FileSystem interface for testing
type MockFileSystem struct {
	homeDir    string
	homeErr    error
	mkdirErr   error
	writeErr   error
	readData   []byte
	readErr    error
	writeFiles map[string][]byte // Track what was written
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		writeFiles: make(map[string][]byte),
	}
}

func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.homeDir, m.homeErr
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return m.mkdirErr
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writeFiles[filename] = data
	return nil
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	return m.readData, m.readErr
}

// SetHomeDir is a helper for tests
func (m *MockFileSystem) SetHomeDir(dir string) {
	m.homeDir = dir
}

// SetReadData is a helper for tests
func (m *MockFileSystem) SetReadData(data []byte) {
	m.readData = data
}

// SetReadError is a helper for tests
func (m *MockFileSystem) SetReadError(err error) {
	m.readErr = err
}

// SetHomeError is a helper for tests
func (m *MockFileSystem) SetHomeError(err error) {
	m.homeErr = err
}

// SetMkdirError is a helper for tests
func (m *MockFileSystem) SetMkdirError(err error) {
	m.mkdirErr = err
}

// SetWriteError is a helper for tests
func (m *MockFileSystem) SetWriteError(err error) {
	m.writeErr = err
}

// GetWrittenFile is a helper for tests
func (m *MockFileSystem) GetWrittenFile(path string) []byte {
	return m.writeFiles[path]
}
