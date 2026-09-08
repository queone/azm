package utl

import (
	"fmt"
	"os"
	"strings"
)

// Color helpers follow the terminal-gating policy shared by the sibling
// repositories: escapes are emitted only when stdout is a terminal, NO_COLOR is
// unset, TERM is not dumb, and the terminal advertises 256-color support. Each
// helper wraps its text in a 256-color SGR sequence when enabled and returns
// the plain text otherwise.

var enabled = colorEnabled(os.Getenv, stdoutIsTTY())

func stdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// colorEnabled applies the gating policy to the given environment and TTY state.
func colorEnabled(getenv func(string) string, tty bool) bool {
	if !tty || getenv("NO_COLOR") != "" || getenv("TERM") == "dumb" {
		return false
	}
	colorterm := getenv("COLORTERM")
	return colorterm == "truecolor" || colorterm == "24bit" || strings.Contains(getenv("TERM"), "256color")
}

// wrap renders value inside the given SGR code when color is enabled.
func wrap(code string, value any) string {
	text := fmt.Sprint(value)
	if !enabled {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// The exported names match the helpers azm has always called; each maps to
// the nearest step of the sibling palette's 256-color ramps.

// Red renders value in bright red.
func Red(value any) string { return wrap("38;5;196", value) }

// Red2 renders value in dark red.
func Red2(value any) string { return wrap("38;5;160", value) }

// Blu renders value in light blue.
func Blu(value any) string { return wrap("38;5;75", value) }

// Gre renders value in green.
func Gre(value any) string { return wrap("38;5;46", value) }

// Yel renders value in yellow.
func Yel(value any) string { return wrap("38;5;220", value) }

// Whi renders value in white.
func Whi(value any) string { return wrap("38;5;231", value) }

// Whi2 renders value in bright white.
func Whi2(value any) string { return wrap("38;5;255", value) }

// Cya renders value in cyan.
func Cya(value any) string { return wrap("38;5;51", value) }

// Cya2 renders value in light cyan.
func Cya2(value any) string { return wrap("38;5;123", value) }

// Mag renders value in magenta.
func Mag(value any) string { return wrap("38;5;201", value) }

// Gra renders value in gray.
func Gra(value any) string { return wrap("38;5;245", value) }
