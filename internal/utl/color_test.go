package utl

import (
	"strings"
	"testing"
)

func envOf(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestColorEnabledRequiresTerminalAndColorSupport(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		tty  bool
		want bool
	}{
		{"truecolor terminal", map[string]string{"COLORTERM": "truecolor"}, true, true},
		{"256color TERM", map[string]string{"TERM": "xterm-256color"}, true, true},
		{"plain TERM", map[string]string{"TERM": "xterm"}, true, false},
		{"not a tty", map[string]string{"COLORTERM": "truecolor"}, false, false},
		{"NO_COLOR set", map[string]string{"COLORTERM": "truecolor", "NO_COLOR": "1"}, true, false},
		{"dumb TERM", map[string]string{"COLORTERM": "truecolor", "TERM": "dumb"}, true, false},
	}
	for _, c := range cases {
		if got := colorEnabled(envOf(c.env), c.tty); got != c.want {
			t.Errorf("%s: colorEnabled() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestColorHelpersReturnPlainTextWhenDisabled(t *testing.T) {
	prev := enabled
	enabled = false
	defer func() { enabled = prev }()
	for name, fn := range map[string]func(any) string{"Red": Red, "Gre": Gre, "Blu": Blu, "Yel": Yel, "Whi": Whi, "Whi2": Whi2, "Cya": Cya, "Cya2": Cya2, "Mag": Mag, "Gra": Gra, "Red2": Red2} {
		if got := fn("text"); got != "text" {
			t.Errorf("%s(\"text\") = %q, want plain text", name, got)
		}
	}
}

func TestColorHelpersWrapWithSgrWhenEnabled(t *testing.T) {
	prev := enabled
	enabled = true
	defer func() { enabled = prev }()
	got := Red("text")
	if !strings.HasPrefix(got, "\x1b[38;5;196m") || !strings.HasSuffix(got, "text\x1b[0m") {
		t.Fatalf("Red(\"text\") = %q, want 256-color SGR wrapping", got)
	}
	if got := Gre(42); !strings.Contains(got, "42") {
		t.Fatalf("Gre(42) = %q, want the number rendered", got)
	}
}
