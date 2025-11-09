package ui

import "testing"

func TestConsolePrinter(t *testing.T) {
	// Basic smoke test to ensure ConsolePrinter implements the interface
	var _ Printer = &ConsolePrinter{}

	printer := NewConsolePrinter()
	if printer == nil {
		t.Error("NewConsolePrinter should not return nil")
	}
}
