package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/datapointchris/helpnav/nav"
)

// action is one row of the key table. The key handler, the ? screen and the
// footer all read it, so none of them lists a key of its own.
type action struct {
	keys []string
	// shown is how the ? screen names the keys, and does is what it says they
	// do. A row with neither is listed with the row above it.
	shown, does string
	// hint is the row's word in the footer, or nil to leave it out. named
	// says the footer has room for the longer form.
	hint func(m Model, named bool) string
	// keep is how narrow a footer still carries the hint: one of footerSteps.
	keep int
	// beforeTree says the key means something while the tool is still being
	// read. Every other key waits for the tree.
	beforeTree bool
	run        func(*Model) tea.Cmd
}

// footerSteps is the footer from fullest to barest. Each step leaves out the
// hints kept for less than its keep, and only the first names the section
// space folds.
var footerSteps = []struct {
	named bool
	keep  int
}{{true, 0}, {false, 0}, {false, 1}, {false, 2}}

var actions = []action{
	{keys: []string{"j", "down"}, shown: "j k, down up", does: "move through the commands", hint: say("j/k move"), keep: 1, run: move((*nav.Nav).Next)},
	{keys: []string{"k", "up"}, run: move((*nav.Nav).Prev)},
	{keys: []string{"l", "right"}, shown: "l h, right left", does: "into a command, back out", hint: say("l/h in/out"), run: (*Model).descend},
	{keys: []string{"h", "left"}, run: (*Model).ascend},
	{keys: []string{"g", "home"}, shown: "g G, home end", does: "first, last", run: move((*nav.Nav).First)},
	{keys: []string{"G", "end"}, run: move((*nav.Nav).Last)},
	{keys: []string{"ctrl+d", "pgdown"}, shown: "ctrl+d ctrl+u, pgdn pgup", does: "scroll the help half a screen", run: scroll(func(h int) int { return max(h/2, 1) })},
	{keys: []string{"ctrl+u", "pgup"}, run: scroll(func(h int) int { return -max(h/2, 1) })},
	{keys: []string{"f"}, shown: "f b", does: "scroll the help a page", run: scroll(func(h int) int { return h })},
	{keys: []string{"b"}, run: scroll(func(h int) int { return -h })},
	{keys: []string{"J"}, shown: "J K", does: "scroll the help a line", run: scroll(func(int) int { return 1 })},
	{keys: []string{"K"}, run: scroll(func(int) int { return -1 })},
	{keys: []string{"]"}, shown: "[ ]", does: "previous, next help section", run: jump(1)},
	{keys: []string{"["}, run: jump(-1)},
	{keys: []string{" "}, shown: "space", does: "fold or open the top section", hint: Model.spaceHint, run: func(m *Model) tea.Cmd {
		m.help.Toggle()
		return nil
	}},
	{keys: []string{"enter"}, shown: "enter", does: "take this one and quit", hint: say("enter take"), keep: 1, run: func(m *Model) tea.Cmd {
		// Always "this is the one". Descending is l/right, so enter never has
		// to guess which of the two was meant.
		m.chosen = true
		return tea.Quit
	}},
	{keys: []string{"?"}, shown: "?", does: "show these keys", hint: say("? keys"), keep: 2, beforeTree: true, run: func(m *Model) tea.Cmd {
		m.keysOpen = true
		return nil
	}},
	{keys: []string{"q", "esc"}, shown: "q esc", does: "leave with nothing", hint: say("q quit"), keep: 2, beforeTree: true, run: func(*Model) tea.Cmd {
		return tea.Quit
	}},
}

// actionFor is the row holding key, or nil.
func actionFor(key string) *action {
	for i := range actions {
		if slices.Contains(actions[i].keys, key) {
			return &actions[i]
		}
	}
	return nil
}

func say(hint string) func(Model, bool) string {
	return func(Model, bool) string { return hint }
}

func move(step func(*nav.Nav)) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		step(m.nav)
		m.syncPreview()
		return nil
	}
}

func scroll(by func(height int) int) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		m.help.Scroll(by(m.help.Height()))
		return nil
	}
}

func jump(direction int) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		m.help.Jump(direction)
		return nil
	}
}

func (m *Model) descend() tea.Cmd {
	if m.nav.Pending() {
		// Named but not read. Fetching happens in a command so the interface
		// stays alive while it does.
		m.reading = true
		return fill(m.nav.Binary(), m.nav.Selected().Path, m.depth, m.textWidth())
	}
	if m.nav.Enter() {
		m.syncPreview()
	}
	return nil
}

func (m *Model) ascend() tea.Cmd {
	if m.nav.Leave() {
		m.syncPreview()
	}
	return nil
}

// spaceHint names what space would do, and to which section where there is
// room. A help with no headings has nothing for space to do, so no hint.
func (m Model) spaceHint(named bool) string {
	section, ok := m.help.Target()
	if !ok {
		return ""
	}
	hint := "space fold"
	if m.help.Folded(section) {
		hint = "space open"
	}
	if named {
		hint += " " + ansi.Truncate(section.Name, footerSectionWidth, "…")
	}
	return hint
}

// footerKeys is the key hints at each of footerSteps.
func (m Model) footerKeys() []string {
	steps := make([]string, 0, len(footerSteps))
	for _, step := range footerSteps {
		var hints []string
		for _, a := range actions {
			if a.hint == nil || a.keep < step.keep {
				continue
			}
			if hint := a.hint(m, step.named); hint != "" {
				hints = append(hints, hint)
			}
		}
		steps = append(steps, "  "+strings.Join(hints, " · "))
	}
	return steps
}
