//go:build !windows

package core

import (
	"os"
	"os/exec"
)

func configureCommand(cmd *exec.Cmd) {}

// Xray is a single process. Development builds terminate the direct child only.
func containProcess(p *os.Process) (func(), error) { return func() {}, nil }
