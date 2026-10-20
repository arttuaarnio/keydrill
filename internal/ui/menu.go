package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type menuItem struct {
	title, desc string
	open        tea.Msg // the screen it opens; nil while it's not built
	later       bool    // listed so players know it's planned, but not playable yet
}

var menuItems = []menuItem{
	{title: "Shell", desc: "Run real commands in a throwaway folder", open: showTopics{}},
	{title: "Vim motions", desc: "Coming later", later: true},
	{title: "Vim editing", desc: "Coming later", later: true},
	{title: "Stats", desc: "Best scores, streaks and weak spots"},
	{title: "Quit"},
}

// menu is the main menu.
type menu struct {
	cursor int
	note   string // one-line message shown under the list
}

func (m menu) update(msg tea.Msg) (screen, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	m.note = ""
	switch key.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(menuItems)-1 {
			m.cursor++
		}
	case "enter", "space":
		it := menuItems[m.cursor]
		switch {
		case it.title == "Quit":
			return m, tea.Quit
		case it.open != nil:
			return m, show(it.open)
		case it.later:
			m.note = it.title + " is coming later."
		default:
			m.note = it.title + " is not built yet."
		}
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m menu) view(width, height int) string {
	var b strings.Builder
	b.WriteString(title.Render("keydrill") + "\n")
	b.WriteString(faint.Render("Practise shell and Vim commands until they stick.") + "\n\n")

	// Pad descriptions to the longest one so the band is the same width on
	// every row.
	descWidth := 0
	for _, it := range menuItems {
		descWidth = max(descWidth, lipgloss.Width(it.desc))
	}
	for i, it := range menuItems {
		head := fmt.Sprintf("%-13s", it.title)
		b.WriteString(listRow(head, " "+it.desc+" ", 2+13+descWidth+2, i == m.cursor, it.later) + "\n")
	}

	b.WriteString("\n" + faint.Render(m.note) + "\n\n")
	b.WriteString(hints(hint("move", "j", "k"), hint("select", "enter"), hint("quit", "q")))
	return b.String()
}
