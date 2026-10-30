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

const testConfig = `{"outbounds":[{"protocol":"freedom"}]}`

type fakeChild struct {
	done  chan error
	once  sync.Once
	waits atomic.Int32
}

func (c *fakeChild) Wait() error { c.waits.Add(1); return <-c.done }
func (c *fakeChild) Kill() error { c.once.Do(func() { c.done <- nil }); return nil }

type fakeRunner struct {
	preflight func(context.Context, string) error
	child     *fakeChild
	starts    atomic.Int32
}

func (r *fakeRunner) Test(ctx context.Context, _ string, config string) error {
	if r.preflight != nil {
		return r.preflight(ctx, config)
	}
	return nil
}
func (r *fakeRunner) Start(ctx context.Context, _, _ string) (childProcess, error) {
	r.starts.Add(1)
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
	s, p, path, r := fixture(t)
	entered := make(chan struct{})
	r.preflight = func(ctx context.Context, _ string) error { close(entered); <-ctx.Done(); return ctx.Err() }
	if status, e := s.Connect(p.ID, path); e != nil || status.ProcessState != "preparing" {
		t.Fatalf("%+v %v", status, e)
	}
	<-entered
	if _, e := s.Connect(p.ID, path); e == nil {
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
	s, p, path, r := fixture(t)
	if _, e := s.Connect(p.ID, path); e != nil {
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
	s, p, path, r := fixture(t)
	entered := make(chan struct{})
	r.preflight = func(ctx context.Context, _ string) error { close(entered); <-ctx.Done(); return ctx.Err() }
	_, _ = s.Connect(p.ID, path)
	<-entered
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Connect(p.ID, path); e == nil {
		t.Fatal("closed service connected")
	}
	if _, e := s.ImportProfile("new", testConfig); e == nil {
		t.Fatal("closed service imported")
	}
	assertNoRuntime(t, s)
}
func TestPreflightErrorsNeverExposeCoreDiagnostics(t *testing.T) {
	s, p, path, r := fixture(t)
	r.preflight = func(context.Context, string) error { return errors.New("secret-id secret-server private-path") }
	_, _ = s.Connect(p.ID, path)
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
func TestUnexpectedExitIsNotHealthy(t *testing.T) {
	s, p, path, r := fixture(t)
	probeEntered := make(chan struct{})
	s.probe = func(ctx context.Context) bool { close(probeEntered); <-ctx.Done(); return false }
	_, _ = s.Connect(p.ID, path)
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
	s, p, path, r := fixture(t)
	s.ports = func() error { return errors.New("busy") }
	_, _ = s.Connect(p.ID, path)
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
	s, p, path, r := fixture(t)
	var previous uint64
	for i := 0; i < 20; i++ {
		r.child = &fakeChild{done: make(chan error, 1)}
		if _, e := s.Connect(p.ID, path); e != nil {
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
	s, p, path, _ := fixture(t)
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
	_, _ = s.Connect(p.ID, path)
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
