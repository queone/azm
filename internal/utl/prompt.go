// Package utl carries the subset of the former external queone utl helpers
// that azm uses, copied into this module so azm has no external dependency on
// them. Color helpers follow the terminal-gating policy shared by the sibling
// repositories, and the colorized YAML/JSON printer mirrors gkit's jy utility.
package utl

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
)

// Same as regular Printf function but always exits with a return code of 1
func Die(format string, args ...any) {
	if len(args) == 0 {
		fmt.Print(format) // No arguments, use Print
	} else {
		fmt.Printf(format, args...) // Use Printf with arguments
	}
	os.Exit(1)
}

// Returns a string showing the current filepath line number and function name.
func Trace() string {
	progCounter, fp, ln, ok := runtime.Caller(1)
	if !ok {
		return fmt.Sprintf("%s\n    %s:%d\n", "?", "?", 0)
	}
	funcPointer := runtime.FuncForPC(progCounter)
	if funcPointer == nil {
		return fmt.Sprintf("%s\n    %s:%d\n", "?", fp, ln)
	}
	return fmt.Sprintf("%s\n    %s:%d\n", funcPointer.Name(), fp, ln)
}

// Trace type
type TraceInfo struct {
	FuncName string
	File     string
	Line     int
}

// Improved Trace v2 with depth and with more flexible TraceInfo type return

// When depth == 1: The immediate function/file/line number
// When depth == 2: The function/file/line that called the immediate function
// When depth == 3: The function before the last, and so on

// Check if two variables are of the same type

// Checks if given rune is an alphabetic character (either uppercase or lowercase).
// Returns true if it is, false otherwise.

// Returns true if rune is a numerical digit. False otherwise.

// Returns true if rune is a hexadeximal digit. False otherwise.

// Return the object's keys sorted

// TODO: Combine above SortMapStringKeys and SortObjStringKeys using interfaces
// SortStringKeys()?

// Print prompt message and return single rune character input
func PromptMsg(msg string) rune {
	fmt.Print(Yel(msg))
	reader := bufio.NewReader(os.Stdin)
	confirm, _, err := reader.ReadRune()
	if err != nil {
		fmt.Println(err)
	}
	return confirm
}
