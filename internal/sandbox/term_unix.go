//go:build unix

package sandbox

import (
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// execute runs cmd with its output on a pseudo-terminal, so programs format it
// as they would in a real terminal (ls prints columns, for example). Input
// stays closed, so a program waiting for input ends at once. cmd runs in a
// session of its own, and everything left in that session is killed when
// it finishes or times out, so background jobs can't outlive it.
func execute(cmd *exec.Cmd, out io.Writer, cols int) error {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return err
	}
	defer ptmx.Close()
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: uint16(cols)}); err != nil {
		tty.Close()
		return err
	}

	cmd.Stdout, cmd.Stderr = tty, tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 1} // Ctty is stdout
	kill := func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.Cancel = kill
	err = cmd.Start()
	tty.Close() // the command holds its own copy
	if err != nil {
		return err
	}

	copied := make(chan struct{})
	go func() {
		io.Copy(out, ptmx) // ends when nothing holds the terminal open any more
		close(copied)
	}()
	err = cmd.Wait()
	kill() // ESRCH just means nothing was left running
	select {
	case <-copied:
	case <-time.After(time.Second):
	}
	return err
}
