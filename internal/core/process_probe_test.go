package core

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

const probeControlURL = "http://control.invalid/generate_204"

func startProbe(t *testing.T, ctx context.Context, proxyURL string) <-chan bool {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	done := make(chan bool, 1)
	go func() { done <- probeNetworkVia(ctx, proxyURL, probeControlURL) }()
	return done
}

func expectProbe(t *testing.T, done <-chan bool, want bool) {
	t.Helper()
	select {
	case got := <-done:
		if got != want {
			t.Fatalf("probe = %v, want %v", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("probe did not finish promptly")
	}
}

func expectProbePending(t *testing.T, done <-chan bool) {
	t.Helper()
	select {
	case got := <-done:
		t.Fatalf("probe finished before readiness/control response: %v", got)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestProbeNetworkDelayedListenerAndPendingResponse(t *testing.T) {
	var requests, connections atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.String() != probeControlURL || r.Method != http.MethodGet {
			t.Error("control request was not sent through the proxy")
		}
		close(entered)
		select {
		case <-release:
			w.WriteHeader(http.StatusNoContent)
		case <-r.Context().Done():
		}
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	address := server.Listener.Addr().String()
	server.Listener.Close()
	t.Cleanup(server.Close)
	done := startProbe(t, context.Background(), "http://"+address)
	// Keep the real loopback port closed long enough for the first refused dial.
	// The old immediate-dial implementation fails here, before a listener exists.
	expectProbePending(t, done)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = listener
	server.Start()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first probe did not reach the newly ready proxy")
	}
	expectProbePending(t, done)
	close(release)
	expectProbe(t, done, true)
	if requests.Load() != 1 || connections.Load() != 1 {
		t.Fatalf("requests = %d, connections = %d; readiness must carry the sole request", requests.Load(), connections.Load())
	}
}

func TestProbeNetworkReadyProxyResponses(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusOK, http.StatusServiceUnavailable, http.StatusFound, 0} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.String() != probeControlURL {
					t.Error("unexpected proxy control URL")
				}
				if status == 0 {
					connection, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					connection.Close()
					return
				}
				w.Header().Set("Location", probeControlURL+"/redirect")
				w.WriteHeader(status)
			}))
			defer server.Close()
			if got := probeNetworkVia(context.Background(), server.URL, probeControlURL); got != (status == http.StatusNoContent) {
				t.Fatalf("probe = %v for status %d", got, status)
			}
			if requests.Load() != 1 {
				t.Fatalf("control request retried: %d requests", requests.Load())
			}
		})
	}
}

func TestProbeNetworkRejectsUntrustedControlTLS(t *testing.T) {
	certificateServer := httptest.NewTLSServer(http.NotFoundHandler())
	defer certificateServer.Close()
	var requests atomic.Int32
	handshakeDone := make(chan struct{})
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		defer close(handshakeDone)
		if r.Method != http.MethodConnect {
			t.Error("HTTPS control request did not establish a proxy tunnel")
			return
		}
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer connection.Close()
		if _, err := io.WriteString(connection, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			t.Error(err)
			return
		}
		if tls.Server(connection, certificateServer.TLS).Handshake() == nil {
			t.Error("control TLS accepted an untrusted test certificate")
		}
	}))
	defer proxy.Close()
	if probeNetworkVia(context.Background(), proxy.URL, "https://control.invalid/generate_204") {
		t.Fatal("untrusted control TLS reported healthy")
	}
	select {
	case <-handshakeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("failed TLS left its connection open")
	}
	if requests.Load() != 1 {
		t.Fatalf("failed TLS request retried: %d", requests.Load())
	}
}

func closedProbeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	return "http://" + address
}

func TestProbeNetworkNeverReadyHasTotalFiveSecondBudget(t *testing.T) {
	proxy := closedProbeAddress(t)
	started := time.Now()
	if probeNetworkVia(context.Background(), proxy, probeControlURL) {
		t.Fatal("closed proxy reported healthy")
	}
	if elapsed := time.Since(started); elapsed < 4800*time.Millisecond || elapsed > 6500*time.Millisecond {
		t.Fatalf("total probe budget was not five seconds: %s", elapsed)
	}
}

func TestProbeNetworkCancellationDuringReadiness(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startProbe(t, ctx, closedProbeAddress(t))
	expectProbePending(t, done)
	cancel()
	expectProbe(t, done, false)
}

func TestProbeNetworkCancellationDuringHTTP(t *testing.T) {
	entered, stopped := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(stopped)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startProbe(t, ctx, server.URL)
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("control request did not reach proxy")
	}
	cancel()
	expectProbe(t, done, false)
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("canceled control request kept its connection open")
	}
}

func TestProbeNetworkReadinessAndHTTPShareDeadline(t *testing.T) {
	entered := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
	}))
	address := server.Listener.Addr().String()
	server.Listener.Close()
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	done := startProbe(t, ctx, "http://"+address)
	expectProbePending(t, done)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = listener
	server.Start()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("request did not reach proxy within shared budget")
	}
	expectProbe(t, done, false)
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatal("pending HTTP response failed before the shared deadline")
	}
}
