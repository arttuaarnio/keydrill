// Package progress saves the player's best result for each drill.
package progress

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrCorrupt means the save file couldn't be read. Load moves it aside to
// progress.json.bak, never deleting it, and starts fresh.
var ErrCorrupt = errors.New("progress file was unreadable")

// Result is the best a player has done on one drill.
type Result struct {
	Stars    int `json:"stars"`
	Commands int `json:"commands"`
}

// Progress maps drill keys ("topic/drill") to the player's best result.
type Progress struct {
	Drills map[string]Result `json:"drills"`
	path   string
}

// DefaultPath is where progress is saved, for example
// ~/Library/Application Support/keydrill/progress.json on macOS.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "keydrill", "progress.json"), nil
}

// Load reads progress from path. A missing file is a fresh start. An
// unreadable one is moved to path+".bak" and Load returns fresh progress
// together with ErrCorrupt, so the player can be told.
func Load(path string) (*Progress, error) {
	p := &Progress{Drills: map[string]Result{}, path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, p); err != nil || p.Drills == nil {
		p.Drills = map[string]Result{}
		if err := os.Rename(path, path+".bak"); err != nil {
			return nil, err
		}
		return p, fmt.Errorf("%w; moved it to %s.bak", ErrCorrupt, filepath.Base(path))
	}
	return p, nil
}

// Solved reports whether the drill has been solved at least once.
func (p *Progress) Solved(key string) bool {
	_, ok := p.Drills[key]
	return ok
}

// Record saves a solve if it beats the best so far: more stars, or the
// same stars in fewer commands.
func (p *Progress) Record(key string, stars, commands int) error {
	old, ok := p.Drills[key]
	if ok && (old.Stars > stars || old.Stars == stars && old.Commands <= commands) {
		return nil
	}
	p.Drills[key] = Result{Stars: stars, Commands: commands}
	return p.save()
}

// save writes to a temporary file first and renames it into place, so a
// crash halfway through can't leave a half-written save.
func (p *Progress) save() error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}
