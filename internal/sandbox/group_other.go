//go:build !unix

package sandbox

import "os/exec"

// Shell drills need a Unix shell; these keep the package building elsewhere.
func inGroup(cmd *exec.Cmd)         {}
func killGroup(cmd *exec.Cmd) error { return nil }
