// Package helpbody keeps a window over one command's help screen, with some of
// its sections folded away.
//
// It renders nothing and knows nothing about a terminal, for the reason nav
// does not: what a fold hides and where the window lands after one are what is
// worth testing, and a test of them needs no screen. Where the sections start
// is clisurface's to say, because it reads the same screens to walk the tool.
package helpbody

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/datapointchris/clisurface"
)

// tab is what a tab in a help screen becomes. kubectl indents its option
// descriptions with one, and a tab's width is otherwise the terminal's guess.
const tab = "    "

// minRoom is the narrowest a wrapped description runs. A description whose
// own column leaves less than this moves left to get it.
const minRoom = 20

// nextLineHang is where a description goes when it cannot sit beside its flag,
// which is where clap puts every description in its long help.
const nextLineHang = 10

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

// Parse splits a help screen at the headings [clisurface.Sections] finds.
func Parse(text string) Body {
	text = strings.TrimRight(strings.ReplaceAll(text, "\t", tab), "\n")
	if strings.TrimSpace(ansi.Strip(text)) == "" {
		return Body{}
	}
	body := Body{lines: strings.Split(text, "\n")}
	for i, line := range body.lines {
		body.lines[i] = strings.TrimRight(line, " ")
	}
	for _, s := range clisurface.Sections(text) {
		end := min(s.End, len(body.lines))
		last := end
		for last > s.Line+1 && blank(body.lines[last-1]) {
			last--
		}
		body.sections = append(body.sections, Section{Name: s.Title, Lines: last - s.Line - 1, heading: s.Line, end: end})
	}
	return body
}

func blank(line string) bool { return strings.TrimSpace(ansi.Strip(line)) == "" }

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

// layout lays the body out at width. A section named in folds shows only its
// heading row.
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
			for n, text := range wrapLine(b.lines[i], width) {
				row := Row{Text: text, section: section, line: i}
				if n == 0 {
					row.Mark = Open
				}
				rows = append(rows, row)
			}
			continue
		}
		until := len(b.lines)
		if next < len(b.sections) {
			until = b.sections[next].heading
		}
		texts, used := fit(b.lines[i:until], width)
		for _, text := range texts {
			rows = append(rows, Row{Text: text, section: section, line: i})
		}
		i += used - 1
	}
	return rows
}

// fit lays out the first of lines at width, with the lines after it that
// continue its description, and says how many lines it used.
//
// Lines that fit are drawn as the tool printed them. An option whose lines do
// not is laid out again with its description as one paragraph, because a tool
// asked for a narrow width can still pad its flag column past it, and clap then
// prints one word of description a line. The paragraph wraps down its own
// column where that leaves room, and under its flag at nextLineHang otherwise,
// so no word is cut in two.
func fit(lines []string, width int) ([]string, int) {
	plain := ansi.Strip(lines[0])
	termEnd, column, ok := descriptionColumn(plain)
	if !ok {
		return wrapLine(lines[0], width), 1
	}
	n := 1
	for n < len(lines) && continues(lines[n], column) {
		n++
	}
	entry := lines[:n]
	if !anyWider(entry, width) {
		return append([]string(nil), entry...), n
	}
	parts := make([]string, 0, n)
	for _, line := range entry {
		parts = append(parts, strings.TrimSpace(ansi.TruncateLeft(line, column, "")))
	}
	description := strings.Join(parts, " ")
	term := ansi.Cut(lines[0], 0, termEnd)

	hang, rows := column, []string(nil)
	if width-column < minRoom {
		indent := len(plain) - len(strings.TrimLeft(plain, " "))
		hang = min(max(nextLineHang, indent+2), max(width-minRoom, 0))
		rows = append(rows, ansi.Truncate(term, width, "…"))
		term = ""
	}
	for i, row := range strings.Split(ansi.Wrap(description, width-hang, ""), "\n") {
		lead := strings.Repeat(" ", hang)
		if i == 0 && term != "" {
			lead = term + strings.Repeat(" ", hang-termEnd)
		}
		rows = append(rows, lead+row)
	}
	return rows, n
}

func anyWider(lines []string, width int) bool {
	for _, line := range lines {
		if ansi.StringWidth(line) > width {
			return true
		}
	}
	return false
}

// continues says a line carries on the description that starts at column.
func continues(line string, column int) bool {
	text := ansi.Strip(line)
	trimmed := strings.TrimLeft(text, " ")
	return trimmed != "" && len(text)-len(trimmed) == column
}

// wrapLine breaks one line into rows of width, each continuing at the line's
// indent where that leaves minRoom, and further left where it does not.
func wrapLine(line string, width int) []string {
	if ansi.StringWidth(line) <= width {
		return []string{line}
	}
	text := ansi.Strip(line)
	indent := len(text) - len(strings.TrimLeft(text, " "))
	hang := min(indent, max(width-minRoom, 0))
	var rows []string
	for _, row := range strings.Split(ansi.Wrap(ansi.TruncateLeft(line, indent, ""), width-hang, ""), "\n") {
		rows = append(rows, strings.Repeat(" ", hang)+row)
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
	r.keepInText()
}

// Height is how many rows the window shows.
func (r *Reading) Height() int { return r.height }

// Window is the rows that show, top first. It holds fewer than Height rows
// where a fold or a jump left the top row within a screen of the end.
func (r *Reading) Window() []Row {
	return r.rows[r.top:min(r.top+r.height, len(r.rows))]
}

// Scroll moves the window by lines, stopping where the last row reaches the
// window's bottom. A window already further down than that, as a fold at the
// end leaves it, moves only up.
func (r *Reading) Scroll(lines int) {
	r.top = max(min(r.top+lines, max(len(r.rows)-r.height, r.top)), 0)
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
// window brings that heading to the top, and the window stays there however
// little text follows, so the heading is still the target and the next press
// opens what this one closed.
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
	r.keepInText()
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
	r.keepInText()
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

// keepInText holds the top row on a row of the text.
func (r *Reading) keepInText() {
	r.top = max(min(r.top, len(r.rows)-1), 0)
}
