//go:build !windows

package core

import (
	"context"
	"os/exec"
	"path/filepath"
	"sync"
)

// Development adapter only. Windows uses atomic job membership at creation.
type execChild struct {
	cmd   *exec.Cmd
	drain outputDrain
	once  sync.Once
	err   error
}

func (c *execChild) Wait() error {
	c.once.Do(func() { c.err = c.cmd.Wait(); c.drain.wg.Wait() })
	return c.err
}
func (c *execChild) Kill() error           { return c.cmd.Process.Kill() }
func (c *execChild) Diagnostics() []string { return c.drain.Diagnostics() }
func startContained(ctx context.Context, path string, args []string) (childProcess, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = filepath.Dir(path)
	out, e := cmd.StdoutPipe()
	if e != nil {
		return nil, e
	}
	stderr, e := cmd.StderrPipe()
	if e != nil {
		out.Close()
		return nil, e
	}
	c := &execChild{cmd: cmd}
	if e = cmd.Start(); e != nil {
		out.Close()
		stderr.Close()
		return nil, e
	}
	c.drain.start(out, 0)
	c.drain.start(stderr, 1)
	return c, nil
}

func lockExecutable(string) (func(), error) { return func() {}, nil }
