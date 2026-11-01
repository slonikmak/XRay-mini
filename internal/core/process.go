package core

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
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

func (execRunner) Test(ctx context.Context, path, config string) error {
	child, err := startVerifiedContained(ctx, path, []string{"run", "-test", "-config", config})
	if err != nil {
		return err
	}
	return child.Wait()
}
func (execRunner) Start(ctx context.Context, path, config string) (childProcess, error) {
	return startVerifiedContained(ctx, path, []string{"run", "-config", config})
}

// Hold a deny-write/delete handle across hash verification and native process
// creation on Windows, closing the ordinary executable replacement race.
func startVerifiedContained(ctx context.Context, path string, args []string) (childProcess, error) {
	release, err := lockExecutable(path)
	if err != nil {
		return nil, err
	}
	defer release()
	if err = validateBinary(path); err != nil {
		return nil, err
	}
	return startContained(ctx, path, args)
}

// Raw child output is never retained or interpreted: even apparently harmless
// Xray messages can contain addresses, credentials or secret URL parameters.
// Fixed-size reads drain both streams, recording only an allowlisted category.
type outputDrain struct {
	mu   sync.Mutex
	seen [2]bool
	wg   sync.WaitGroup
}

func (d *outputDrain) start(r io.ReadCloser, stream int) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer r.Close()
		var b [4096]byte
		for {
			n, e := r.Read(b[:])
			if n > 0 {
				d.mu.Lock()
				d.seen[stream] = true
				d.mu.Unlock()
			}
			if e != nil {
				return
			}
		}
	}()
}
func (d *outputDrain) Diagnostics() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := []string{}
	if d.seen[0] {
		result = append(result, "xray-stdout-observed")
	}
	if d.seen[1] {
		result = append(result, "xray-stderr-observed")
	}
	return result
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
	return probeNetworkVia(ctx, "http://127.0.0.1:10809", "https://www.gstatic.com/generate_204")
}

func probeNetworkVia(ctx context.Context, proxyURL, controlURL string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	proxy, e := url.Parse(proxyURL)
	if e != nil {
		return false
	}
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), TLSHandshakeTimeout: 5 * time.Second}
	transport.DialContext = func(_ context.Context, network, address string) (net.Conn, error) {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			// Transport may detach its dial context from request cancellation.
			// Use the whole probe's budget, and keep this connection for HTTP.
			connection, err := dialer.DialContext(ctx, network, address)
			if err == nil {
				return connection, nil
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
			}
		}
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, controlURL, nil)
	if e != nil {
		return false
	}
	response, e := client.Do(req)
	if e != nil {
		return false
	}
	defer response.Body.Close()
	return ctx.Err() == nil && response.StatusCode == http.StatusNoContent
}
