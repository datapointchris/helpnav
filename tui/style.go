package tui

import "github.com/charmbracelet/lipgloss"

// Colors are declared as ANSI 0-15 rather than as hex, so the browser inherits
// whatever palette the reader's terminal is already using instead of imposing
// one that clashes with it.
var (
	rowStyle = lipgloss.NewStyle()

	// selectedStyle marks the cursor in the column being moved.
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("6")).
			Bold(true)

	// trailStyle marks the row in the parent column that was walked through to
	// get here. Dimmer than the cursor, because it says where you came from
	// rather than what you are choosing.
	trailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true)

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)
