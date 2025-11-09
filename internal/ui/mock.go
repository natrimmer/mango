package ui

import "strings"

// MockPrinter implements Printer interface for testing
type MockPrinter struct {
	messages []string
}

func NewMockPrinter() *MockPrinter {
	return &MockPrinter{}
}

func (m *MockPrinter) Print(msg string) {
	m.messages = append(m.messages, msg)
}

func (m *MockPrinter) PrintSuccess(msg string) {
	m.messages = append(m.messages, "[SUCCESS] "+msg)
}

func (m *MockPrinter) PrintError(msg string) {
	m.messages = append(m.messages, "[ERROR] "+msg)
}

func (m *MockPrinter) PrintWarning(msg string) {
	m.messages = append(m.messages, "[WARNING] "+msg)
}

func (m *MockPrinter) GetMessages() []string {
	return m.messages
}

func (m *MockPrinter) Reset() {
	m.messages = nil
}

func (m *MockPrinter) ContainsMessage(msg string) bool {
	for _, message := range m.messages {
		if strings.Contains(message, msg) {
			return true
		}
	}
	return false
}
