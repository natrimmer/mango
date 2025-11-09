package git

// MockClient implements Client interface for testing
type MockClient struct {
	stagedDiff  string
	stagedFiles string
	diffErr     error
	filesErr    error
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) GetStagedDiff() (string, error) {
	return m.stagedDiff, m.diffErr
}

func (m *MockClient) GetStagedFiles() (string, error) {
	return m.stagedFiles, m.filesErr
}

// SetStagedDiff is a helper for tests
func (m *MockClient) SetStagedDiff(diff string) {
	m.stagedDiff = diff
}

// SetStagedFiles is a helper for tests
func (m *MockClient) SetStagedFiles(files string) {
	m.stagedFiles = files
}

// SetDiffError is a helper for tests
func (m *MockClient) SetDiffError(err error) {
	m.diffErr = err
}

// SetFilesError is a helper for tests
func (m *MockClient) SetFilesError(err error) {
	m.filesErr = err
}
