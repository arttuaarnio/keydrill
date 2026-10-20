// Package sandbox runs shell drills in a throwaway folder, so a drill never
// touches the player's real files.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Limits for a single command. They are variables so tests can shorten them.
var (
	timeout   = 5 * time.Second
	maxOutput = 64 << 10 // bytes of output kept per command
	maxRead   = 1 << 20  // bytes of each file that Files reads
)

// Files describes a folder tree as path → content, using forward slashes.
// A path ending in "/" is an empty folder; other folders are implied by the
// files inside them.
type Files map[string]string

// Equal reports whether two trees hold the same paths. File contents are
// compared without leading or trailing whitespace, so "3" matches "   3\n".
func (f Files) Equal(g Files) bool {
	if len(f) != len(g) {
		return false
	}
	for path, content := range f {
		other, ok := g[path]
		if !ok || strings.TrimSpace(content) != strings.TrimSpace(other) {
			return false
		}
	}
	return true
}

// Result is what one command did.
type Result struct {
	Output   string // stdout and stderr together, cut off after maxOutput bytes, with the sandbox path shown as ~
	ExitCode int
	TimedOut bool
	Escaped  bool // the command ended outside the sandbox, so we moved back to ~
}

// Sandbox is a temporary folder that acts as the player's home folder.
type Sandbox struct {
	base string // the keydrill-* temp folder; holds home and the pwd file
	home string // the player's ~
	dir  string // current folder, always inside home
	prev string // previous folder, for cd -
}

// New creates a sandbox holding the start files.
func New(start Files) (*Sandbox, error) {
	tmp, err := os.MkdirTemp("", "keydrill-")
	if err != nil {
		return nil, err
	}
	// Resolve symlinks (on macOS /var is /private/var) so paths match pwd -P.
	base, err := filepath.EvalSymlinks(tmp)
	if err != nil {
		os.Remove(tmp)
		return nil, err
	}
	home := filepath.Join(base, "home")
	s := &Sandbox{base: base, home: home, dir: home, prev: home}
	if err := s.write(start); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Sandbox) write(files Files) error {
	if err := os.MkdirAll(s.home, 0o755); err != nil {
		return err
	}
	for path, content := range files {
		local := filepath.FromSlash(path)
		if !filepath.IsLocal(local) {
			return fmt.Errorf("sandbox: %q is outside the sandbox", path)
		}
		full := filepath.Join(s.home, local)
		if strings.HasSuffix(path, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Run runs one command line in the player's shell, from the current folder.
// A failing command is not an error: err is only for failing to run at all.
func (s *Sandbox) Run(command string) (Result, error) {
	if err := s.restore(); err != nil {
		return Result{}, err
	}
	pwdFile := filepath.Join(s.base, "pwd")
	if err := os.WriteFile(pwdFile, nil, 0o600); err != nil {
		return Result{}, err
	}

	// Cap file size (10–20 MB, depending on the shell's block size) so a
	// runaway command can't fill the disk. Step into the previous folder and
	// back so cd - works in every shell, then run the command and record
	// where it ended, keeping its exit code.
	script := `ulimit -f 20000 2>/dev/null; cd -- "$KEYDRILL_PREV" 2>/dev/null; cd -- "$KEYDRILL_DIR"` + "\n" +
		command + "\n" +
		`__keydrill=$?; pwd -P > "$KEYDRILL_PWD"; exit $__keydrill`

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell(), "-c", script)
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(), // later entries win
		"HOME="+s.home,
		"PWD="+s.dir,
		"ZDOTDIR="+s.home, // keep the player's own zsh config out
		"BASH_ENV=", "ENV=", "CDPATH=",
		"KEYDRILL_PREV="+s.prev, "KEYDRILL_DIR="+s.dir, "KEYDRILL_PWD="+pwdFile,
	)
	out := &capped{max: maxOutput}
	cmd.Stdout, cmd.Stderr = out, out
	cmd.WaitDelay = time.Second // don't wait long on background jobs holding the output open
	inGroup(cmd)

	err := cmd.Run()
	killGroup(cmd) // ESRCH just means nothing was left running
	// The sandbox is the player's home, so show its long temp path as ~.
	res := Result{
		Output:   strings.ReplaceAll(out.buf.String(), s.home, "~"),
		TimedOut: ctx.Err() == context.DeadlineExceeded,
	}
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	case err != nil && !res.TimedOut && !errors.Is(err, exec.ErrWaitDelay):
		return res, err
	}

	// An empty pwd file means the command exited early; stay where we were.
	if b, err := os.ReadFile(pwdFile); err == nil && len(b) > 0 {
		dir := strings.TrimSuffix(string(b), "\n")
		switch {
		case !s.contains(dir):
			res.Escaped = true
			s.prev, s.dir = s.dir, s.home
		case dir != s.dir:
			s.prev, s.dir = s.dir, dir
		}
	}

	return res, s.restore()
}

// restore brings back ~ and steps out of a deleted folder, since the player
// can delete either.
func (s *Sandbox) restore() error {
	if err := os.MkdirAll(s.home, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(s.dir); err != nil {
		s.dir = s.home
	}
	return nil
}

func (s *Sandbox) contains(path string) bool {
	rel, err := filepath.Rel(s.home, path)
	return err == nil && filepath.IsLocal(rel)
}

// Where returns the current folder the way a prompt shows it: "~" or "~/docs".
func (s *Sandbox) Where() string {
	rel, err := filepath.Rel(s.home, s.dir)
	if err != nil || rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}

// Files lists everything in the sandbox. It never follows symlinks, so it
// can't read anything outside, and it reads at most maxRead bytes per file.
func (s *Sandbox) Files() (Files, error) {
	files := Files{}
	err := filepath.WalkDir(s.home, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == s.home {
			return err
		}
		rel, err := filepath.Rel(s.home, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		switch {
		case d.IsDir():
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				files[rel+"/"] = ""
			}
		case d.Type().IsRegular():
			content, err := readFile(path)
			if err != nil {
				return err
			}
			files[rel] = content
		default:
			files[rel] = "" // symlinks and other special files are listed, never read
		}
		return nil
	})
	return files, err
}

func readFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, int64(maxRead)))
	return string(b), err
}

// Close deletes the sandbox. As a guard, it only ever deletes a keydrill-*
// folder directly inside the system temp folder.
func (s *Sandbox) Close() error {
	tmp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return err
	}
	if filepath.Dir(s.base) != tmp || !strings.HasPrefix(filepath.Base(s.base), "keydrill-") {
		return fmt.Errorf("sandbox: refusing to delete %s", s.base)
	}
	return os.RemoveAll(s.base)
}

// shell returns the player's shell if it understands sh syntax, which the
// script in Run needs. Others, like fish, fall back to /bin/sh.
func shell() string {
	sh := os.Getenv("SHELL")
	if slices.Contains([]string{"sh", "bash", "zsh", "dash", "ksh"}, filepath.Base(sh)) {
		return sh
	}
	return "/bin/sh"
}

// capped keeps the first max bytes written to it and drops the rest, so a
// runaway command like `yes` can't fill memory.
type capped struct {
	buf bytes.Buffer
	max int
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - c.buf.Len(); room > 0 {
		c.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}
