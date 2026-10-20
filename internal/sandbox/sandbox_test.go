package sandbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newSandbox(t *testing.T, start Files) *Sandbox {
	t.Helper()
	s, err := New(start)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func run(t *testing.T, s *Sandbox, command string) Result {
	t.Helper()
	res, err := s.Run(command)
	if err != nil {
		t.Fatalf("Run(%q): %v", command, err)
	}
	return res
}

func files(t *testing.T, s *Sandbox) Files {
	t.Helper()
	got, err := s.Files()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// shells lists the sh-style shells installed here, so behaviour is checked
// in each one a player might have as $SHELL.
func shells(t *testing.T) []string {
	var found []string
	for _, sh := range []string{"/bin/sh", "/bin/bash", "/bin/zsh", "/bin/dash"} {
		if _, err := os.Stat(sh); err == nil {
			found = append(found, sh)
		}
	}
	if len(found) == 0 {
		t.Skip("no sh-style shell installed")
	}
	return found
}

func TestNewWritesStartFiles(t *testing.T) {
	start := Files{"notes.txt": "hello", "empty/": "", "src/main.go": "package main"}
	s := newSandbox(t, start)
	if got := files(t, s); !got.Equal(start) {
		t.Errorf("Files() = %v, want %v", got, start)
	}
	if s.Where() != "~" {
		t.Errorf("Where() = %q, want ~", s.Where())
	}
}

func TestNewRejectsPathsOutside(t *testing.T) {
	for _, path := range []string{"../escape.txt", "/tmp/escape.txt", "a/../../escape.txt", ""} {
		if s, err := New(Files{path: "x"}); err == nil {
			s.Close()
			t.Errorf("New accepted %q", path)
		}
	}
}

func TestRun(t *testing.T) {
	start := Files{"a.txt": "a", "docs/guide.md": "guide", "docs/old/": ""}
	tests := []struct {
		name     string
		commands []string
		where    string
		files    Files // nil means unchanged
	}{
		{"mv renames", []string{"mv a.txt b.txt"}, "~",
			Files{"b.txt": "a", "docs/guide.md": "guide", "docs/old/": ""}},
		{"cd stays in effect", []string{"cd docs"}, "~/docs", nil},
		{"cd .. goes up", []string{"cd docs/old", "cd .."}, "~/docs", nil},
		{"cd - goes back", []string{"cd docs", "cd old", "cd -"}, "~/docs", nil},
		{"cd alone goes home", []string{"cd docs/old", "cd"}, "~", nil},
		{"~ is the sandbox", []string{"cd docs", "cd ~"}, "~", nil},
		{"commands run in the current folder", []string{"cd docs", "touch new.txt"}, "~/docs",
			Files{"a.txt": "a", "docs/guide.md": "guide", "docs/new.txt": "", "docs/old/": ""}},
		{"deleting the current folder moves you home", []string{"cd docs/old", "rmdir ../old"}, "~",
			Files{"a.txt": "a", "docs/guide.md": "guide"}},
		{"a failed cd stays put", []string{"cd docs", "cd missing"}, "~/docs", nil},
	}
	for _, sh := range shells(t) {
		for _, tt := range tests {
			t.Run(filepath.Base(sh)+"/"+tt.name, func(t *testing.T) {
				t.Setenv("SHELL", sh)
				s := newSandbox(t, start)
				for _, c := range tt.commands {
					run(t, s, c)
				}
				if got := s.Where(); got != tt.where {
					t.Errorf("Where() = %q, want %q", got, tt.where)
				}
				want := tt.files
				if want == nil {
					want = start
				}
				if got := files(t, s); !got.Equal(want) {
					t.Errorf("Files() = %v, want %v", got, want)
				}
			})
		}
	}
}

func TestRunResult(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	s := newSandbox(t, nil)

	if res := run(t, s, "echo hi"); res.Output != "hi\n" || res.ExitCode != 0 {
		t.Errorf("echo hi: output %q, exit %d", res.Output, res.ExitCode)
	}
	if res := run(t, s, "ls missing"); res.ExitCode == 0 || res.Output == "" {
		t.Errorf("ls missing: exit %d, output %q; want an error", res.ExitCode, res.Output)
	}
	if res := run(t, s, "mkdir docs && cd docs && pwd"); res.Output != "~/docs\n" {
		t.Errorf("pwd printed %q, want the sandbox shown as ~", res.Output)
	}
	if res := run(t, s, "exit 3"); res.ExitCode != 3 {
		t.Errorf("exit 3: exit %d", res.ExitCode)
	}
	if res := run(t, s, "cd /"); !res.Escaped || s.Where() != "~" {
		t.Errorf("cd /: escaped %v, where %q; want moved back to ~", res.Escaped, s.Where())
	}
}

func TestRunLimits(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	defer func(d time.Duration) { timeout = d }(timeout)
	timeout = 300 * time.Millisecond
	s := newSandbox(t, nil)

	if res := run(t, s, "sleep 5"); !res.TimedOut {
		t.Error("sleep 5 did not time out")
	}
	if res := run(t, s, "yes"); len(res.Output) != maxOutput {
		t.Errorf("yes kept %d bytes of output, want %d", len(res.Output), maxOutput)
	}
}

func TestRunDoesNotWaitForBackgroundJobs(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	s := newSandbox(t, nil)
	start := time.Now()
	if res := run(t, s, "sleep 2 &"); res.TimedOut {
		t.Error("a background job counted as timed out")
	}
	if waited := time.Since(start); waited > 1500*time.Millisecond {
		t.Errorf("waited %v for a background job", waited)
	}
}

func TestRunStopsLeftoverJobs(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	s := newSandbox(t, nil)
	run(t, s, "(while :; do echo x >> log.txt; sleep 0.05; done) >/dev/null 2>&1 &")

	size := func() int64 {
		fi, err := os.Stat(filepath.Join(s.home, "log.txt"))
		if err != nil {
			return 0
		}
		return fi.Size()
	}
	before := size()
	time.Sleep(300 * time.Millisecond)
	if after := size(); after != before {
		t.Errorf("a background job kept running: log.txt grew from %d to %d bytes", before, after)
	}
}

func TestRunCapsFileSize(t *testing.T) {
	limit := int64(20000 * 1024) // ulimit blocks are 512 or 1024 bytes, depending on the shell
	for _, sh := range shells(t) {
		t.Run(filepath.Base(sh), func(t *testing.T) {
			t.Setenv("SHELL", sh)
			s := newSandbox(t, nil)
			// head stops at 30 MB on its own, so this stays bounded even if the cap failed.
			run(t, s, "head -c 30000000 /dev/zero > big.txt")
			fi, err := os.Stat(filepath.Join(s.home, "big.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if fi.Size() > limit {
				t.Errorf("big.txt is %d bytes, want at most %d", fi.Size(), limit)
			}
		})
	}
}

func TestRunRecoversDeletedHome(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	s := newSandbox(t, Files{"docs/a.txt": "a"})
	run(t, s, "cd docs")
	if err := os.RemoveAll(s.home); err != nil {
		t.Fatal(err)
	}
	run(t, s, "true")
	if s.Where() != "~" {
		t.Errorf("Where() = %q, want ~", s.Where())
	}
	if got := files(t, s); len(got) != 0 {
		t.Errorf("Files() = %v, want empty", got)
	}
}

func TestFilesNeverFollowsSymlinks(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newSandbox(t, nil)
	if err := os.Symlink(secret, filepath.Join(s.home, "link")); err != nil {
		t.Fatal(err)
	}
	if got := files(t, s); !got.Equal(Files{"link": ""}) {
		t.Errorf("Files() = %v, want the link listed but not read", got)
	}
}

func TestCloseDeletesSandbox(t *testing.T) {
	s, err := New(Files{"a.txt": "a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.base); !os.IsNotExist(err) {
		t.Errorf("sandbox still exists after Close: %v", err)
	}
}

func TestCloseRefusesOtherFolders(t *testing.T) {
	for _, dir := range []string{
		t.TempDir(),                              // in temp, but not a keydrill folder
		filepath.Join(t.TempDir(), "keydrill-1"), // keydrill name, wrong place
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		s := &Sandbox{base: dir}
		if err := s.Close(); err == nil {
			t.Errorf("Close agreed to delete %s", dir)
		}
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("Close deleted %s", dir)
		}
	}
}

func TestFilesEqual(t *testing.T) {
	tests := []struct {
		a, b Files
		want bool
	}{
		{Files{"count.txt": "3"}, Files{"count.txt": "   3\n"}, true},
		{Files{"a.txt": "a"}, Files{"a.txt": "b"}, false},
		{Files{"a.txt": "a"}, Files{"a.txt": "a", "b.txt": ""}, false},
		{Files{"docs/": ""}, Files{"docs": ""}, false},
		{Files{}, Files{}, true},
	}
	for _, tt := range tests {
		if got := tt.a.Equal(tt.b); got != tt.want {
			t.Errorf("%v.Equal(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
