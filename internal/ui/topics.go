package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/arttuaarnio/keydrill/internal/drills"
	"github.com/arttuaarnio/keydrill/internal/progress"
)

// key is a drill's name in the save file.
func key(t drills.Topic, d drills.Drill) string { return t.ID + "/" + d.ID }

// topicOpen reports whether topic i can be played: it has drills, and it's
// either the first topic or the exam of the topic before it is passed.
func topicOpen(p *progress.Progress, i int) bool {
	if len(drills.Shell[i].Drills) == 0 {
		return false
	}
	if i == 0 {
		return true
	}
	prev := drills.Shell[i-1]
	return len(prev.Drills) > 0 && p.Solved(key(prev, prev.Drills[len(prev.Drills)-1]))
}

// drillOpen reports whether drill d of topic t can be played: its topic is
// open, and the drill before it is solved.
func drillOpen(p *progress.Progress, t, d int) bool {
	if d == 0 {
		return topicOpen(p, t)
	}
	topic := drills.Shell[t]
	return p.Solved(key(topic, topic.Drills[d-1]))
}

// skill is the share of all stars earned so far, as a percentage.
func skill(p *progress.Progress) int {
	earned, possible := 0, 0
	for _, t := range drills.Shell {
		for _, d := range t.Drills {
			possible += 3
			earned += p.Drills[key(t, d)].Stars
		}
	}
	if possible == 0 {
		return 0
	}
	return earned * 100 / possible
}

// topicList is the list of shell topics.
type topicList struct {
	progress *progress.Progress
	cursor   int
	note     string
}

// newTopicList puts the cursor on the newest topic the player can open.
func newTopicList(p *progress.Progress) topicList {
	l := topicList{progress: p}
	for i := range drills.Shell {
		if topicOpen(p, i) {
			l.cursor = i
		}
	}
	return l
}

func (l topicList) update(msg tea.Msg) (screen, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return l, nil
	}
	l.note = ""
	switch key.String() {
	case "up", "k":
		l.cursor = max(0, l.cursor-1)
	case "down", "j":
		l.cursor = min(len(drills.Shell)-1, l.cursor+1)
	case "enter", "space":
		t := drills.Shell[l.cursor]
		switch {
		case len(t.Drills) == 0:
			l.note = t.Title + " is coming later."
		case !topicOpen(l.progress, l.cursor):
			l.note = "Pass the " + drills.Shell[l.cursor-1].Title + " exam first."
		default:
			return l, show(showTopic{l.cursor})
		}
	case "esc":
		return l, show(showMenu{})
	case "q", "ctrl+c":
		return l, tea.Quit
	}
	return l, nil
}

func (l topicList) view(width, height int) string {
	w := pageWidth(width)
	var b strings.Builder
	b.WriteString(spread(title.Render("Shell"), faint.Render(fmt.Sprintf("skill %d%%", skill(l.progress))), w) + "\n\n")
	for i, t := range drills.Shell {
		status := "coming later"
		open := topicOpen(l.progress, i)
		switch {
		case open:
			solved := 0
			for _, d := range t.Drills {
				if l.progress.Solved(key(t, d)) {
					solved++
				}
			}
			status = fmt.Sprintf("%d/%d", solved, len(t.Drills))
		case len(t.Drills) > 0:
			status = "locked"
		}
		head := fmt.Sprintf("%2d  %-25s", i+1, t.Title)
		rest := fmt.Sprintf("%-22s%*s ", t.Commands, w-2-lipgloss.Width(head)-22-1, status)
		b.WriteString(listRow(head, rest, w, i == l.cursor, !open) + "\n")
	}
	b.WriteString("\n" + faint.Render(l.note) + "\n\n")
	b.WriteString(hints(hint("move", "j", "k"), hint("open", "enter"), hint("back", "esc")))
	return b.String()
}

// topicPage is one topic: its briefing, then its drills.
type topicPage struct {
	progress *progress.Progress
	topic    int
	cursor   int
	note     string
}

// newTopicPage puts the cursor on the first drill not yet solved.
func newTopicPage(p *progress.Progress, topic int) topicPage {
	page := topicPage{progress: p, topic: topic}
	t := drills.Shell[topic]
	for i, d := range t.Drills {
		if !p.Solved(key(t, d)) {
			page.cursor = i
			break
		}
	}
	return page
}

func (p topicPage) update(msg tea.Msg) (screen, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}
	t := drills.Shell[p.topic]
	p.note = ""
	switch key.String() {
	case "up", "k":
		p.cursor = max(0, p.cursor-1)
	case "down", "j":
		p.cursor = min(len(t.Drills)-1, p.cursor+1)
	case "enter", "space":
		if !drillOpen(p.progress, p.topic, p.cursor) {
			p.note = "Solve " + t.Drills[p.cursor-1].Title + " first."
			return p, nil
		}
		return p, show(showDrill{p.topic, p.cursor})
	case "esc":
		return p, show(showTopics{})
	case "q", "ctrl+c":
		return p, tea.Quit
	}
	return p, nil
}

func (p topicPage) view(width, height int) string {
	w := pageWidth(width)
	t := drills.Shell[p.topic]
	var b strings.Builder
	brief := briefing(p.topic, w)
	b.WriteString(brief + "\n\n")
	b.WriteString(faint.Render("Drills") + "\n")

	// Show as many drills as fit under the briefing, keeping the cursor in view.
	if height == 0 {
		height = 40
	}
	fits := max(3, height-strings.Count(brief, "\n")-12)
	start := min(max(0, p.cursor-fits/2), max(0, len(t.Drills)-fits))
	end := min(len(t.Drills), start+fits)
	b.WriteString(more("↑", start) + "\n")
	for i := start; i < end; i++ {
		d := t.Drills[i]
		open := drillOpen(p.progress, p.topic, i)
		status := "locked"
		if open {
			status = stars(p.progress.Drills[key(t, d)].Stars)
		}
		head := fmt.Sprintf("%d  %-20s", i+1, d.Title)
		rest := fmt.Sprintf("%*s ", w-2-lipgloss.Width(head)-1, status)
		b.WriteString(listRow(head, rest, w, i == p.cursor, !open) + "\n")
	}
	b.WriteString(more("↓", len(t.Drills)-end) + "\n")
	b.WriteString("\n" + faint.Render(p.note) + "\n\n")
	b.WriteString(hints(hint("move", "j", "k"), hint("start", "enter"), hint("back", "esc")))
	return b.String()
}

// more says how many list rows are scrolled out of view, or nothing.
func more(arrow string, n int) string {
	if n == 0 {
		return ""
	}
	return faint.Render(fmt.Sprintf("  %s %d more", arrow, n))
}

// briefing renders a topic's lesson: what its commands do, with examples.
func briefing(topic, width int) string {
	t := drills.Shell[topic]
	var b strings.Builder
	header := fmt.Sprintf("%d · %s", topic+1, t.Title)
	b.WriteString(spread(title.Render(header), faint.Render(fmt.Sprintf("%d drills", len(t.Drills))), width) + "\n\n")
	b.WriteString(lipgloss.NewStyle().Width(width).Render(t.Summary) + "\n\n")
	col := 0
	for _, u := range t.Briefing {
		col = max(col, lipgloss.Width(u.Example))
	}
	for _, u := range t.Briefing {
		fmt.Fprintf(&b, "  %-*s   %s\n", col, u.Example, faint.Render(u.Does))
	}
	if t.Note != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Width(width).Render(t.Note))
	}
	return strings.TrimRight(b.String(), "\n")
}
