package progress

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func load(t *testing.T, path string) *Progress {
	t.Helper()
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadMissingFileStartsFresh(t *testing.T) {
	p := load(t, filepath.Join(t.TempDir(), "keydrill", "progress.json"))
	if len(p.Drills) != 0 || p.Solved("moving-around/exam") {
		t.Errorf("fresh progress has drills: %v", p.Drills)
	}
}

func TestRecordKeepsTheBest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keydrill", "progress.json")
	p := load(t, path)

	steps := []struct {
		stars, commands int
		want            Result
	}{
		{2, 4, Result{2, 4}},
		{1, 9, Result{2, 4}}, // fewer stars: ignored
		{2, 3, Result{2, 3}}, // same stars, fewer commands
		{2, 5, Result{2, 3}}, // same stars, more commands: ignored
		{3, 6, Result{3, 6}}, // more stars wins even with more commands
	}
	for _, s := range steps {
		if err := p.Record("mv/rename", s.stars, s.commands); err != nil {
			t.Fatal(err)
		}
		// Reload each time to check what actually reached the disk.
		if got := load(t, path).Drills["mv/rename"]; got != s.want {
			t.Errorf("after Record(%d, %d): saved %v, want %v", s.stars, s.commands, got, s.want)
		}
	}
}

func TestLoadCorruptFileKeepsABackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Load error = %v, want ErrCorrupt", err)
	}
	if p == nil || len(p.Drills) != 0 {
		t.Fatalf("want fresh progress, got %v", p)
	}
	if b, err := os.ReadFile(path + ".bak"); err != nil || string(b) != "{not json" {
		t.Errorf("backup = %q, %v; want the original contents", b, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("corrupt file still in place: %v", err)
	}
}
