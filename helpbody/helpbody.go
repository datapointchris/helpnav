// Package helpbody splits one command's help screen at its headings, and keeps
// a window over it with some of those sections folded away.
//
// It renders nothing and knows nothing about a terminal, for the reason nav
// does not: which section a line belongs to, what a fold hides and where the
// window lands after one are what is worth testing, and a test of them needs
// no screen.
package helpbody

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// tab is what a tab in a help screen becomes. kubectl indents its option
// descriptions with one, and a tab's width is otherwise the terminal's guess.
const tab = "    "

// maxHeadingWords is the longest a heading runs. git's "Low-level Commands /
// Syncing Repositories" is the longest seen, and a sentence is longer.
const maxHeadingWords = 6

// Section is one heading of a help screen and the lines under it, up to the
// next heading.
type Section struct {
	// Name is the heading's words, without its colors or a trailing colon. A
	// fold is keyed by it, so closing "Global options" in one command's help
	// closes it in every other help that prints the same heading.
	Name string
	// Lines is how many lines sit under the heading, leaving out the blank
	// lines that end the section.
	Lines int

	heading int
	end     int
}

// Body is one help screen split at its headings. The lines above the first
// heading belong to no section, so they never fold.
type Body struct {
	lines    []string
	sections []Section
}

// Parse splits a help screen at its headings.
//
// A heading is a short line at the left edge, after a blank line, with an
// indented line or an underline rule under it. That one shape covers clap's
// "Options:", gh's "GENERAL COMMANDS", cobra's "Global Flags:" and git's
// "Main Porcelain Commands". A line carrying its content after a colon, as
// "Usage: uv add [OPTIONS]" does, is not a heading, so the description
// docker prints under its usage line never folds away with it.
func Parse(text string) Body {
	text = strings.TrimRight(strings.ReplaceAll(text, "\t", tab), "\n")
	if strings.TrimSpace(ansi.Strip(text)) == "" {
		return Body{}
	}
	body := Body{lines: strings.Split(text, "\n")}
	for i := range body.lines {
		if !body.headingAt(i) {
			continue
		}
		if n := len(body.sections); n > 0 {
			body.sections[n-1].end = i
		}
		body.sections = append(body.sections, Section{Name: headingName(body.lines[i]), heading: i})
	}
	for i := range body.sections {
		s := &body.sections[i]
		if i == len(body.sections)-1 {
			s.end = len(body.lines)
		}
		last := s.end
		for last > s.heading+1 && blank(body.lines[last-1]) {
			last--
		}
		s.Lines = last - s.heading - 1
	}
	return body
}

func (b Body) headingAt(i int) bool {
	if i+1 >= len(b.lines) || (i > 0 && !blank(b.lines[i-1])) {
		return false
	}
	text := strings.TrimRight(ansi.Strip(b.lines[i]), " ")
	if text == "" || text[0] == ' ' || len(strings.Fields(text)) > maxHeadingWords {
		return false
	}
	if colon := strings.IndexByte(text, ':'); colon >= 0 && colon != len(text)-1 {
		return false
	}
	end, _ := utf8.DecodeLastRuneInString(text)
	if !unicode.IsLetter(end) && !unicode.IsDigit(end) && end != ':' && end != ')' {
		return false
	}
	return indented(b.lines[i+1]) || underline(b.lines[i+1])
}

func headingName(line string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(ansi.Strip(line)), ":"))
}

func blank(line string) bool { return strings.TrimSpace(ansi.Strip(line)) == "" }

func indented(line string) bool {
	text := ansi.Strip(line)
	return strings.HasPrefix(text, " ") && strings.TrimSpace(text) != ""
}

func underline(line string) bool {
	text := strings.TrimSpace(ansi.Strip(line))
	return len(text) >= 3 && strings.Trim(text, "-=") == ""
}

// Mark is what a row's chevron cell holds.
type Mark int

const (
	// NoMark is every row but a heading, because nothing else folds.
	NoMark Mark = iota
	// Open is a heading whose lines show under it.
	Open
	// Folded is a heading whose lines are out of view.
	Folded
)

// Row is one line of the help as laid out at a width.
type Row struct {
	// Text is the line in the tool's own colors, no wider than the width.
	Text string
	// Mark is set on the first row of a heading.
	Mark Mark
	// Hidden is how many lines a folded heading keeps out of view.
	Hidden int

	section int
	line    int
}

// layout lays the body out at width, with each section folds names hidden
// under its heading. A line wider than width wraps rather than being cut,
// because gh wraps its help in source and ignores the width it is given.
func (b Body) layout(width int, folds map[string]bool) []Row {
	var rows []Row
	section, next := -1, 0
	for i := 0; i < len(b.lines); i++ {
		if next < len(b.sections) && b.sections[next].heading == i {
			section, next = next, next+1
			s := b.sections[section]
			if folds[s.Name] {
				rows = append(rows, Row{Text: ansi.Truncate(b.lines[i], width, "…"), Mark: Folded, Hidden: s.Lines, section: section, line: i})
				i = s.end - 1
				continue
			}
			for n, text := range wrap(b.lines[i], width) {
				row := Row{Text: text, section: section, line: i}
				if n == 0 {
					row.Mark = Open
				}
				rows = append(rows, row)
			}
			continue
		}
		for _, text := range wrap(b.lines[i], width) {
			rows = append(rows, Row{Text: text, section: section, line: i})
		}
	}
	return rows
}

// minHangRoom is the narrowest a wrapped line's continuation runs. A line
// hanging further right than that hangs at the width less this instead.
const minHangRoom = 10

// wrap breaks one line into rows of width. Where an option's flag and its
// description are two columns, the description wraps down its own column, so
// it reads as the tool laid it out. A tool asked for a narrow width can still
// lay its help out wider: docker pads its flag column to 41 cells, and clap
// to 49 with one word of description a line. Such a column moves left to fit,
// and a flag too long to share a row with it takes a row of its own.
func wrap(line string, width int) []string {
	if ansi.StringWidth(line) <= width {
		return []string{line}
	}
	text := ansi.Strip(line)
	limit := max(width-minHangRoom, 0)
	from := len(text) - len(strings.TrimLeft(text, " "))
	var rows []string
	term, termEnd := "", 0
	if end, column, ok := descriptionColumn(text); ok {
		from, termEnd = column, end
		term = ansi.Cut(line, 0, end)
	}
	hang := min(from, limit)
	if term != "" && termEnd+2 > hang {
		rows = append(rows, ansi.Truncate(term, width, "…"))
		term = ""
	}
	for i, row := range strings.Split(ansi.Wrap(ansi.TruncateLeft(line, from, ""), width-hang, ""), "\n") {
		lead := strings.Repeat(" ", hang)
		if i == 0 && term != "" {
			lead = term + strings.Repeat(" ", hang-termEnd)
		}
		rows = append(rows, lead+row)
	}
	return rows
}

// descriptionColumn finds a flag, an argument or a single word followed by two
// or more spaces and then its description, and says where the term ends and
// the description starts. A sentence with two spaces after its period is none
// of those.
func descriptionColumn(text string) (termEnd, column int, ok bool) {
	indent := len(text) - len(strings.TrimLeft(text, " "))
	gap := strings.Index(text[indent:], "  ")
	if gap < 0 {
		return 0, 0, false
	}
	term := text[indent : indent+gap]
	if !strings.ContainsAny(term[:1], "-[<") && strings.Contains(term, " ") {
		return 0, 0, false
	}
	start := len(text) - len(strings.TrimLeft(text[indent+gap:], " "))
	if start == len(text) {
		return 0, 0, false
	}
	return ansi.StringWidth(text[:indent+gap]), ansi.StringWidth(text[:start]), true
}

// Reading is a window over a Body laid out at one width and height, with the
// sections a shared set names folded away.
type Reading struct {
	body   Body
	folds  map[string]bool
	width  int
	height int
	rows   []Row
	top    int
}

// NewReading opens a window at the top of body. folds is shared rather than
// copied, so a section closed in one command's help stays closed in the next.
func NewReading(body Body, folds map[string]bool, width, height int) *Reading {
	r := &Reading{body: body, folds: folds, width: max(width, 1), height: max(height, 1)}
	r.rows = body.layout(r.width, folds)
	return r
}

// Resize lays the body out again and keeps the same line at the top.
func (r *Reading) Resize(width, height int) {
	line := r.topLine()
	r.width, r.height = max(width, 1), max(height, 1)
	r.rows = r.body.layout(r.width, r.folds)
	r.top = r.firstRowOf(line)
	r.clamp()
}

// Height is how many rows the window shows.
func (r *Reading) Height() int { return r.height }

// Window is the rows that show, top first.
func (r *Reading) Window() []Row {
	return r.rows[r.top:min(r.top+r.height, len(r.rows))]
}

// Scroll moves the window by lines, stopping where either end of the text
// reaches the window's edge.
func (r *Reading) Scroll(lines int) {
	r.top += lines
	r.clamp()
}

// Target is the section space acts on: the one holding the top row, or the
// first section where the top row sits above every heading.
func (r *Reading) Target() (Section, bool) {
	i := r.target()
	if i < 0 {
		return Section{}, false
	}
	return r.body.sections[i], true
}

// Folded says whether a section's lines are out of view.
func (r *Reading) Folded(s Section) bool { return r.folds[s.Name] }

// Toggle folds the target section, or opens it where it is folded, and reports
// whether there was one. A section folded while its heading sat above the
// window brings that heading to the top, so it is still the target and the
// next press opens what this one closed.
func (r *Reading) Toggle() bool {
	i := r.target()
	if i < 0 {
		return false
	}
	s := r.body.sections[i]
	scrolledOff := len(r.rows) > 0 && r.rows[r.top].section == i && r.rows[r.top].line != s.heading
	r.folds[s.Name] = !r.folds[s.Name]
	r.rows = r.body.layout(r.width, r.folds)
	if scrolledOff {
		r.top = r.firstRowOf(s.heading)
	}
	r.clamp()
	return true
}

// Jump brings the next heading below the top row to the top, or the previous
// one above it where direction is negative, and reports whether the window
// moved. Back from the first heading is the top of the text, where the
// description sits.
func (r *Reading) Jump(direction int) bool {
	from, step := r.top, 1
	if direction < 0 {
		r.top, step = 0, -1
	}
	for i := from + step; i >= 0 && i < len(r.rows); i += step {
		if r.rows[i].Mark != NoMark {
			r.top = i
			break
		}
	}
	r.clamp()
	return r.top != from
}

// Sticky is the section holding the top row once its heading has scrolled
// off, which a view names above the window so the text keeps its place.
func (r *Reading) Sticky() (Section, bool) {
	if len(r.rows) == 0 {
		return Section{}, false
	}
	row := r.rows[r.top]
	if row.section < 0 || row.line == r.body.sections[row.section].heading {
		return Section{}, false
	}
	return r.body.sections[row.section], true
}

func (r *Reading) target() int {
	switch {
	case len(r.body.sections) == 0:
		return -1
	case len(r.rows) > 0 && r.rows[r.top].section >= 0:
		return r.rows[r.top].section
	}
	return 0
}

func (r *Reading) topLine() int {
	if len(r.rows) == 0 {
		return 0
	}
	return r.rows[r.top].line
}

func (r *Reading) firstRowOf(line int) int {
	for i, row := range r.rows {
		if row.line >= line {
			return i
		}
	}
	return len(r.rows)
}

func (r *Reading) clamp() {
	r.top = max(min(r.top, len(r.rows)-r.height), 0)
}
