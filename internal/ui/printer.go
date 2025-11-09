package ui

import "fmt"

// Printer defines the interface for outputting messages
type Printer interface {
	Print(msg string)
	PrintSuccess(msg string)
	PrintError(msg string)
	PrintWarning(msg string)
}

// ConsolePrinter implements Printer for console output
type ConsolePrinter struct{}

func NewConsolePrinter() *ConsolePrinter {
	return &ConsolePrinter{}
}

func (p *ConsolePrinter) Print(msg string) {
	fmt.Println(msg)
}

func (p *ConsolePrinter) PrintSuccess(msg string) {
	fmt.Println(Green + msg + Reset)
}

func (p *ConsolePrinter) PrintError(msg string) {
	fmt.Println(Red + msg + Reset)
}

func (p *ConsolePrinter) PrintWarning(msg string) {
	fmt.Println(Yellow + msg + Reset)
}
