// Package tui draws a command tree as three columns and lets you walk it.
//
// The state of *where you are* lives in [nav.Nav], not here. This package owns
// only what that looks like and which keys move it, so a change to how the tree
// is navigated is testable without a terminal and a change to how it is drawn
// cannot break the navigation.
package tui

import (
	"fmt"
	"os"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/datapointchris/clisurface"
	"github.com/datapointchris/gotermboard/breadcrumb"

	"github.com/datapointchris/helpnav/helpbody"
	"github.com/datapointchris/helpnav/nav"
)

// Column widths are a ratio rather than fixed sizes, so the same layout holds
// in a narrow split and a full-screen terminal. Reading gets the most room
// because the description is what the browser exists to show.
const (
	parentShare  = 1
	currentShare = 2
	previewShare = 4
	totalShare   = parentShare + currentShare + previewShare
)

// gutter is the cells left of the help text, where a heading's chevron sits.
// Every row keeps it, so no column of the help moves as a section folds.
const gutter = 2

const footerSectionWidth = 20

// surfaceMsg carries a finished reading back into the loop.
type surfaceMsg struct{ tool *clisurface.Tool }

// filledMsg carries the children of one command that had been named but not
// read, for a surface too large to read whole.
type filledMsg struct{ node *clisurface.Node }

// failedMsg carries a reading that could not be done, which is what a name that
// is not on PATH produces.
type failedMsg struct{ err error }

// Model is the whole browser state.
//
// nav is a pointer, so the copies bubbletea makes of this struct all point at
// one cursor. That is deliberate rather than an oversight: exactly one model is
// alive at a time and the runtime never revisits an old copy, so there is
// nothing for sharing to corrupt. The purity would only buy something if we
// wanted to replay history, which a help browser does not.
type Model struct {
	binary string
	depth  int

	nav  *nav.Nav
	help *helpbody.Reading
	err  error

	// folds names the help sections folded away. It outlives each command's
	// help, so a section closed in one stays closed in the next that prints it.
	folds map[string]bool

	width   int
	height  int
	sized   bool
	reading bool

	keysOpen bool

	// chosen records that the reader stopped on a command rather than walking
	// away, which is the difference between printing argv and printing nothing.
	chosen bool
}

// New builds a browser that will read binary once it knows how wide it is.
func New(binary string, depth int) Model {
	return Model{binary: binary, depth: depth, folds: map[string]bool{}}
}

// Init is the first Cmd the runtime runs. Nothing happens yet: reading the tool
// needs a width, and the width arrives on its own as the first message.
func (m Model) Init() tea.Cmd { return nil }

// read is the side effect, expressed as a value.
//
// This is the shape every bubbletea effect takes: Update never does I/O, it
// returns a function the runtime runs elsewhere, whose result comes back as an
// ordinary message. That is what keeps Update a pure fold over messages, and it
// is why a tool that takes three seconds to read leaves the interface alive
// instead of freezing it.
func read(binary string, depth, width int) tea.Cmd {
	return func() tea.Msg {
		tool, err := clisurface.Extract(binary, clisurface.Options{
			WithBody: true,
			MaxDepth: depth,
			Runner:   clisurface.DisplayRunner(width),
		})
		if err != nil {
			return failedMsg{err}
		}
		return surfaceMsg{tool}
	}
}

// fill reads the children of one command that the first walk named but did not
// read, so descending into it costs one read rather than the whole tree.
func fill(binary string, path []string, depth, width int) tea.Cmd {
	return func() tea.Msg {
		node, err := clisurface.ExtractAt(binary, path, clisurface.Options{
			WithBody: true,
			MaxDepth: depth,
			Runner:   clisurface.DisplayRunner(width),
		})
		if err != nil {
			return failedMsg{err}
		}
		return filledMsg{node}
	}
}

// Update folds one message into a new model.
//
// The signature returns tea.Model rather than Model because the runtime holds
// the interface. That is why every branch returns m by value: this method never
// mutates the receiver, it describes what the next state is.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		first := !m.sized
		m.sized = true
		// Read once, on the first size. A resize afterwards re-lays the panes
		// out but does not read the tool again — that would spend the whole
		// cost of a read on a window drag.
		if first && !m.reading && m.nav == nil {
			m.reading = true
			return m, read(m.binary, m.depth, m.textWidth())
		}
		return m, nil

	case surfaceMsg:
		m.reading = false
		m.nav = nav.New(msg.tool)
		m.syncPreview()
		return m, nil

	case filledMsg:
		m.reading = false
		// Enter only after the children are attached, so the descent lands on a
		// level with something to draw.
		if m.nav.Fill(msg.node.Children) && m.nav.Enter() {
			m.syncPreview()
		}
		return m, nil

	case failedMsg:
		m.reading = false
		m.err = msg.err
		return m, tea.Quit

	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

// key handles one keypress. Split out of Update so the message dispatch stays
// one screen and the bindings read as a list.
func (m Model) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	// Any key closes the table, so esc backs out of it and leaves the browser
	// open.
	if m.keysOpen {
		m.keysOpen = false
		return m, nil
	}
	a := actionFor(msg.String())
	// While the tool is still being read, quitting and ? are the only keys that
	// can mean anything.
	if a == nil || (m.nav == nil && !a.beforeTree) {
		return m, nil
	}
	cmd := a.run(&m)
	return m, cmd
}

// Command is the argv the reader stopped on, or nil if they walked away.
func (m Model) Command() []string {
	if !m.chosen || m.nav == nil {
		return nil
	}
	return m.nav.Command()
}

// Err is why the browser closed, when it closed because it could not read.
func (m Model) Err() error { return m.err }

// layout fits the help to the preview column again, keeping its top line.
func (m *Model) layout() {
	if m.help != nil {
		m.help.Resize(m.textWidth(), m.helpHeight())
	}
}

// bodyHeight is what the footer and the gap above it leave for the columns.
func (m Model) bodyHeight() int {
	return max(m.height-2, 1)
}

// helpHeight is the preview column under its band line.
func (m Model) helpHeight() int {
	return max(m.bodyHeight()-1, 1)
}

// textWidth is the preview column right of the chevron gutter, which is the
// width a tool is asked to wrap its help at.
func (m Model) textWidth() int {
	return max(m.previewWidth()-gutter, 1)
}

func (m Model) columnWidth(share int) int {
	return max(m.width*share/totalShare, 1)
}

func (m Model) previewWidth() int {
	return max(m.width-m.columnWidth(parentShare)-m.columnWidth(currentShare), 1)
}

// previewed is the command whose help shows. The selection is preferred over
// the current command, because the reader is deciding about the thing they are
// pointing at.
func (m Model) previewed() *clisurface.Node {
	if node := m.nav.Selected(); node != nil {
		return node
	}
	return m.nav.Node()
}

// syncPreview opens the help of whatever is under the cursor now, at its top.
func (m *Model) syncPreview() {
	if m.nav == nil {
		return
	}
	m.help = helpbody.NewReading(helpbody.Parse(m.previewed().Body), m.folds, m.textWidth(), m.helpHeight())
}

// View draws the three columns and the command being assembled.
func (m Model) View() string {
	switch {
	case m.err != nil:
		// Draw nothing. The failure is returned to the caller, which reports it
		// on a torn-down terminal; rendering it here as well prints it twice.
		return ""
	case !m.sized:
		return ""
	case m.keysOpen:
		return keysScreen()
	case m.nav == nil:
		return hintStyle.Render("reading " + m.binary + "…")
	}

	body := m.bodyHeight()
	parentNode, parentSel := m.nav.Parent()

	left := column(childNames(parentNode), parentSel, m.columnWidth(parentShare), body, false)
	middle := column(names(m.nav.Children()), m.nav.Cursor(), m.columnWidth(currentShare), body, true)
	right := lipgloss.NewStyle().Width(m.previewWidth()).Height(body).Render(m.helpPane())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)
	return panes + "\n" + m.footer()
}

// helpPane is the band line naming what the help belongs to, then the window
// over the help with each heading's chevron in the gutter.
func (m Model) helpPane() string {
	width := m.previewWidth()
	lines := []string{m.band(width)}
	for _, row := range m.help.Window() {
		lines = append(lines, helpRow(row, width))
	}
	return strings.Join(lines, "\n")
}

// band names the command whose help shows, then the section the top row sits
// in once that section's heading has scrolled off. It holds the top of the
// column whatever the scroll, so the help never has to be paged back to learn
// whose it is.
func (m Model) band(width int) string {
	words := append([]string{m.nav.Binary()}, m.previewed().Path...)
	segments := slices.Clone(words)
	if section, ok := m.help.Sticky(); ok {
		segments = append(segments, section.Name)
	}
	fitted := fitPath(segments, width-gutter)
	command := len(words) - 1 - (len(segments) - len(fitted))
	painted := make([]string, len(fitted))
	for i, segment := range fitted {
		painted[i] = segment
		if i == command {
			painted[i] = previewedStyle.Render(segment)
		}
	}
	return strings.Repeat(" ", gutter) + strings.Join(painted, hintStyle.Render(pathSeparator))
}

const pathSeparator = " › "

// fitPath cuts segments to fit room as gotermboard's reader cuts its top line,
// so a long section name gives way before any command name does.
func fitPath(segments []string, room int) []string {
	widths := make([]int, len(segments))
	for i, segment := range segments {
		widths[i] = ansi.StringWidth(segment)
	}
	dropped, widths := breadcrumb.Fit(widths, ansi.StringWidth(pathSeparator), room)
	fitted := make([]string, len(widths))
	for i, segment := range segments[dropped:] {
		fitted[i] = ansi.Truncate(segment, widths[i], "…")
	}
	return fitted
}

// helpRow draws one row of help behind its gutter. A folded heading says how
// many lines it holds, so a closed section still shows its size.
func helpRow(row helpbody.Row, width int) string {
	mark := strings.Repeat(" ", gutter)
	switch row.Mark {
	case helpbody.Open:
		mark = "▾ "
	case helpbody.Folded:
		mark = "▸ "
	case helpbody.NoMark:
	}
	text := row.Text
	// A tool's color left open at the end of a row would run into the next
	// pane, because the columns are joined line by line.
	if strings.Contains(text, "\x1b") {
		text += ansi.ResetStyle
	}
	if row.Mark == helpbody.Folded {
		text += hintStyle.Render(" · " + lineCount(row.Hidden))
	}
	return ansi.Truncate(mark+text, width, "…")
}

func lineCount(n int) string {
	if n == 1 {
		return "1 line"
	}
	return fmt.Sprintf("%d lines", n)
}

// footer shows the command the reader would leave with, which is the answer the
// browser exists to produce. Key hints give way before the command is cut.
func (m Model) footer() string {
	cmd := strings.Join(m.nav.Command(), " ")
	var keys string
	for _, keys = range m.footerKeys() {
		if lipgloss.Width(cmd)+lipgloss.Width(keys) <= m.width {
			break
		}
	}
	room := max(m.width-lipgloss.Width(keys), 0)
	return commandStyle.Render(ansi.Truncate(cmd, room, "…")) + hintStyle.Render(keys)
}

func keysScreen() string {
	rows := make([]string, 0, len(actions)+2)
	for _, a := range actions {
		if a.does != "" {
			rows = append(rows, commandStyle.Render(fmt.Sprintf(" %-26s", a.shown))+rowStyle.Render(a.does))
		}
	}
	rows = append(rows, "", hintStyle.Render(" any key closes this"))
	return strings.Join(rows, "\n")
}

func names(nodes []*clisurface.Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Name)
	}
	return out
}

func childNames(parent *clisurface.Node) []string {
	if parent == nil {
		return nil
	}
	return names(parent.Children)
}

// column renders one list, scrolled so the cursor stays on screen.
func column(rows []string, cursor, width, height int, active bool) string {
	if height < 1 || width < 1 {
		return ""
	}
	first := 0
	if cursor >= height {
		first = cursor - height + 1
	}

	var b strings.Builder
	for i := range height {
		row := first + i
		if row < len(rows) {
			// Truncate with the ANSI-aware helper rather than slicing: a byte
			// slice cuts a multi-byte rune in half, and a rune slice cuts an
			// escape sequence in half, which bleeds color into the next pane.
			text := " " + ansi.Truncate(rows[row], max(width-1, 0), "…")
			switch {
			case row == cursor && active:
				b.WriteString(selectedStyle.Width(width).Render(text))
			case row == cursor:
				b.WriteString(trailStyle.Width(width).Render(text))
			default:
				b.WriteString(rowStyle.Width(width).Render(text))
			}
		}
		if i < height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Run opens the browser and returns the command the reader stopped on.
//
// The interface is drawn on stderr so stdout carries only the answer. That is
// what lets `$(helpnav docker)` work: the browser paints on the terminal while
// the chosen command goes down the pipe.
func Run(binary string, depth int) ([]string, error) {
	program := tea.NewProgram(
		New(binary, depth),
		tea.WithAltScreen(),
		tea.WithOutput(os.Stderr),
	)
	final, err := program.Run()
	if err != nil {
		return nil, err
	}
	model, ok := final.(Model)
	if !ok {
		return nil, nil
	}
	if model.Err() != nil {
		return nil, model.Err()
	}
	return model.Command(), nil
}
