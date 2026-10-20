//go:build unix

package sandbox

import (
	"os/exec"
	"syscall"
)

// inGroup runs cmd in its own process group, and makes a timeout kill the
// whole group rather than just the shell.
func inGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd) }
}

// killGroup kills everything left in cmd's process group, such as
// background jobs, so nothing outlives the command.
func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
