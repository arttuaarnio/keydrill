//go:build !unix

package sandbox

import (
	"io"
	"os/exec"
	"time"
)

// Shell drills need a Unix shell; this keeps the package building elsewhere.
func execute(cmd *exec.Cmd, out io.Writer, cols int) error {
	cmd.Stdout, cmd.Stderr = out, out
	cmd.WaitDelay = time.Second
	return cmd.Run()
}
