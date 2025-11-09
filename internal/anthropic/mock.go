package anthropic

import (
	"io"
	"net/http"
	"strings"
)

// MockHTTPClient implements HTTPClient interface for testing
type MockHTTPClient struct {
	response *http.Response
	err      error
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{}
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// SetResponse is a helper for tests
func (m *MockHTTPClient) SetResponse(resp *http.Response) {
	m.response = resp
}

// SetError is a helper for tests
func (m *MockHTTPClient) SetError(err error) {
	m.err = err
}

// CreateHTTPResponse is a helper function to create HTTP response for testing
func CreateHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
