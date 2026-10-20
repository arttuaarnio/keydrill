// Keydrill is a terminal trainer for shell and Vim commands.
package main

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/arttuaarnio/keydrill/internal/progress"
	"github.com/arttuaarnio/keydrill/internal/ui"
)

func main() {
	path, err := progress.DefaultPath()
	if err != nil {
		fail(err)
	}
	p, err := progress.Load(path)
	var note string
	switch {
	case errors.Is(err, progress.ErrCorrupt):
		note = "Saved progress was unreadable, so you start fresh. The old file is progress.json.bak."
	case err != nil:
		fail(err)
	}

	final, err := tea.NewProgram(ui.NewApp(p, note)).Run()
	if app, ok := final.(ui.App); ok {
		app.Close()
	}
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "keydrill:", err)
	os.Exit(1)
}
