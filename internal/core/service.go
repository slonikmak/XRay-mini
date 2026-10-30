package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Status struct {
	ProcessState  string `json:"processState"`
	NetworkHealth string `json:"networkHealth"`
	ProfileID     string `json:"profileId"`
	Message       string `json:"message"`
	Revision      uint64 `json:"revision"`
}
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}
type session struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	id     string
}
type Service struct {
	mu             sync.Mutex
	store          *profileStore
	status         Status
	logs           []LogEntry
	active         *session
	closed         bool
	runner         processRunner
	probe          func(context.Context) bool
	ports          func() error
	callback       func(Status)
	healthInterval time.Duration
}

func NewService(dataDir string) (*Service, error) {
	store, e := newStore(dataDir)
	if e != nil {
		return nil, e
	}
	return &Service{store: store, status: Status{ProcessState: "stopped", NetworkHealth: "unknown", Message: "Local proxy is stopped"}, logs: []LogEntry{}, runner: execRunner{}, probe: probeNetwork, ports: checkPorts, healthInterval: 30 * time.Second}, nil
}
func (s *Service) SetEventCallback(callback func(Status)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callback = callback
}
func (s *Service) ListProfiles() ([]Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.list()
}
func (s *Service) ImportProfile(name, configJSON string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Profile{}, errors.New("service is closed")
	}
	return s.store.add(name, []byte(configJSON))
}
func (s *Service) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("service is closed")
	}
	if s.active != nil && s.active.id == id {
		return errors.New("disconnect before deleting the active profile")
	}
	return s.store.remove(id)
}
func (s *Service) GetStatus() Status { s.mu.Lock(); defer s.mu.Unlock(); return s.status }
func (s *Service) GetLogs() []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]LogEntry{}, s.logs...)
}

// Called under mu. Callbacks must tolerate asynchronous delivery; poll GetStatus for authoritative state.
func (s *Service) publish(state, health, id, message string) {
	s.status = Status{ProcessState: state, NetworkHealth: health, ProfileID: id, Message: message, Revision: s.status.Revision + 1}
	s.logs = append(s.logs, LogEntry{time.Now().UTC().Format(time.RFC3339), "info", message})
	if len(s.logs) > 200 {
		s.logs = append([]LogEntry{}, s.logs[len(s.logs)-200:]...)
	}
	if s.callback != nil {
		callback, status := s.callback, s.status
		go callback(status)
	}
}
func (s *Service) Connect(id, xrayPath string) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.status, errors.New("service is closed")
	}
	if s.active != nil {
		return s.status, errors.New("a session is active or still cleaning up")
	}
	if !validID(id) {
		return s.status, errors.New("invalid profile ID")
	}
	if e := validateBinary(xrayPath); e != nil {
		return s.status, e
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &session{ctx, cancel, make(chan struct{}), id}
	s.active = run
	s.publish("preparing", "unknown", id, "Preparing local proxy")
	go s.run(run, xrayPath)
	return s.status, nil
}
func (s *Service) Disconnect() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		s.active.cancel()
		if s.status.ProcessState != "stopping" {
			s.publish("stopping", "unknown", s.active.id, "Stopping local proxy")
		}
	}
	return s.status
}
func (s *Service) Close() error {
	s.mu.Lock()
	s.closed = true
	run := s.active
	if run != nil {
		run.cancel()
	}
	s.mu.Unlock()
	if run != nil {
		<-run.done
	}
	return nil
}
func (s *Service) setRunning(run *session, health, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == run && run.ctx.Err() == nil {
		s.publish("running", health, run.id, message)
	}
}
func (s *Service) run(run *session, path string) {
	defer run.cancel()
	failure := ""
	tempDir := ""
	defer func() {
		if tempDir != "" {
			if e := os.RemoveAll(tempDir); e != nil {
				failure = "Stopped, but temporary configuration cleanup failed; remove the runtime directory manually"
			}
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.active = nil
		if failure != "" {
			s.publish("error", "unknown", run.id, failure)
		} else {
			s.publish("stopped", "unknown", "", "Local proxy is stopped")
		}
		close(run.done)
	}()
	// Disk and process preparation occur outside the service mutex, so Disconnect remains responsive.
	p, e := s.store.read(run.id)
	if e != nil {
		failure = "Could not read the selected profile"
		return
	}
	cfg, e := PrepareConfig(p.Config)
	if e != nil {
		failure = "Stored configuration failed validation"
		return
	}
	if run.ctx.Err() != nil {
		return
	}
	if e = s.ports(); e != nil {
		failure = "Local proxy port 10808 or 10809 is already in use"
		return
	}
	tempDir, e = os.MkdirTemp(s.store.dir, "runtime-")
	if e != nil {
		failure = "Could not create private runtime directory"
		return
	}
	configPath := filepath.Join(tempDir, "config.json")
	if e = os.WriteFile(configPath, cfg, 0600); e != nil {
		failure = "Could not write runtime configuration"
		return
	}
	testCtx, cancel := context.WithTimeout(run.ctx, 15*time.Second)
	e = s.runner.Test(testCtx, path, configPath)
	cancel()
	if run.ctx.Err() != nil {
		return
	}
	if e != nil {
		failure = "Xray configuration preflight failed or timed out; verify the trusted core version and profile"
		return
	}
	child, e := s.runner.Start(run.ctx, path, configPath)
	if e != nil {
		if run.ctx.Err() == nil {
			failure = "Could not start or contain Xray"
		}
		return
	}
	exited := make(chan error, 1)
	go func() { exited <- child.Wait() }()
	s.setRunning(run, "checking", "Xray is running; checking the local HTTP proxy")
	// Only this owner consumes exit and waits for cleanup. Probe cancellation is session-scoped.
	healthCtx, healthCancel := context.WithCancel(run.ctx)
	healthDone := make(chan struct{})
	defer func() { healthCancel(); <-healthDone }()
	health := make(chan bool, 1)
	go func() {
		defer close(healthDone)
		for {
			ok := false
			for attempt := 0; attempt < 3; attempt++ {
				if healthCtx.Err() != nil {
					return
				}
				if s.probe(healthCtx) {
					ok = true
					break
				}
				select {
				case <-healthCtx.Done():
					return
				case <-time.After(500 * time.Millisecond):
				}
			}
			select {
			case health <- ok:
			case <-healthCtx.Done():
				return
			}
			select {
			case <-time.After(s.healthInterval):
			case <-healthCtx.Done():
				return
			}
		}
	}()
	for {
		select {
		case <-run.ctx.Done():
			_ = child.Kill()
			<-exited
			return
		case <-exited:
			if run.ctx.Err() == nil {
				failure = "Xray exited unexpectedly; no automatic restart was attempted"
			}
			return
		case ok := <-health:
			// An already-exited process wins over a positive health result.
			select {
			case <-exited:
				if run.ctx.Err() == nil {
					failure = "Xray exited unexpectedly"
				}
				return
			default:
			}
			if ok {
				s.setRunning(run, "healthy", "HTTP proxy check passed; this does not verify all traffic is tunneled")
			} else {
				s.setRunning(run, "unhealthy", "Xray is running, but the HTTP proxy check failed")
			}
		}
	}
}
