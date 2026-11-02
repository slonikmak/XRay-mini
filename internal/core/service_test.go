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
	kills     atomic.Int32
	killError error
	killHook  func() error
}

func (c *fakeChild) Wait() error { c.waits.Add(1); return <-c.done }
func (c *fakeChild) Kill() error {
	c.kills.Add(1)
	if c.killHook != nil {
		if err := c.killHook(); err != nil {
			return err
		}
	}
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
		status := s.GetStatus()
		if status.ProcessState == state && ((state != "stopped" && state != "error") || status.SessionID == "") {
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
	for _, log := range s.GetRecentLogs(200) {
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
	waitForService(t, func() bool { return s.GetStatus().State == "Connected" })
	before := s.GetStatus()
	for _, kind := range []string{"ProcessExited", "ProcessStarted", "ProxyReady", "HealthFailed", "ConfigValidated", "CleanupFailed", "StopFailed", "StopCompleted"} {
		s.events <- workerEvent{sessionID: old, kind: kind, message: "stale failure", ok: true}
	}
	s.events <- workerEvent{sessionID: before.SessionID, kind: "Diagnostic", message: "xray-stdout-observed", at: time.Now()}
	waitForService(t, func() bool {
		for _, log := range s.GetRecentLogs(200) {
			if log.Message == "xray-stdout-observed" {
				return true
			}
		}
		return false
	})
	if s.GetStatus() != before {
		t.Fatalf("old event changed new session: %+v", s.GetStatus())
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
	for i := 0; i < 220; i++ {
		s.events <- workerEvent{sessionID: id, kind: "Diagnostic", message: "xray-stderr-observed", at: time.Now()}
	}
	for _, message := range []string{"secret-id https://secret.invalid/?token=secret", "xray-output", "xray-warning", "xray-error"} {
		s.events <- workerEvent{sessionID: id, kind: "Diagnostic", message: message, at: time.Now()}
	}
	s.events <- workerEvent{sessionID: id, kind: "Diagnostic", message: "xray-stdout-observed", at: time.Now()}
	waitForService(t, func() bool {
		for _, entry := range s.GetRecentLogs(200) {
			if entry.Message == "xray-stdout-observed" {
				return true
			}
		}
		return false
	})
	logs := s.GetRecentLogs(1000)
	if len(logs) != 200 {
		t.Fatalf("bounded log count %d", len(logs))
	}
	for _, entry := range logs {
		if strings.Contains(entry.Message, "secret") || entry.Message == "xray-output" || entry.Message == "xray-warning" || entry.Message == "xray-error" {
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
	// Sending the fake's result only makes Wait runnable. A Close before the
	// manager observes ProcessExited legitimately retries the still-failing Kill.
	waitForService(t, func() bool { return s.GetStatus().ProcessState == "stopped" })
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if r.child.waits.Load() != 1 || s.GetStatus().SessionID != "" {
		t.Fatalf("Close did not join the owned exit/cleanup: %+v", s.GetStatus())
	}
	assertNoRuntime(t, s)
}

func TestExitedSessionRejectsLateEventsUntilCleanup(t *testing.T) {
	for _, kind := range []string{"ProcessStarted", "ProxyReady", "HealthFailed", "ConfigValidated", "StopFailed"} {
		t.Run(kind, func(t *testing.T) {
			s, p, _, r := fixture(t)
			probeEntered, releaseProbe := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			s.probe = func(ctx context.Context) bool { close(probeEntered); <-ctx.Done(); <-releaseProbe; return true }
			defer releaseOnce.Do(func() { close(releaseProbe) })
			s.Connect(p.ID)
			<-probeEntered
			id := s.GetStatus().SessionID
			s.events <- workerEvent{sessionID: id, kind: "ProcessExited", ok: true}
			s.events <- workerEvent{sessionID: id, kind: kind, ok: true, at: time.Now()}
			s.events <- workerEvent{sessionID: id, kind: "Diagnostic", message: "xray-stdout-observed", at: time.Now()}
			waitForService(t, func() bool {
				for _, log := range s.GetRecentLogs(200) {
					if log.Message == "xray-stdout-observed" {
						return true
					}
				}
				return false
			})
			st := s.GetStatus()
			if st.ProcessState != "stopped" || st.NetworkHealth == "healthy" || st.SessionID != id || !strings.Contains(st.Message, "unexpectedly") {
				t.Fatalf("late %s resurrected/lost terminal state: %+v", kind, st)
			}
			if _, err := s.Connect(p.ID); err == nil {
				t.Fatal("connected before cleanup")
			}
			st = s.Disconnect()
			if st.ProcessState != "stopped" || !strings.Contains(st.Message, "unexpectedly") {
				t.Fatalf("Disconnect erased exit cause: %+v", st)
			}
			// StopCompleted must retain the cause even though manager cancellation
			// has already reached the supervisor.
			releaseOnce.Do(func() { close(releaseProbe) })
			waitState(t, s, "error")
			if !strings.Contains(s.GetStatus().Message, "unexpectedly") || r.child.waits.Load() != 1 {
				t.Fatalf("final cause or Wait ownership lost: %+v", s.GetStatus())
			}
			assertNoRuntime(t, s)
		})
	}
}

func TestUnexpectedExitFactSurvivesStopCommandBeforeEvent(t *testing.T) {
	s, p, _, r := fixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseKill := make(chan struct{})
	var releaseKillOnce sync.Once
	defer releaseKillOnce.Do(func() { close(releaseKill) })
	r.child.killHook = func() error { <-releaseKill; return nil }
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	s.probe = func(ctx context.Context) bool { close(entered); <-ctx.Done(); <-release; return false }
	s.Connect(p.ID)
	<-entered
	// Wait captured an unexpected exit before publishing. Force the stop
	// command to be consumed before that captured fact reaches the manager.
	exit := workerEvent{sessionID: s.GetStatus().SessionID, kind: "ProcessExited", ok: true}
	s.Disconnect()
	s.events <- exit
	waitForService(t, func() bool { return s.GetStatus().ProcessState == "stopped" })
	st := s.GetStatus()
	if st.State != "Error" || st.SessionID == "" || !strings.Contains(st.Message, "unexpectedly") {
		t.Fatalf("stop command erased captured unexpected exit: %+v", st)
	}
	releaseKillOnce.Do(func() { close(releaseKill) })
	releaseOnce.Do(func() { close(release) })
	waitState(t, s, "error")
	if !strings.Contains(s.GetStatus().Message, "unexpectedly") {
		t.Fatal("cleanup erased captured exit cause")
	}
}

func TestRepeatedStopBeforeFailureNotificationRetriesOnce(t *testing.T) {
	s, p, _, r := fixture(t)
	r.child.killHook = func() error {
		switch r.child.kills.Load() {
		case 1:
			s.Disconnect() // consumed before the first StopFailed can be sent
			return errors.New("first stop failed")
		case 2:
			return errors.New("second stop failed")
		}
		return nil
	}
	t.Cleanup(func() { r.child.once.Do(func() { r.child.done <- nil }) })
	s.Connect(p.ID)
	waitState(t, s, "running")
	s.Disconnect()
	waitForService(t, func() bool { return r.child.kills.Load() == 2 && s.GetStatus().State == "Error" })
	if s.GetStatus().SessionID == "" || r.child.waits.Load() != 1 {
		t.Fatal("failed attempts lost session/Wait ownership")
	}
	time.Sleep(10 * time.Millisecond)
	if r.child.kills.Load() != 2 {
		t.Fatal("failed stop retried without another explicit request")
	}
	s.Disconnect()
	waitState(t, s, "stopped")
	if r.child.kills.Load() != 3 || r.child.waits.Load() != 1 {
		t.Fatalf("kills=%d waits=%d", r.child.kills.Load(), r.child.waits.Load())
	}
}

func waitForService(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("service condition not reached")
}

func TestCancelledSessionRejectsLateProgressUntilCleanup(t *testing.T) {
	s, p, _, r := fixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	r.preflight = func(ctx context.Context, _ string) error {
		close(entered)
		<-ctx.Done()
		<-release
		return ctx.Err()
	}
	s.Connect(p.ID)
	<-entered
	before := s.Disconnect()
	for _, kind := range []string{"ConfigValidated", "ProcessStarted", "ProxyReady", "HealthFailed"} {
		s.events <- workerEvent{sessionID: before.SessionID, kind: kind, ok: true, at: time.Now()}
	}
	s.events <- workerEvent{sessionID: before.SessionID, kind: "Diagnostic", message: "xray-stdout-observed", at: time.Now()}
	waitForService(t, func() bool {
		for _, log := range s.GetRecentLogs(200) {
			if log.Message == "xray-stdout-observed" {
				return true
			}
		}
		return false
	})
	if s.GetStatus() != before {
		t.Fatalf("late progress changed canceled session: %+v", s.GetStatus())
	}
	if _, err := s.Connect(p.ID); err == nil {
		t.Fatal("connected before canceled worker cleanup")
	}
	releaseOnce.Do(func() { close(release) })
	waitState(t, s, "stopped")
	assertNoRuntime(t, s)
}

func TestExplicitStopRetriesFailedKillWithoutAnotherWait(t *testing.T) {
	for _, recovery := range []string{"disconnect", "close"} {
		t.Run(recovery, func(t *testing.T) {
			s, p, _, r := fixture(t)
			r.child.killHook = func() error {
				if r.child.kills.Load() <= 3 {
					return errors.New("private-stop-detail")
				}
				return nil
			}
			t.Cleanup(func() { r.child.once.Do(func() { r.child.done <- nil }) })
			s.Connect(p.ID)
			waitState(t, s, "running")
			for attempt := int32(1); attempt <= 3; attempt++ {
				if recovery == "close" {
					if err := s.Close(); !errors.Is(err, errStopPending) {
						t.Fatalf("Close did not report failed stop: %v", err)
					}
				} else {
					s.Disconnect()
				}
				waitForService(t, func() bool { return s.GetStatus().State == "Error" && r.child.kills.Load() == attempt })
				st := s.GetStatus()
				if st.SessionID == "" || st.ProcessState != "stopping" || strings.Contains(st.Message, "private") {
					t.Fatalf("lost stop ownership: %+v", st)
				}
				if _, err := s.Connect(p.ID); err == nil {
					t.Fatal("connected while stop failed")
				}
				if replacement, err := NewService(s.store.dir); err == nil {
					replacement.Close()
					t.Fatal("released store while child owned")
				}
				time.Sleep(5 * time.Millisecond)
				if r.child.kills.Load() != attempt {
					t.Fatal("automatic stop retry")
				}
			}
			if recovery == "close" {
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				s.Disconnect()
				waitState(t, s, "stopped")
			}
			if r.child.kills.Load() != 4 || r.child.waits.Load() != 1 {
				t.Fatalf("kills=%d waits=%d", r.child.kills.Load(), r.child.waits.Load())
			}
			if s.GetStatus().SessionID != "" {
				t.Fatal("completed cleanup retained session ID")
			}
			assertNoRuntime(t, s)
		})
	}
}
