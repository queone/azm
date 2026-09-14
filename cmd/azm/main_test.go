package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/queone/azm/internal/utl"
)

// strictStableSemver mirrors the canonical build contract for programVersion:
// MAJOR.MINOR.PATCH with no leading zeroes, prerelease, or build metadata.
var strictStableSemver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func TestProgramVersionIsStrictStableSemver(t *testing.T) {
	if !strictStableSemver.MatchString(programVersion) {
		t.Fatalf("programVersion %q must match MAJOR.MINOR.PATCH", programVersion)
	}
}

func TestVersionLineMatchesDeclaredVersion(t *testing.T) {
	want := programName + " v" + programVersion
	if got := versionLine(); got != want {
		t.Fatalf("versionLine() = %q, want %q", got, want)
	}
}

// plainHelp renders the full help page without color, as the build gate reads it.
func plainHelp() string { return utl.RenderHelp(helpPage(false), false) }

// helpLines splits a page into lines without the trailing empty element.
func helpLines(page string) []string {
	return strings.Split(strings.TrimSuffix(page, "\n"), "\n")
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

// sectionLines returns the body lines of the named section without trailing blanks.
func sectionLines(page, heading string) []string {
	var body []string
	in := false
	for _, l := range helpLines(page)[4:] {
		if l != "" && !strings.HasPrefix(l, " ") {
			if in {
				break
			}
			in = l == heading
			continue
		}
		if in {
			body = append(body, l)
		}
	}
	for len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
	}
	return body
}

func TestHelpHeaderLines(t *testing.T) {
	lines := helpLines(plainHelp())
	want := []string{versionLine(), programDescription, programURL, "", "Usage"}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d = %q, want %q", i+1, lines[i], w)
		}
	}
	if strings.HasSuffix(lines[1], ".") {
		t.Fatal("line 2 must not end with a period")
	}
	if strings.Contains(lines[2], "://") || strings.Contains(lines[2], " ") {
		t.Fatal("line 3 must be the bare URL with no scheme")
	}
}

func TestHelpSectionsAreOrderedAndBodiesIndented(t *testing.T) {
	heading := regexp.MustCompile(`^[A-Z][a-z]+$`)
	var got []string
	for _, l := range helpLines(plainHelp())[4:] {
		switch {
		case l == "" || strings.HasPrefix(l, "  "):
		case heading.MatchString(l):
			got = append(got, l)
		default:
			t.Fatalf("line outside a section or malformed heading: %q", l)
		}
	}
	if strings.Join(got, ",") != "Usage,Options,Types,Examples" {
		t.Fatalf("headings = %v", got)
	}
}

func TestHelpEscapeSequences(t *testing.T) {
	pages := map[string]string{"full": plainHelp(), "short": utl.RenderHelp(helpPage(true), false)}
	for name, page := range pages {
		if strings.Contains(page, "\x1b") {
			t.Fatalf("%s plain page carries an escape sequence", name)
		}
	}
	colored := utl.RenderHelp(helpPage(false), true)
	counts := map[string]int{"\x1b[1;38;5;231m": 5, "\x1b[38;5;245m": 1, "\x1b[38;5;242m": 1, "\x1b[0m": 7}
	for seq, want := range counts {
		if got := strings.Count(colored, seq); got != want {
			t.Errorf("%q count = %d, want %d", seq, got, want)
		}
	}
	if got := strings.Count(colored, "\x1b"); got != 14 {
		t.Errorf("escape count = %d, want 14", got)
	}
}

func TestOptionsEndWithVersionAndHelpRowsThenOneParagraph(t *testing.T) {
	body := sectionLines(plainHelp(), "Options")
	rows := 0
	for rows < len(body) && body[rows] != "" {
		rows++
	}
	if rows < 2 || !strings.HasPrefix(body[rows-2], "  -v, --version ") || !strings.HasPrefix(body[rows-1], "  -h, -?, --help ") {
		t.Fatalf("Options rows must end with the version and help rows, got %q", body[:rows])
	}
	note := body[rows:]
	if len(note) < 2 || note[0] != "" {
		t.Fatalf("Options must end with one trailing paragraph, got %q", note)
	}
	for _, l := range note[1:] {
		if !strings.HasPrefix(l, "  ") {
			t.Fatalf("trailing paragraph line %q must be indented", l)
		}
	}
}

func TestRowMeaningsAlignTwoPastLongestForm(t *testing.T) {
	page := plainHelp()
	for _, s := range helpPage(false).Sections {
		rows := append([]utl.Row{}, s.Rows...)
		if s.Heading == "Options" {
			rows = append(rows, utl.Row{Form: "-v, --version"}, utl.Row{Form: "-h, -?, --help"})
		}
		width := 0
		for _, r := range rows {
			if len(r.Form) > width {
				width = len(r.Form)
			}
		}
		for _, r := range s.Rows {
			if len(r.Meaning) == 0 {
				continue
			}
			if want := fmt.Sprintf("  %-*s  %s\n", width, r.Form, r.Meaning[0]); !strings.Contains(page, want) {
				t.Errorf("%s: missing aligned row %q", s.Heading, want)
			}
			for _, more := range r.Meaning[1:] {
				if want := strings.Repeat(" ", width+4) + more + "\n"; !strings.Contains(page, want) {
					t.Errorf("%s: missing continuation %q", s.Heading, want)
				}
			}
		}
	}
}

func TestReadmeUsageBlockMatchesHelp(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	var block []string
	state := 0
	for l := range strings.SplitSeq(string(data), "\n") {
		switch state {
		case 0:
			if l == "### Usage" {
				state = 1
			}
		case 1:
			if l == "```text" {
				state = 2
			}
		case 2:
			if l == "```" {
				state = 3
			} else {
				block = append(block, l)
			}
		}
		if state == 3 {
			break
		}
	}
	if state != 3 {
		t.Fatal("README.md needs a ```text block under its first ### Usage heading")
	}
	got, want := strings.Join(block, "\n")+"\n", plainHelp()
	if got == want {
		return
	}
	gl, wl := helpLines(got), helpLines(want)
	for i := 0; i < len(gl) || i < len(wl); i++ {
		g, w := "", ""
		if i < len(gl) {
			g = gl[i]
		}
		if i < len(wl) {
			w = wl[i]
		}
		if g != w {
			t.Fatalf("README usage block differs from azm -h at line %d:\n readme: %q\n   help: %q", i+1, g, w)
		}
	}
	t.Fatal("README usage block differs from azm -h")
}

func TestShortPageSharesHeaderAndPointsAtHelp(t *testing.T) {
	sl, fl := helpLines(utl.RenderHelp(helpPage(true), false)), helpLines(plainHelp())
	for i := range 4 {
		if sl[i] != fl[i] {
			t.Fatalf("short page line %d = %q, want %q", i+1, sl[i], fl[i])
		}
	}
	if got := headingsOf(sl[4:]); strings.Join(got, ",") != "Usage,Examples" {
		t.Fatalf("short page headings = %v", got)
	}
	if last := sl[len(sl)-1]; !strings.Contains(last, "azm -h") {
		t.Fatalf("short page must end by naming azm -h, got %q", last)
	}
}
