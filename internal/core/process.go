package core

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type childProcess interface {
	Wait() error
	Kill() error
}
type processRunner interface {
	Test(context.Context, string, string) error
	Start(context.Context, string, string) (childProcess, error)
}
type execRunner struct{}
type execChild struct {
	cmd     *exec.Cmd
	release func()
}

func (c *execChild) Wait() error { defer c.release(); return c.cmd.Wait() }
func (c *execChild) Kill() error { return c.cmd.Process.Kill() }
func (execRunner) Test(ctx context.Context, path, config string) error {
	cmd := exec.CommandContext(ctx, path, "run", "-test", "-config", config)
	cmd.Dir = filepath.Dir(path)
	configureCommand(cmd)
	// nil stdout/stderr are the null device. Xray diagnostics may contain credentials.
	return cmd.Run()
}
func (execRunner) Start(ctx context.Context, path, config string) (childProcess, error) {
	cmd := exec.CommandContext(ctx, path, "run", "-config", config)
	cmd.Dir = filepath.Dir(path)
	configureCommand(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	release, err := containProcess(cmd.Process)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	return &execChild{cmd, release}, nil
}
func validateBinary(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("choose an absolute path to a trusted Xray executable")
	}
	fi, e := os.Stat(path)
	if e != nil || !fi.Mode().IsRegular() {
		return errors.New("Xray executable does not exist or is not a regular file")
	}
	return nil
}
func checkPorts() error {
	var listeners []net.Listener
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()
	for _, address := range []string{"127.0.0.1:10808", "127.0.0.1:10809"} {
		l, e := net.Listen("tcp", address)
		if e != nil {
			return errors.New("local proxy port 10808 or 10809 is already in use")
		}
		listeners = append(listeners, l)
	}
	return nil
}
func probeNetwork(ctx context.Context) bool {
	proxy, _ := url.Parse("http://127.0.0.1:10809")
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.gstatic.com/generate_204", nil)
	if e != nil {
		return false
	}
	response, e := client.Do(req)
	if e != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusNoContent
}
