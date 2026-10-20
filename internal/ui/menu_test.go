package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

var (
	keyJ     = tea.KeyPressMsg{Code: 'j', Text: "j"}
	keyK     = tea.KeyPressMsg{Code: 'k', Text: "k"}
	keyQ     = tea.KeyPressMsg{Code: 'q', Text: "q"}
	keyR     = tea.KeyPressMsg{Code: 'r', Text: "r"}
	keyDown  = tea.KeyPressMsg{Code: tea.KeyDown}
	keyUp    = tea.KeyPressMsg{Code: tea.KeyUp}
	keyEnter = tea.KeyPressMsg{Code: tea.KeyEnter}
	keyCtrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

// press sends keys to a fresh menu and returns it with the last command.
func press(keys ...tea.KeyPressMsg) (menu, tea.Cmd) {
	var s screen = menu{}
	var cmd tea.Cmd
	for _, k := range keys {
		s, cmd = s.update(k)
	}
	return s.(menu), cmd
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestMenuCursor(t *testing.T) {
	tests := []struct {
		name string
		keys []tea.KeyPressMsg
		want int
	}{
		{"starts at top", nil, 0},
		{"j moves down", []tea.KeyPressMsg{keyJ, keyJ}, 2},
		{"k moves up", []tea.KeyPressMsg{keyJ, keyJ, keyK}, 1},
		{"stops at top", []tea.KeyPressMsg{keyK}, 0},
		{"stops at bottom", []tea.KeyPressMsg{keyDown, keyDown, keyDown, keyDown, keyDown, keyDown}, len(menuItems) - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := press(tt.keys...)
			if m.cursor != tt.want {
				t.Errorf("cursor = %d, want %d", m.cursor, tt.want)
			}
		})
	}
}

func TestMenuSelect(t *testing.T) {
	if _, cmd := press(keyEnter); cmd == nil || cmd() != (showTopics{}) {
		t.Error("enter on Shell should open the topic list")
	}
	if m, _ := press(keyJ, keyEnter); m.note != "Vim motions is coming later." {
		t.Errorf("enter on Vim motions: note = %q, want it marked as coming later", m.note)
	}
	if _, cmd := press(keyQ); !isQuit(cmd) {
		t.Error("q should quit")
	}
	if _, cmd := press(keyJ, keyJ, keyJ, keyJ, keyEnter); !isQuit(cmd) {
		t.Error("enter on Quit should quit")
	}
}
