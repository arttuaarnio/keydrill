// Package ui holds Keydrill's Bubble Tea screens.
package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/arttuaarnio/keydrill/internal/progress"
)

// A screen is one page of the app. App owns the window and switches screens
// when a screen returns one of the show messages below.
type screen interface {
	update(tea.Msg) (screen, tea.Cmd)
	view(width, height int) string
}

// Messages that switch screens.
type (
	showMenu   struct{}
	showTopics struct{}
	showTopic  struct{ topic int }
	showDrill  struct{ topic, drill int }
)

func show(msg tea.Msg) tea.Cmd { return func() tea.Msg { return msg } }

// App is the root model.
type App struct {
	screen        screen
	progress      *progress.Progress
	width, height int
}

// NewApp starts at the main menu, showing note under it if it isn't empty.
func NewApp(p *progress.Progress, note string) App {
	return App{screen: menu{note: note}, progress: p}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case showMenu:
		a.screen = menu{}
		return a, nil
	case showTopics:
		a.screen = newTopicList(a.progress)
		return a, nil
	case showTopic:
		a.screen = newTopicPage(a.progress, msg.topic)
		return a, nil
	case showDrill:
		d, err := newDrillPage(a.progress, msg.topic, msg.drill)
		if err != nil {
			page := newTopicPage(a.progress, msg.topic)
			page.note = "Couldn't start the drill: " + err.Error()
			a.screen = page
			return a, nil
		}
		a.screen = d
		return a, nil
	}
	var cmd tea.Cmd
	a.screen, cmd = a.screen.update(msg)
	return a, cmd
}

func (a App) View() tea.View {
	// Aligning pads every line to the same width, so Place centres the page
	// as one block instead of centring each line on its own.
	block := lipgloss.NewStyle().Align(lipgloss.Left).Render(a.screen.view(a.width, a.height))
	v := tea.NewView(lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, block))
	v.AltScreen = true
	return v
}

// Close cleans up after the app exits, such as deleting an open drill's sandbox.
func (a App) Close() {
	if d, ok := a.screen.(drillPage); ok {
		d.close()
	}
}
