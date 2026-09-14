package utl

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Help is one utility's help page: a four-line header followed by sections. Name
// prints on line one with the version, Description on line two with no trailing
// period, URL on line three with no scheme, and line four is blank.
type Help struct {
	Name        string
	Version     string
	Description string
	URL         string
	Sections    []Section
}

// Section is one help section: a heading, aligned rows, and Note, one optional
// trailing paragraph printed after a blank line.
type Section struct {
	Heading string
	Rows    []Row
	Note    []string
}

// Row is one form and its meaning; extra meaning lines continue at the meaning column.
type Row struct {
	Form    string
	Meaning []string
}

const (
	helpIndent     = "  "
	sgrHeading     = "\x1b[1;38;5;231m"
	sgrDescription = "\x1b[38;5;245m"
	sgrURL         = "\x1b[38;5;242m"
	sgrReset       = "\x1b[0m"
	optionsHeading = "Options"
)

// standardOptionRows end every Options section; the renderer appends them.
var standardOptionRows = []Row{
	{Form: "-v, --version", Meaning: []string{"Print the version"}},
	{Form: "-h, -?, --help", Meaning: []string{"Print this help"}},
}

// Select returns a copy of h that keeps only the named sections, in h's order.
func (h Help) Select(headings ...string) Help {
	keep := map[string]bool{}
	for _, name := range headings {
		keep[name] = true
	}
	out := h
	out.Sections = nil
	for _, s := range h.Sections {
		if keep[s.Heading] {
			out.Sections = append(out.Sections, s)
		}
	}
	return out
}

// RenderHelp renders h as help page text ending with a newline. With color on, the
// name, headings, description, and URL each carry one SGR span; with color off the
// output holds no escape sequence.
func RenderHelp(h Help, color bool) string {
	span := func(code, text string) string {
		if !color {
			return text
		}
		return code + text + sgrReset
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s v%s\n", span(sgrHeading, h.Name), h.Version)
	b.WriteString(span(sgrDescription, h.Description) + "\n")
	b.WriteString(span(sgrURL, h.URL) + "\n")
	for _, s := range h.Sections {
		b.WriteString("\n" + span(sgrHeading, s.Heading) + "\n")
		rows := s.Rows
		if s.Heading == optionsHeading {
			rows = append(append([]Row{}, rows...), standardOptionRows...)
		}
		writeRows(&b, rows)
		if len(s.Note) > 0 {
			b.WriteString("\n")
			for _, line := range s.Note {
				b.WriteString(helpIndent + line + "\n")
			}
		}
	}
	return b.String()
}

// writeRows prints rows with every meaning aligned two spaces past the longest form.
func writeRows(b *strings.Builder, rows []Row) {
	width := 0
	for _, r := range rows {
		if n := utf8.RuneCountInString(r.Form); n > width {
			width = n
		}
	}
	column := strings.Repeat(" ", len(helpIndent)+width+2)
	for _, r := range rows {
		if len(r.Meaning) == 0 {
			b.WriteString(helpIndent + r.Form + "\n")
			continue
		}
		fmt.Fprintf(b, "%s%-*s  %s\n", helpIndent, width, r.Form, r.Meaning[0])
		for _, more := range r.Meaning[1:] {
			b.WriteString(column + more + "\n")
		}
	}
}
