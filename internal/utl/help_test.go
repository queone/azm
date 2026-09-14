package utl

import (
	"fmt"
	"strings"
	"testing"
)

func sampleHelp() Help {
	return Help{
		Name:        "tool",
		Version:     "1.0.0",
		Description: "Does things",
		URL:         "example.com/tool",
		Sections: []Section{
			{Heading: "Usage", Rows: []Row{{Form: "tool OPTION"}}},
			{Heading: "Options", Rows: []Row{
				{Form: "-a", Meaning: []string{"Alpha", "continued"}},
				{Form: "-bbb X", Meaning: []string{"Beta"}},
			}, Note: []string{"Trailing note."}},
			{Heading: "Examples", Rows: []Row{{Form: "tool -a", Meaning: []string{"Run alpha"}}}},
		},
	}
}

// headingsOf returns the unindented non-blank lines.
func headingsOf(lines []string) []string {
	var out []string
	for _, l := range lines {
		if l != "" && !strings.HasPrefix(l, " ") {
			out = append(out, l)
		}
	}
	return out
}

func TestRenderHelpHeaderAndHeadings(t *testing.T) {
	lines := strings.Split(RenderHelp(sampleHelp(), false), "\n")
	want := []string{"tool v1.0.0", "Does things", "example.com/tool", "", "Usage"}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d = %q, want %q", i+1, lines[i], w)
		}
	}
	if got := headingsOf(lines[4:]); strings.Join(got, ",") != "Usage,Options,Examples" {
		t.Fatalf("headings = %v", got)
	}
}

func TestRenderHelpAlignsMeaningsTwoPastLongestForm(t *testing.T) {
	out := RenderHelp(sampleHelp(), false)
	// The longest Options form is the appended "-h, -?, --help" at 14 runes.
	for _, want := range []string{
		fmt.Sprintf("  %-14s  Alpha\n", "-a"),
		strings.Repeat(" ", 18) + "continued\n",
		fmt.Sprintf("  %-14s  Beta\n", "-bbb X"),
		fmt.Sprintf("  %-14s  Print the version\n", "-v, --version"),
		"  -h, -?, --help  Print this help\n",
		"  tool -a  Run alpha\n",
		"  tool OPTION\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestRenderHelpAppendsStandardRowsToOptionsBeforeNote(t *testing.T) {
	out := RenderHelp(sampleHelp(), false)
	if strings.Count(out, "-v, --version") != 1 || strings.Count(out, "-h, -?, --help") != 1 {
		t.Fatalf("standard rows must appear exactly once:\n%s", out)
	}
	if !strings.Contains(out, "  -h, -?, --help  Print this help\n\n  Trailing note.\n\nExamples\n") {
		t.Fatalf("note must follow the appended rows after one blank line:\n%s", out)
	}
}

func TestRenderHelpColorSpans(t *testing.T) {
	if plain := RenderHelp(sampleHelp(), false); strings.Contains(plain, "\x1b") {
		t.Fatal("plain output must carry no escape sequence")
	}
	colored := strings.Split(RenderHelp(sampleHelp(), true), "\n")
	want := map[int]string{
		0: sgrHeading + "tool" + sgrReset + " v1.0.0",
		1: sgrDescription + "Does things" + sgrReset,
		2: sgrURL + "example.com/tool" + sgrReset,
		4: sgrHeading + "Usage" + sgrReset,
	}
	for i, w := range want {
		if colored[i] != w {
			t.Errorf("colored line %d = %q, want %q", i+1, colored[i], w)
		}
	}
	if n := strings.Count(strings.Join(colored, "\n"), "\x1b["); n != 12 {
		t.Errorf("escape count = %d, want 12 for three header spans and three headings", n)
	}
}

func TestHelpSelectKeepsOnlyNamedSectionsInOrder(t *testing.T) {
	h := sampleHelp()
	short := h.Select("Examples", "Usage")
	if len(short.Sections) != 2 || short.Sections[0].Heading != "Usage" || short.Sections[1].Heading != "Examples" {
		t.Fatalf("Select kept %v", short.Sections)
	}
	if len(h.Sections) != 3 {
		t.Fatal("Select must not change the original")
	}
}
