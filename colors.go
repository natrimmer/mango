package main

import "fmt"

// ANSI color codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

func printSuccess(msg string) { fmt.Println(Green + "✓ " + msg + Reset) }
func printError(msg string)   { fmt.Println(Red + msg + Reset) }
func printWarning(msg string) { fmt.Println(Yellow + msg + Reset) }
