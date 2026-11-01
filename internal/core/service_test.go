package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testConfig = `{"outbounds":[{"protocol":"vless","settings":{"vnext":[{"address":"example.invalid","port":443,"users":[{"id":"00000000-0000-0000-0000-000000000001"}]}]},"streamSettings":{"network":"raw","security":"reality","realitySettings":{"serverName":"example.invalid","publicKey":"test"}}}]}`

type fakeChild struct {
	done      chan error
	once      sync.Once
	waits     atomic.Int32
	killError error
}

func (c *fakeChild) Wait() error { c.waits.Add(1); return <-c.done }
func (c *fakeChild) Kill() error {
	if c.killError != nil {
		return c.killError
	}
	c.once.Do(func() { c.done <- nil })
	return nil
}

type fakeRunner struct {
	preflight func(context.Context, string) error
	child     *fakeChild
	starts    atomic.Int32
	startGate func(context.Context)
}

func (r *fakeRunner) Test(ctx context.Context, _ string, config string) error {
	if r.preflight != nil {
		return r.preflight(ctx, config)
	}
	return nil
}
func (r *fakeRunner) Start(ctx context.Context, _, _ string) (childProcess, error) {
	r.starts.Add(1)
	if r.startGate != nil {
		r.startGate(ctx)
	}
	return r.child, nil
}
func fixture(t *testing.T) (*Service, Profile, string, *fakeRunner) {
	t.Helper()
	s, e := NewService(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	p, e := s.ImportProfile("Test", testConfig)
	if e != nil {
		t.Fatal(e)
	}
	binary := filepath.Join(t.TempDir(), "fake-xray")
	if e = os.WriteFile(binary, []byte("not executable; fake runner only"), 0600); e != nil {
		t.Fatal(e)
	}
	runner := &fakeRunner{child: &fakeChild{done: make(chan error, 1)}}
	s.runner = runner
	s.ports = func() error { return nil }
	s.probe = func(context.Context) bool { return true }
	return s, p, binary, runner
}
func waitState(t *testing.T, s *Service, state string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.GetStatus().ProcessState == state {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("wanted %s, got %+v", state, s.GetStatus())
}
func TestDisconnectDuringPreparation(t *testing.T) {
	s, p, _, r := fixture(t)
	entered := make(chan struct{})
	r.preflight = func(ctx context.Context, _ string) error { close(entered); <-ctx.Done(); return ctx.Err() }
	if status, e := s.Connect(p.ID); e != nil || status.ProcessState != "preparing" {
		t.Fatalf("%+v %v", status, e)
	}
	<-entered
	if _, e := s.Connect(p.ID); e == nil {
		t.Fatal("second connect accepted")
	}
	start := time.Now()
	s.Disconnect()
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("disconnect blocked")
	}
	waitState(t, s, "stopped")
	if r.starts.Load() != 0 {
		t.Fatal("started after cancellation")
	}
	assertNoRuntime(t, s)
}
func TestRunningDisconnectWaitsOnceAndCleans(t *testing.T) {
	s, p, _, r := fixture(t)
	if _, e := s.Connect(p.ID); e != nil {
		t.Fatal(e)
	}
	waitState(t, s, "running")
	if e := s.DeleteProfile(p.ID); e == nil {
		t.Fatal("deleted active profile")
	}
	s.Disconnect()
	s.Disconnect()
	waitState(t, s, "stopped")
	if r.child.waits.Load() != 1 {
		t.Fatal("Wait must be called once")
	}
	assertNoRuntime(t, s)
}
func TestCloseCancelsPreparationAndRejectsNewWork(t *testing.T) {
	s, p, _, r := fixture(t)
	entered := make(chan struct{})
	r.preflight = func(ctx context.Context, _ string) error { close(entered); <-ctx.Done(); return ctx.Err() }
	_, _ = s.Connect(p.ID)
	<-entered
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Connect(p.ID); e == nil {
		t.Fatal("closed service connected")
	}
	if _, e := s.ImportProfile("new", testConfig); e == nil {
		t.Fatal("closed service imported")
	}
	assertNoRuntime(t, s)
}
func TestPreflightErrorsNeverExposeCoreDiagnostics(t *testing.T) {
	s, p, _, r := fixture(t)
	r.preflight = func(context.Context, string) error { return errors.New("secret-id secret-server private-path") }
	_, _ = s.Connect(p.ID)
	waitState(t, s, "error")
	if strings.Contains(s.GetStatus().Message, "secret") {
		t.Fatal("status leaked diagnostics")
	}
	for _, log := range s.GetLogs() {
		if strings.Contains(log.Message, "secret") {
			t.Fatal("log leaked diagnostics")
		}
	}
	assertNoRuntime(t, s)
}

func TestMissingRuntimeExplainsSupportedLaunchWithoutStartingChild(t *testing.T) {
	s, p, _, r := fixture(t)
	r.preflight = func(context.Context, string) error {
		return errors.Join(errBundledRuntimeMissing, errors.New("secret-private-location"))
	}
	_, _ = s.Connect(p.ID)
	waitState(t, s, "error")
	message := s.GetStatus().Message
	if !strings.Contains(message, "runtime is missing") || !strings.Contains(message, "wails3 task dev") || strings.Contains(message, "secret") {
		t.Fatalf("missing runtime cause lost or unsafe: %q", message)
	}
	if r.starts.Load() != 0 {
		t.Fatal("missing runtime admitted a working child")
	}
	assertNoRuntime(t, s)
}
func TestUnexpectedExitIsNotHealthy(t *testing.T) {
	s, p, _, r := fixture(t)
	probeEntered := make(chan struct{})
	s.probe = func(ctx context.Context) bool { close(probeEntered); <-ctx.Done(); return false }
	_, _ = s.Connect(p.ID)
	<-probeEntered
	r.child.once.Do(func() { r.child.done <- errors.New("exit") })
	waitState(t, s, "error")
	if s.GetStatus().NetworkHealth == "healthy" {
		t.Fatal("exited process marked healthy")
	}
	assertNoRuntime(t, s)
}
func TestStoreRoundTripAndTraversal(t *testing.T) {
	s, p, _, _ := fixture(t)
	list, e := s.ListProfiles()
	if e != nil || len(list) != 1 || list[0].ID != p.ID {
		t.Fatalf("list: %+v %v", list, e)
	}
	if e = s.DeleteProfile("../bad"); e == nil {
		t.Fatal("accepted traversal")
	}
	if e = s.DeleteProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	list, e = s.ListProfiles()
	if e != nil || len(list) != 0 {
		t.Fatal("delete failed")
	}
}
func TestOccupiedPortStopsBeforePreflight(t *testing.T) {
	s, p, _, r := fixture(t)
	s.ports = func() error { return errors.New("busy") }
	_, _ = s.Connect(p.ID)
	waitState(t, s, "error")
	if r.starts.Load() != 0 {
		t.Fatal("started with busy port")
	}
}
func assertNoRuntime(t *testing.T, s *Service) {
	t.Helper()
	entries, e := os.ReadDir(s.store.dir)
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "runtime-") {
			t.Fatal("runtime configuration was not cleaned")
		}
	}
}

func TestRapidReconnectAndRevision(t *testing.T) {
	s, p, _, r := fixture(t)
	var previous uint64
	for i := 0; i < 20; i++ {
		r.child = &fakeChild{done: make(chan error, 1)}
		if _, e := s.Connect(p.ID); e != nil {
			t.Fatal(e)
		}
		s.Disconnect()
		waitState(t, s, "stopped")
		status := s.GetStatus()
		if status.Revision <= previous {
			t.Fatal("revision did not advance")
		}
		previous = status.Revision
		assertNoRuntime(t, s)
	}
}
func TestHealthRepeatsSeriallyAndStops(t *testing.T) {
	s, p, _, _ := fixture(t)
	s.healthInterval = time.Millisecond
	var calls, concurrent atomic.Int32
	s.probe = func(ctx context.Context) bool {
		if concurrent.Add(1) != 1 {
			t.Error("overlapping probes")
		}
		defer concurrent.Add(-1)
		calls.Add(1)
		return true
	}
	_, _ = s.Connect(p.ID)
	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if calls.Load() < 3 {
		t.Fatal("health check did not repeat")
	}
	s.Disconnect()
	waitState(t, s, "stopped")
	count := calls.Load()
	time.Sleep(10 * time.Millisecond)
	if calls.Load() != count {
		t.Fatal("probe continued after stopped")
	}
}

func TestHealthTwoFailuresAndRecovery(t *testing.T) {
	s, p, _, _ := fixture(t)
	s.healthInterval = time.Millisecond
	results := make(chan bool)
	s.probe = func(ctx context.Context) bool {
		select {
		case ok := <-results:
			return ok
		case <-ctx.Done():
			return false
		}
	}
	if _, err := s.Connect(p.ID); err != nil {
		t.Fatal(err)
	}
	var last string
	for i, step := range []struct {
		ok    bool
		state string
	}{{true, "Connected"}, {false, "Connected"}, {false, "Degraded"}, {true, "Connected"}} {
		results <- step.ok
		deadline := time.Now().Add(time.Second)
		for s.GetStatus().LastCheckedAt == last && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		st := s.GetStatus()
		if st.State != step.state || st.LastCheckedAt == "" || st.LastCheckedAt == last {
			t.Fatalf("step %d: %+v", i, st)
		}
		last = st.LastCheckedAt
	}
	s.Disconnect()
	waitState(t, s, "stopped")
}
func TestOldSessionEventsIgnoredAndUniqueIDs(t *testing.T) {
	s, p, _, r := fixture(t)
	s.Connect(p.ID)
	waitState(t, s, "running")
	old := s.GetStatus().SessionID
	s.Disconnect()
	waitState(t, s, "stopped")
	r.child = &fakeChild{done: make(chan error, 1)}
	s.Connect(p.ID)
	waitState(t, s, "running")
	if old == s.GetStatus().SessionID {
		t.Fatal("session ID reused")
	}
	s.events <- workerEvent{sessionID: old, kind: "StopCompleted", message: "stale failure"}
	time.Sleep(5 * time.Millisecond)
	if s.GetStatus().Message == "stale failure" {
		t.Fatal("old event changed state")
	}
}
func TestSlowSubscriberDoesNotBlockManager(t *testing.T) {
	s, p, _, _ := fixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	s.SetEventCallback(func(Status) { once.Do(func() { close(entered) }); <-release })
	s.Connect(p.ID)
	<-entered
	done := make(chan struct{})
	go func() { s.Disconnect(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscriber blocked manager")
	}
	waitState(t, s, "stopped")
	close(release)
}
func TestStoreIODoesNotBlockDisconnectOrSnapshot(t *testing.T) {
	s, p, _, _ := fixture(t)
	s.storeMu.Lock()
	s.Connect(p.ID)
	done := make(chan struct{})
	go func() { s.GetStatus(); s.Disconnect(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("store I/O blocks manager")
	}
	s.storeMu.Unlock()
	waitState(t, s, "stopped")
}
func TestRecentLogsBounded(t *testing.T) {
	s, _, _, _ := fixture(t)
	if len(s.GetRecentLogs(-1)) != 0 {
		t.Fatal("negative limit")
	}
	if len(s.GetRecentLogs(1000)) > 200 {
		t.Fatal("unbounded logs")
	}
}

func TestLateProcessStartAfterCancelStillWaitsAndCleans(t *testing.T) {
	s, p, _, r := fixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	r.startGate = func(context.Context) { close(entered); <-release }
	s.Connect(p.ID)
	<-entered
	s.Disconnect()
	if _, err := s.Connect(p.ID); err == nil {
		t.Fatal("reconnect before cleanup")
	}
	close(release)
	waitState(t, s, "stopped")
	if r.child.waits.Load() != 1 {
		t.Fatal("late child not waited exactly once")
	}
	assertNoRuntime(t, s)
}

// Diagnostic category filtering is fail closed even for an erroneous adapter.
func TestDiagnosticAllowlistAndBound(t *testing.T) {
	s, p, _, _ := fixture(t)
	s.Connect(p.ID)
	waitState(t, s, "running")
	id := s.GetStatus().SessionID
	s.events <- workerEvent{sessionID: id, kind: "Diagnostic", message: "secret-id https://secret.invalid/?token=secret", at: time.Now()}
	for i := 0; i < 220; i++ {
		s.events <- workerEvent{sessionID: id, kind: "Diagnostic", message: "xray-stderr-observed", at: time.Now()}
	}
	deadline := time.Now().Add(time.Second)
	for len(s.GetRecentLogs(1000)) < 200 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	logs := s.GetRecentLogs(1000)
	if len(logs) != 200 {
		t.Fatalf("bounded log count %d", len(logs))
	}
	for _, entry := range logs {
		if strings.Contains(entry.Message, "secret") {
			t.Fatal("untrusted diagnostic leaked")
		}
	}
	if len(s.GetRecentLogs(3)) != 3 {
		t.Fatal("limit not enforced")
	}
}

func TestCloseBoundsBlockedNotificationAndWaitsForChild(t *testing.T) {
	s, p, _, r := fixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce, enteredOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var callbacks atomic.Int32
	s.SetEventCallback(func(Status) {
		callbacks.Add(1)
		enteredOnce.Do(func() { close(entered) })
		<-release
	})
	if _, err := s.Connect(p.ID); err != nil {
		t.Fatal(err)
	}
	<-entered
	waitState(t, s, "running")
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	select {
	case err := <-closed:
		if !errors.Is(err, errNotificationPending) || errors.Is(err, errStopPending) {
			t.Fatalf("wanted notification-only failure, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocked callback held Close")
	}
	if r.child.waits.Load() != 1 || s.GetStatus().ProcessState != "stopped" {
		t.Fatalf("Close did not confirm child cleanup: %+v", s.GetStatus())
	}
	assertNoRuntime(t, s)
	// Releasing the existing invocation must not drain pending notifications.
	releaseOnce.Do(func() { close(release) })
	select {
	case <-s.publisherDone:
	case <-time.After(time.Second):
		t.Fatal("publisher did not stop after callback returned")
	}
	if callbacks.Load() != 1 {
		t.Fatalf("callback invoked %d times", callbacks.Load())
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFailedKillRetainsWaitAndOwnershipUntilActualExit(t *testing.T) {
	s, p, _, r := fixture(t)
	r.child.killError = errors.New("private-process-detail")
	// Always release the synthetic unreturned Wait before fixture Close, even
	// when a fatal assertion aborts the test.
	t.Cleanup(func() { r.child.once.Do(func() { r.child.done <- nil }) })
	if _, err := s.Connect(p.ID); err != nil {
		t.Fatal(err)
	}
	waitState(t, s, "running")
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	select {
	case err := <-closed:
		if !errors.Is(err, errStopPending) {
			t.Fatalf("wanted unconfirmed stop error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("failed Kill was not reported promptly")
	}
	status := s.GetStatus()
	if status.State != "Error" || status.ProcessState != "stopping" || strings.Contains(status.Message, "private") {
		t.Fatalf("unsafe/false stopped status: %+v", status)
	}
	if _, err := s.Connect(p.ID); err == nil || r.starts.Load() != 1 {
		t.Fatal("new child admitted before actual exit")
	}
	if replacement, err := NewService(s.store.dir); err == nil {
		replacement.Close()
		t.Fatal("store lock released while child was still owned")
	}
	if r.child.waits.Load() != 1 {
		t.Fatal("Wait not owned exactly once")
	}
	r.child.once.Do(func() { r.child.done <- nil })
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	assertNoRuntime(t, s)
}
