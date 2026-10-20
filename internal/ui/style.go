package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Catppuccin Macchiato.
var (
	mauve    = lipgloss.Color("#c6a0f6")
	text     = lipgloss.Color("#cad3f5")
	surface0 = lipgloss.Color("#363a4f")
	surface1 = lipgloss.Color("#494d64")
)

// Colour is used sparingly: one accent, and faint text for anything
// secondary. Plain text keeps the terminal's own foreground; anything with a
// background sets its own foreground too, so it reads on light themes as well.
var (
	accent = lipgloss.NewStyle().Foreground(mauve)
	title  = accent.Bold(true)
	faint  = lipgloss.NewStyle().Faint(true)
	band   = lipgloss.NewStyle().Foreground(text).Background(surface0)
	keycap = lipgloss.NewStyle().Foreground(text).Background(surface1).Padding(0, 1)
)

// pageWidth is how wide a screen draws: up to 72 columns, less on a narrow
// terminal.
func pageWidth(termWidth int) int {
	if termWidth == 0 {
		return 72
	}
	return max(40, min(72, termWidth-4))
}

// listRow renders one line of a list. The selected line gets the band, with
// its head in the accent colour; dim lines are for things not open yet.
// head and rest must be unstyled, so the band can colour them.
func listRow(head, rest string, width int, selected, dim bool) string {
	switch {
	case selected:
		pad := max(0, width-2-lipgloss.Width(head)-lipgloss.Width(rest))
		return band.Foreground(mauve).Render("▌ "+head) + band.Render(rest+strings.Repeat(" ", pad))
	case dim:
		return faint.Render("  " + head + rest)
	default:
		return "  " + head + faint.Render(rest)
	}
}

// spread puts left and right at either end of a line.
func spread(left, right string, width int) string {
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

// hint renders keys as keycaps, followed by what they do.
func hint(action string, keys ...string) string {
	var s string
	for _, k := range keys {
		s += keycap.Render(k) + " "
	}
	return s + faint.Render(action)
}

// hints joins several hints into one footer line.
func hints(hs ...string) string { return strings.Join(hs, "   ") }

// stars renders a 0–3 star score as plain text.
func stars(n int) string { return strings.Repeat("★", n) + strings.Repeat("☆", 3-n) }
