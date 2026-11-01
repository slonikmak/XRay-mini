//go:build windows

package core

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// This opt-in suite needs its executable beside runtime/xray.exe, e.g.:
// go test -c ./internal/core -o bin/core-native.test.exe
// XRAY_MINI_NATIVE_LOOPBACK=1 bin/core-native.test.exe -test.run TestNativeWindows -test.v -test.timeout 180s
// Only the health URL/gate and health interval differ from production. All imports,
// fixed proxy ports, executable verification, DPAPI, ACLs and process jobs are real.
func TestNativeWindowsLoopback(t *testing.T) {
	if os.Getenv("XRAY_MINI_NATIVE_LOOPBACK") != "1" {
		t.Skip("set XRAY_MINI_NATIVE_LOOPBACK=1 for owned native loopback fixtures")
	}
	if err := validateBinary(bundledRuntimePath()); err != nil {
		t.Fatal(err)
	}
	if err := checkPorts(); err != nil {
		t.Fatal("fixed product ports busy; no existing process was touched")
	}
	t.Logf("verified Xray %s sha256=%s; native Windows loopback only", XrayVersion, XrayExecutableSHA256)
	for _, transport := range []struct{ network, security string }{{"raw", "reality"}, {"xhttp", "reality"}, {"xhttp", "tls"}} {
		t.Run(transport.network+"-"+transport.security, func(t *testing.T) {
			nativeTransportCase(t, transport.network, transport.security)
		})
	}
	if err := checkPorts(); err != nil {
		t.Fatal("fixed product listeners were not released")
	}
}

// Observe actual execRunner starts/Wait calls without replacing any process work.
type nativeObservedRunner struct {
	execRunner
	mu       sync.Mutex
	children []*nativeObservedChild
}

type nativeObservedChild struct {
	childProcess
	waits atomic.Int32
}

func (c *nativeObservedChild) Wait() error {
	c.waits.Add(1)
	return c.childProcess.Wait()
}

func (c *nativeObservedChild) Diagnostics() []string {
	if d, ok := c.childProcess.(interface{ Diagnostics() []string }); ok {
		return d.Diagnostics()
	}
	return nil
}

func (r *nativeObservedRunner) Start(ctx context.Context, path, config string) (childProcess, error) {
	c, err := r.execRunner.Start(ctx, path, config)
	if err != nil {
		return nil, err
	}
	observed := &nativeObservedChild{childProcess: c}
	r.mu.Lock()
	r.children = append(r.children, observed)
	r.mu.Unlock()
	return observed, nil
}

func (r *nativeObservedRunner) childrenSnapshot() []*nativeObservedChild {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*nativeObservedChild(nil), r.children...)
}

func nativeTransportCase(t *testing.T, network, security string) {
	t.Helper()
	// t.TempDir would unconditionally delete configuration after failed teardown.
	work, err := os.MkdirTemp("", "xray-native-")
	if err != nil {
		t.Fatal("cannot create owned fixture directory")
	}
	retainWork := false
	t.Cleanup(func() {
		if retainWork {
			t.Error("owned fixture configuration retained because child exit is unconfirmed")
			return
		}
		if err := os.RemoveAll(work); err != nil {
			t.Error("owned fixture directory cleanup failed")
		}
	})
	if err := secureDirectory(work); err != nil {
		t.Fatal(err)
	}
	cert, certPath, keyPath, pin := nativeCertificate(t, work)
	camouflage := nativeTLSCamouflage(t, cert)
	var hits atomic.Int32
	var tokenBytes [16]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		t.Fatal(err)
	}
	token := hex.EncodeToString(tokenBytes[:])
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health204" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		hits.Add(1)
		_, _ = io.WriteString(w, token)
	}))
	t.Cleanup(target.Close)
	targetURL, _ := url.Parse(target.URL)
	_, targetPort, _ := net.SplitHostPort(targetURL.Host)
	peerPort := nativeFreePort(t)
	private, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serverStream := map[string]any{"network": network, "security": security}
	clientStream := map[string]any{"network": network, "security": security}
	if network == "xhttp" {
		serverStream["xhttpSettings"] = map[string]any{"path": "/fixture", "mode": "auto"}
		clientStream["xhttpSettings"] = map[string]any{"path": "/fixture", "mode": "auto"}
	}
	if security == "reality" {
		serverStream["realitySettings"] = map[string]any{"target": camouflage, "serverNames": []string{"localhost"}, "privateKey": base64.RawURLEncoding.EncodeToString(private.Bytes()), "shortIds": []string{"0123456789abcdef"}}
		clientStream["realitySettings"] = map[string]any{"serverName": "localhost", "fingerprint": "chrome", "password": base64.RawURLEncoding.EncodeToString(private.PublicKey().Bytes()), "shortId": "0123456789abcdef"}
	} else {
		serverStream["tlsSettings"] = map[string]any{"certificates": []any{map[string]any{"certificateFile": certPath, "keyFile": keyPath}}, "alpn": []string{"h2"}}
		// Exact generated leaf-certificate verification, not OS trust-chain acceptance.
		clientStream["tlsSettings"] = map[string]any{"serverName": "localhost", "alpn": []string{"h2"}, "pinnedPeerCertSha256": pin}
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		t.Fatal(err)
	}
	id[6], id[8] = id[6]&15|64, id[8]&63|128
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
	user, peer := map[string]any{"id": uuid, "encryption": "none"}, map[string]any{"id": uuid}
	if network == "raw" {
		user["flow"], peer["flow"] = "xtls-rprx-vision", "xtls-rprx-vision"
	}
	serverConfig := nativeJSON(t, map[string]any{
		"log":       map[string]any{"loglevel": "none"},
		"inbounds":  []any{map[string]any{"listen": "127.0.0.1", "port": peerPort, "protocol": "vless", "settings": map[string]any{"clients": []any{peer}, "decryption": "none"}, "streamSettings": serverStream}},
		"outbounds": []any{map[string]any{"protocol": "freedom", "settings": map[string]any{"finalRules": []any{map[string]any{"action": "allow", "ip": []string{"127.0.0.1"}, "port": targetPort}}}}},
	})
	serverPath := filepath.Join(work, "peer.json")
	if err := os.WriteFile(serverPath, []byte(serverConfig), 0600); err != nil {
		t.Fatal("cannot write owned peer config")
	}
	initialPeer := nativeStartPeer(t, serverPath, peerPort, &retainWork)
	imported := func() string {
		return nativeJSON(t, map[string]any{"outbounds": []any{
			map[string]any{"protocol": "vless", "settings": map[string]any{"vnext": []any{map[string]any{"address": "127.0.0.1", "port": peerPort, "users": []any{user}}}}, "streamSettings": clientStream},
			map[string]any{"protocol": "freedom", "tag": "untrusted-direct"}},
			"routing": map[string]any{"rules": []any{map[string]any{"type": "field", "network": "tcp", "outboundTag": "untrusted-direct"}}}})
	}
	s, err := NewService(filepath.Join(work, "store"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &nativeObservedRunner{}
	s.runner = runner
	s.healthInterval = 200 * time.Millisecond
	gate := make(chan struct{})
	var gateOnce sync.Once
	releaseGate := func() { gateOnce.Do(func() { close(gate) }) }
	s.probe = func(ctx context.Context) bool {
		select {
		case <-gate:
		case <-ctx.Done():
			return false
		}
		return nativeRequest(ctx, "http", target.URL+"/health204", "", http.StatusNoContent)
	}
	t.Cleanup(func() {
		releaseGate()
		if err := s.Close(); err != nil {
			retainWork = true
			t.Error("native Service cleanup failed:", err)
		}
		for _, child := range runner.childrenSnapshot() {
			if child.waits.Load() != 1 {
				t.Error("working child did not receive exactly one Wait")
			}
		}
		nativeNoRuntime(t, s.store.dir)
		if err := checkPorts(); err != nil {
			t.Error("fixed proxy listeners remain after Service.Close")
		}
	})
	profile, err := s.ImportProfile("Native loopback fixture", imported())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Connect(profile.ID); err != nil {
		t.Fatal(err)
	}
	nativeWait(t, "running before probe completion", func() bool {
		st := s.GetStatus()
		return st.State == "Connecting" && st.ProcessState == "running"
	})
	// Withheld result proves process launch alone does not report Connected.
	time.Sleep(100 * time.Millisecond)
	if st := s.GetStatus(); st.State != "Connecting" || st.LastCheckedAt != "" {
		t.Fatal("Connected/health result appeared before the real probe gate was released")
	}
	releaseGate()
	nativeWait(t, "Connected through local real 204", func() bool { return s.GetStatus().State == "Connected" })
	if _, err := s.Connect(profile.ID); err == nil {
		t.Fatal("duplicate Connect accepted while the real client was Connected")
	}
	if len(runner.childrenSnapshot()) != 1 {
		t.Fatal("duplicate Connect launched another working client")
	}
	t.Log("PASS duplicate Connect rejected while Connected; client launches=1")
	for _, protocol := range []string{"http", "socks5"} {
		if !nativeRequest(context.Background(), protocol, target.URL+"/token", token, http.StatusOK) {
			t.Fatalf("%s target response failed", protocol)
		}
		t.Logf("PASS %s target response", protocol)
	}
	if !initialPeer.stop() {
		t.Fatal("owned peer exit unconfirmed; cleanup will retry")
	}
	if !nativeRequest(context.Background(), "", target.URL+"/token", token, http.StatusOK) {
		t.Fatal("target stopped independently of peer")
	}
	before := hits.Load()
	for _, protocol := range []string{"http", "socks5"} {
		if nativeRequest(context.Background(), protocol, target.URL+"/token", token, http.StatusOK) {
			t.Fatalf("%s bypassed stopped peer", protocol)
		}
	}
	nativeWait(t, "Degraded after peer loss", func() bool { return s.GetStatus().State == "Degraded" })
	if hits.Load() != before {
		t.Fatal("target received traffic while peer was stopped")
	}
	restoredPeer := nativeStartPeer(t, serverPath, peerPort, &retainWork)
	nativeWait(t, "Connected after same peer restored", func() bool { return s.GetStatus().State == "Connected" })
	if len(runner.childrenSnapshot()) != 1 {
		t.Fatal("client restarted during peer failure/recovery")
	}
	t.Log("PASS Connecting gate; stopped peer blocks both proxies; Degraded; restored peer returns Connected; client launches=1")
	if network == "raw" {
		if err := runner.childrenSnapshot()[0].Kill(); err != nil {
			t.Fatal("cannot terminate owned working child")
		}
		nativeWait(t, "Error after real working child exit and cleanup", func() bool {
			status := s.GetStatus()
			return status.State == "Error" && status.SessionID == ""
		})
		t.Log("PASS real working child exit becomes Error after cleanup")
	} else {
		s.Disconnect()
		nativeWait(t, "Disconnected", func() bool { return s.GetStatus().State == "Disconnected" })
		t.Log("PASS Disconnect becomes Disconnected after cleanup")
	}
	nativeNoRuntime(t, s.store.dir)
	if err := checkPorts(); err != nil {
		t.Fatal("cleanup completion left fixed proxy listeners")
	}
	if security == "tls" {
		restoredPeer.requireAlive(t)
		clientStream["tlsSettings"].(map[string]any)["pinnedPeerCertSha256"] = strings.Repeat("0", 64)
		wrong, err := s.ImportProfile("Native wrong pin fixture", imported())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Connect(wrong.ID); err != nil {
			t.Fatal(err)
		}
		nativeWait(t, "Degraded for wrong leaf pin", func() bool { return s.GetStatus().State == "Degraded" })
		restoredPeer.requireAlive(t)
		before := hits.Load()
		for _, protocol := range []string{"http", "socks5"} {
			restoredPeer.requireAlive(t)
			if nativeRequest(context.Background(), protocol, target.URL+"/token", token, http.StatusOK) {
				t.Fatalf("%s accepted wrong leaf pin", protocol)
			}
			restoredPeer.requireAlive(t)
		}
		if hits.Load() != before || !nativeRequest(context.Background(), "", target.URL+"/token", token, http.StatusOK) {
			t.Fatal("wrong-pin negative control did not isolate a healthy target")
		}
		s.Disconnect()
		nativeWait(t, "Disconnected after wrong pin", func() bool { return s.GetStatus().State == "Disconnected" })
		restoredPeer.requireAlive(t)
		if _, err := s.Connect(profile.ID); err != nil {
			t.Fatal(err)
		}
		nativeWait(t, "Connected with correct pin after negative control", func() bool { return s.GetStatus().State == "Connected" })
		for _, protocol := range []string{"http", "socks5"} {
			if !nativeRequest(context.Background(), protocol, target.URL+"/token", token, http.StatusOK) {
				t.Fatalf("%s correct-pin response failed after negative control", protocol)
			}
		}
		restoredPeer.requireAlive(t)
		s.Disconnect()
		nativeWait(t, "Disconnected after correct-pin control", func() bool { return s.GetStatus().State == "Disconnected" })
		t.Log("PASS wrong leaf pin rejects both proxies with peer alive throughout; correct pin then succeeds through both proxies")
	}
}

func nativeJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal("cannot encode fixture configuration")
	}
	return string(b)
}

func nativeWait(t *testing.T, condition string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("timeout waiting for " + condition)
}

func nativeNoRuntime(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Error("cannot inspect owned Service store")
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "runtime-") {
			t.Error("temporary Service configuration remains after cleanup")
		}
	}
}

func nativeRequest(parent context.Context, protocol, address, token string, status int) bool {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	transport := &http.Transport{DisableKeepAlives: true, DialContext: (&net.Dialer{Timeout: time.Second}).DialContext}
	if protocol != "" {
		port := "10809"
		if protocol == "socks5" {
			port = "10808"
		}
		proxy, _ := url.Parse(protocol + "://127.0.0.1:" + port)
		transport.Proxy = http.ProxyURL(proxy)
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return false
	}
	response, err := client.Do(req)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 128))
	return err == nil && response.StatusCode == status && string(body) == token
}

func nativeFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("cannot allocate loopback peer port")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return port
}

type nativePeer struct {
	stop func() bool
	done <-chan struct{}
}

func (p nativePeer) requireAlive(t *testing.T) {
	t.Helper()
	select {
	case <-p.done:
		t.Fatal("owned peer exited during TLS pin control")
	default:
	}
}

// One Wait per fixture child; a timed-out stop leaves cleanup free to retry.
func nativeStartPeer(t *testing.T, config string, port int, retainWork *bool) nativePeer {
	t.Helper()
	runner := execRunner{}
	preflight, stop := context.WithTimeout(context.Background(), 10*time.Second)
	err := runner.Test(preflight, bundledRuntimePath(), config)
	stop()
	if err != nil {
		t.Fatal("verified fixture preflight failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	child, err := runner.Start(ctx, bundledRuntimePath(), config)
	if err != nil {
		cancel()
		t.Fatal("verified fixture launch failed")
	}
	done := make(chan struct{})
	go func() { _ = child.Wait(); close(done) }()
	confirmed := false
	stopPeer := func() bool {
		if confirmed {
			return true
		}
		cancel()
		_ = child.Kill()
		for attempt := 0; attempt < 2; attempt++ {
			select {
			case <-done:
				confirmed = true
				listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
				if err != nil {
					t.Error("owned fixture listener was not released")
				} else {
					_ = listener.Close()
				}
				return true
			case <-time.After(5 * time.Second):
				// Closing our non-inherited kill-on-close job also works when
				// TerminateJobObject failed. Wait still owns the process handle.
				if native, ok := child.(*windowsChild); ok {
					native.mu.Lock()
					if native.job != 0 && syscall.CloseHandle(native.job) == nil {
						native.job = 0
					}
					native.mu.Unlock()
				}
			}
		}
		return false
	}
	t.Cleanup(func() {
		if !stopPeer() {
			*retainWork = true
			t.Error("owned fixture exit/join unconfirmed after bounded cleanup retry")
		}
	})
	nativeWait(t, "fixture listener", func() bool {
		select {
		case <-done:
			t.Fatal("owned fixture exited before listener readiness")
		default:
		}
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
		}
		return err == nil
	})
	return nativePeer{stop: stopPeer, done: done}
}

func nativeCertificate(t *testing.T, dir string) (tls.Certificate, string, string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal("fixture RSA generation failed")
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal("fixture certificate generation failed")
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPath, keyPath := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if os.WriteFile(certPath, certPEM, 0600) != nil || os.WriteFile(keyPath, keyPEM, 0600) != nil {
		t.Fatal("cannot write owned fixture certificate")
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal("cannot load fixture certificate")
	}
	pin := sha256.Sum256(der)
	return cert, certPath, keyPath, hex.EncodeToString(pin[:])
}

func nativeTLSCamouflage(t *testing.T, cert tls.Certificate) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("cannot start loopback TLS camouflage")
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13, NextProtos: []string{"h2", "http/1.1"}}
	var workers sync.WaitGroup
	var mu sync.Mutex
	connections := map[net.Conn]bool{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			connections[conn] = true
			mu.Unlock()
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer conn.Close()
				defer func() { mu.Lock(); delete(connections, conn); mu.Unlock() }()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				secured := tls.Server(conn, config)
				if secured.Handshake() == nil {
					_, _ = io.Copy(io.Discard, secured)
				}
			}()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-done
		mu.Lock()
		for conn := range connections {
			_ = conn.Close()
		}
		mu.Unlock()
		workers.Wait()
	})
	return listener.Addr().String()
}
