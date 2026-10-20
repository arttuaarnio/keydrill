package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/arttuaarnio/keydrill/internal/drills"
	"github.com/arttuaarnio/keydrill/internal/progress"
)

func newProgress(t *testing.T) *progress.Progress {
	t.Helper()
	p, err := progress.Load(filepath.Join(t.TempDir(), "progress.json"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// startDrill opens the first Moving around drill, "Get into {a}" (par 1), with random names.
func startDrill(t *testing.T, p *progress.Progress) drillPage {
	t.Helper()
	t.Setenv("SHELL", "/bin/sh")
	m, err := newDrillPage(p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func send(t *testing.T, m drillPage, msg tea.Msg) (drillPage, tea.Cmd) {
	t.Helper()
	s, cmd := m.update(msg)
	return s.(drillPage), cmd
}

// typeLine types a line and presses enter, running the command if there is one.
func typeLine(t *testing.T, m drillPage, line string) drillPage {
	t.Helper()
	m.input.SetValue(line)
	m, cmd := send(t, m, keyEnter)
	if cmd != nil {
		if msg, ok := cmd().(ranMsg); ok {
			m, _ = send(t, m, msg)
		}
	}
	return m
}

func TestScore(t *testing.T) {
	tests := []struct {
		commands, par int
		hinted        bool
		want          int
	}{
		{1, 2, false, 3},
		{2, 2, false, 3},
		{3, 2, false, 2},
		{4, 2, false, 2},
		{5, 2, false, 1},
		{2, 2, true, 2},
		{9, 2, true, 1},
	}
	for _, tt := range tests {
		if got, _ := score(tt.commands, tt.par, tt.hinted); got != tt.want {
			t.Errorf("score(%d, par %d, hinted %v) = %d stars, want %d", tt.commands, tt.par, tt.hinted, got, tt.want)
		}
	}
}

func TestDrillSolvedAtPar(t *testing.T) {
	p := newProgress(t)
	m := startDrill(t, p)
	defer m.close()

	m = typeLine(t, m, m.d.Solution[0])
	if !m.solved || m.stars != 3 || m.commands != 1 {
		t.Fatalf("solved %v, stars %d, commands %d; want solved at par", m.solved, m.stars, m.commands)
	}
	if got := p.Drills["moving-around/into-a-folder"]; got != (progress.Result{Stars: 3, Commands: 1}) {
		t.Errorf("saved %v, want 3 stars in 1 command", got)
	}
}

func TestDrillEveryCommandCounts(t *testing.T) {
	m := startDrill(t, newProgress(t))
	defer m.close()

	for _, line := range []string{"ls", "pwd", "ls -a", m.d.Solution[0]} {
		m = typeLine(t, m, line)
	}
	if !m.solved || m.commands != 4 || m.stars != 1 {
		t.Errorf("solved %v in %d commands with %d stars; want 4 commands, 1 star", m.solved, m.commands, m.stars)
	}
}

func TestDrillHintCapsStars(t *testing.T) {
	m := startDrill(t, newProgress(t))
	defer m.close()

	m = typeLine(t, m, "hint")
	m = typeLine(t, m, m.d.Solution[0])
	if m.stars != 2 {
		t.Errorf("stars = %d after a hint, want 2", m.stars)
	}
}

func TestDrillControlsDontCount(t *testing.T) {
	m := startDrill(t, newProgress(t))
	defer m.close()

	m = typeLine(t, m, "help")
	if !m.help {
		t.Fatal("help didn't open the briefing")
	}
	m, _ = send(t, m, keyJ)
	if m.help {
		t.Fatal("a key press didn't close the briefing")
	}
	m = typeLine(t, m, "clear")
	if m.commands != 0 || len(m.log) != 0 {
		t.Errorf("after help and clear: commands %d, log %q; want neither counted and the log empty", m.commands, m.log)
	}
}

func TestDrillCtrlCClearsTheLine(t *testing.T) {
	m := startDrill(t, newProgress(t))
	defer m.close()

	m.input.SetValue("cd pro")
	m, cmd := send(t, m, keyCtrlC)
	if isQuit(cmd) || m.input.Value() != "" {
		t.Errorf("ctrl+c: quit %v, input %q; want the line cleared and no quit", isQuit(cmd), m.input.Value())
	}
}

func TestDrillHistory(t *testing.T) {
	m := startDrill(t, newProgress(t))
	defer m.close()

	m = typeLine(t, m, "ls")
	m = typeLine(t, m, "pwd")
	for _, step := range []struct {
		key  tea.KeyPressMsg
		want string
	}{{keyUp, "pwd"}, {keyUp, "ls"}, {keyUp, "ls"}, {keyDown, "pwd"}, {keyDown, ""}} {
		m, _ = send(t, m, step.key)
		if got := m.input.Value(); got != step.want {
			t.Fatalf("input = %q, want %q", got, step.want)
		}
	}
}

func TestDrillRetryStartsOver(t *testing.T) {
	m := startDrill(t, newProgress(t))
	m = typeLine(t, m, m.d.Solution[0])
	m, _ = send(t, m, keyR)
	defer m.close()
	if m.solved || m.commands != 0 || m.where != "~" {
		t.Errorf("after retry: solved %v, commands %d, where %q; want a fresh start", m.solved, m.commands, m.where)
	}
}

func TestAppCloseDeletesTheSandbox(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	var a tea.Model = NewApp(newProgress(t), "")
	a, _ = a.Update(showDrill{0, 0})
	d, ok := a.(App).screen.(drillPage)
	if !ok {
		t.Fatalf("screen is %T, want a drill", a.(App).screen)
	}
	a.(App).Close()
	if _, err := d.box.Files(); err == nil {
		t.Error("sandbox still exists after Close")
	}
}

func TestTopicPageScrollsToTheCursor(t *testing.T) {
	page := newTopicPage(newProgress(t), 0)
	last := len(drills.Shell[0].Drills) - 1
	page.cursor = last
	view := page.view(80, 30)
	if !strings.Contains(view, drills.Shell[0].Drills[last].Title) || !strings.Contains(view, "↑") || strings.Contains(view, "↓") {
		t.Errorf("with the cursor on the last drill, want it shown with only an ↑ marker:\n%s", view)
	}
}

func TestUnlocks(t *testing.T) {
	p := newProgress(t)
	topic := drills.Shell[0]
	if !topicOpen(p, 0) || topicOpen(p, 1) {
		t.Errorf("topic 1 open %v, topic 2 open %v; want only topic 1", topicOpen(p, 0), topicOpen(p, 1))
	}
	if !drillOpen(p, 0, 0) || drillOpen(p, 0, 1) {
		t.Error("only the first drill should be open at the start")
	}
	if err := p.Record(key(topic, topic.Drills[0]), 3, 1); err != nil {
		t.Fatal(err)
	}
	if !drillOpen(p, 0, 1) || drillOpen(p, 0, 2) {
		t.Error("solving drill 1 should open drill 2 and only drill 2")
	}
	if got, want := skill(p), 300/(3*len(topic.Drills)); got != want {
		t.Errorf("skill = %d%%, want %d%%", got, want)
	}
}
