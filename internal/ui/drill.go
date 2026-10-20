package ui

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/arttuaarnio/keydrill/internal/drills"
	"github.com/arttuaarnio/keydrill/internal/progress"
	"github.com/arttuaarnio/keydrill/internal/sandbox"
)

// drillPage is one drill: a prompt where the player runs real commands in a
// sandbox until they reach the goal.
type drillPage struct {
	progress *progress.Progress
	topic    int
	index    int
	t        drills.Topic
	d        drills.Drill
	box      *sandbox.Sandbox

	input    textinput.Model
	log      []string // everything shown above the prompt, oldest first
	where    string   // the sandbox's current folder, cached for drawing
	history  []string // commands typed, for up and down
	histPos  int
	commands int
	hinted   bool // saw the hint during this visit, which caps the stars
	running  bool
	solved   bool
	stars    int
	verdict  string
	help     bool // showing the briefing over the drill
}

// ranMsg is a finished command. The sandbox is only touched by the command
// while it runs, so everything the page needs from it comes back here.
type ranMsg struct {
	res    sandbox.Result
	err    error
	where  string
	solved bool
}

func newDrillPage(p *progress.Progress, topic, index int) (drillPage, error) {
	t := drills.Shell[topic]
	d := t.Drills[index].Fill(rand.IntN)
	box, err := d.Sandbox()
	if err != nil {
		return drillPage{}, err
	}
	in := textinput.New()
	in.Prompt = ""
	styles := in.Styles()
	styles.Cursor.Color = mauve
	styles.Cursor.Blink = false
	in.SetStyles(styles)
	in.Focus()
	return drillPage{progress: p, topic: topic, index: index, t: t, d: d, box: box, input: in, where: box.Where()}, nil
}

func (m drillPage) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case ranMsg:
		return m.finish(msg), nil
	case tea.KeyPressMsg:
		switch {
		case m.help:
			m.help = false
			return m, nil
		case m.solved:
			return m.afterSolve(msg)
		case m.running:
			return m, nil
		}
		switch msg.String() {
		case "enter":
			return m.submit()
		case "esc":
			m.close()
			return m, show(showTopic{m.topic})
		case "ctrl+c": // like a shell: abandon the line, don't quit
			m.log = append(m.log, m.prompt()+m.input.Value()+faint.Render("^C"))
			m.input.Reset()
			return m, nil
		case "up":
			if m.histPos > 0 {
				m.histPos--
				m.input.SetValue(m.history[m.histPos])
				m.input.CursorEnd()
			}
			return m, nil
		case "down":
			if m.histPos < len(m.history) {
				m.histPos++
				m.input.SetValue("")
				if m.histPos < len(m.history) {
					m.input.SetValue(m.history[m.histPos])
				}
				m.input.CursorEnd()
			}
			return m, nil
		case "tab":
			return m, nil // shortcut: no tab completion yet; add it when drills get longer paths
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// submit handles the typed line: a game control, or a command to run.
func (m drillPage) submit() (screen, tea.Cmd) {
	line := strings.TrimSpace(m.input.Value())
	m.input.Reset()
	m.log = append(m.log, m.prompt()+line)
	if line == "" {
		return m, nil
	}
	m.history = append(m.history, line)
	m.histPos = len(m.history)

	switch line {
	case "help":
		m.help = true
	case "hint":
		m.hinted = true
		m.log = append(m.log, faint.Render("One way: "+strings.Join(m.d.Solution, ", then ")))
	case "reset":
		return m.reset(), nil
	case "clear":
		m.log = nil
	case "exit":
		m.close()
		return m, show(showTopic{m.topic})
	default:
		m.commands++
		m.running = true
		box, d := m.box, m.d
		return m, func() tea.Msg {
			res, err := box.Run(line)
			return ranMsg{res: res, err: err, where: box.Where(), solved: err == nil && d.Solved(box)}
		}
	}
	return m, nil
}

func (m drillPage) finish(msg ranMsg) drillPage {
	m.running = false
	m.where = msg.where
	if out := strings.TrimRight(msg.res.Output, "\n"); out != "" {
		m.log = append(m.log, out)
	}
	switch {
	case msg.err != nil:
		m.log = append(m.log, faint.Render("Couldn't run that: "+msg.err.Error()))
	case msg.res.TimedOut:
		m.log = append(m.log, faint.Render("Stopped: it ran too long. Interactive programs like vim and less don't work here."))
	case msg.res.Escaped:
		m.log = append(m.log, faint.Render("That's outside the sandbox. Back to ~."))
	}
	if !msg.solved {
		return m
	}

	m.solved = true
	m.input.Blur()
	m.stars, m.verdict = score(m.commands, m.d.Par(), m.hinted)
	if m.index == len(m.t.Drills)-1 {
		m.verdict += " Topic complete."
	}
	if err := m.progress.Record(key(m.t, m.d), m.stars, m.commands); err != nil {
		m.log = append(m.log, faint.Render("Couldn't save progress: "+err.Error()))
	}
	return m
}

func (m drillPage) afterSolve(msg tea.KeyPressMsg) (screen, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.close()
		if m.index+1 < len(m.t.Drills) {
			return m, show(showDrill{m.topic, m.index + 1})
		}
		return m, show(showTopics{})
	case "r": // a retry is a new puzzle: same drill, new names
		m.d = m.t.Drills[m.index].Fill(rand.IntN)
		return m.reset(), nil
	case "esc":
		m.close()
		return m, show(showTopic{m.topic})
	}
	return m, nil
}

// reset starts the drill over in a fresh sandbox with the same names. A hint
// seen this visit still counts.
func (m drillPage) reset() drillPage {
	box, err := m.d.Sandbox()
	if err != nil {
		m.log = append(m.log, faint.Render("Couldn't reset: "+err.Error()))
		return m
	}
	m.close()
	m.box, m.where = box, box.Where()
	m.log, m.commands, m.solved = nil, 0, false
	m.input.Focus()
	return m
}

func (m drillPage) close() {
	if m.box != nil {
		m.box.Close() // only fails if the guard refuses, and then there's nothing to clean
	}
}

func (m drillPage) prompt() string { return accent.Render(m.where) + faint.Render(" $ ") }

// score turns a solve into stars and a short verdict.
func score(commands, par int, hinted bool) (int, string) {
	stars, verdict := 1, "Run it again."
	switch {
	case commands < par:
		stars, verdict = 3, "Under par."
	case commands == par:
		stars, verdict = 3, "At par."
	case commands <= par+2:
		stars, verdict = 2, "Tighten it up."
	}
	if hinted && stars > 2 {
		return 2, "Solved with a hint. ★★ is the cap."
	}
	return stars, verdict
}

func (m drillPage) view(width, height int) string {
	w := pageWidth(width)
	if m.help {
		return briefing(m.topic, w) + "\n\n" + faint.Render("Press any key to get back to the drill.")
	}

	header := spread(title.Render(fmt.Sprintf("%s · %d of %d", m.t.Title, m.index+1, len(m.t.Drills))),
		faint.Render(fmt.Sprintf("par %d", m.d.Par())), w)
	top := header + "\n\n" + lipgloss.NewStyle().Width(w).Render(m.d.Task) + "\n"

	// The log scrolls like a terminal: the prompt follows the newest output.
	var area []string
	if len(m.log) > 0 {
		area = strings.Split(lipgloss.NewStyle().Width(w).Render(strings.Join(m.log, "\n")), "\n")
	}
	var footer string
	switch {
	case m.solved:
		result := fmt.Sprintf("Solved in %d. Par is %d. %s", m.commands, m.d.Par(), m.verdict)
		area = append(area, "", spread(result, accent.Render(stars(m.stars)), w))
		footer = hints(hint("next drill", "enter"), hint("retry", "r"), hint("topic", "esc"))
	case m.running:
		area = append(area, m.prompt()+faint.Render("running…"))
	default:
		in := m.input
		in.SetWidth(w - lipgloss.Width(m.prompt()) - 1)
		area = append(area, m.prompt()+in.View())
	}
	if footer == "" {
		footer = spread(faint.Render(fmt.Sprintf("commands %d", m.commands)),
			faint.Render("type help, hint or reset")+"   "+hint("back", "esc"), w)
	}

	// Keep the page a fixed height so it doesn't jump around as output arrives.
	areaHeight := max(5, min(18, height-strings.Count(top, "\n")-6))
	if len(area) > areaHeight {
		area = area[len(area)-areaHeight:]
	}
	for len(area) < areaHeight {
		area = append(area, "")
	}
	return top + "\n" + strings.Join(area, "\n") + "\n\n" + footer
}
