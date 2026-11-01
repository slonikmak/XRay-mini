package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Status struct {
	State         string `json:"state"`
	ProcessState  string `json:"processState"`
	NetworkHealth string `json:"networkHealth"`
	ProfileID     string `json:"profileId"`
	SessionID     string `json:"sessionId"`
	Message       string `json:"message"`
	Revision      uint64 `json:"revision"`
	LastCheckedAt string `json:"lastCheckedAt"`
	XrayVersion   string `json:"xrayVersion"`
}
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

const (
	sessionShutdownTimeout      = 5 * time.Second
	notificationShutdownTimeout = 250 * time.Millisecond
)

var (
	errCleanupPending        = errors.New("temporary configuration cleanup failed; remove the block and use Retry stop, Disconnect or Close")
	errCleanupRetryRequested = errors.New("temporary configuration cleanup retry requested; connect again after cleanup finishes")
	errStopPending           = errors.New("Xray stop is not confirmed; process and storage ownership are retained")
	errNotificationPending   = errors.New("status notification delivery is incomplete; a callback is still running")
)

type session struct {
	ctx            context.Context
	cancel         context.CancelFunc
	id, profileID  string
	retryCleanup   chan struct{}
	retryStop      chan struct{}
	cleanupBlocked bool // written only by Service.loop
	exited         bool // process exit is terminal, independently of cleanup
	exitFailure    string
}
type command struct {
	kind, id string
	reply    chan commandResult
}
type commandResult struct {
	status Status
	err    error
}
type workerEvent struct {
	sessionID, kind, message string
	ok                       bool
	at                       time.Time
}
type Service struct {
	store             *profileStore
	storeMu           sync.Mutex
	commands          chan command
	events            chan workerEvent
	notifications     chan Status
	loopDone          chan struct{}
	publisherDone     chan struct{}
	closeMu           sync.Mutex
	publisherStopOnce sync.Once
	publisherStop     chan struct{}
	cleanupDone       chan struct{}
	closeErrors       chan error
	closed            atomic.Bool
	snapshot          atomic.Value
	logSnapshot       atomic.Value
	callbackMu        sync.RWMutex
	callback          func(Status)
	runner            processRunner
	probe             func(context.Context) bool
	ports             func() error
	healthInterval    time.Duration
}

func NewService(dataDir string) (*Service, error) {
	st, e := newStore(dataDir)
	if e != nil {
		return nil, e
	}
	s := &Service{store: st, commands: make(chan command), events: make(chan workerEvent, 32), notifications: make(chan Status, 1), loopDone: make(chan struct{}), publisherDone: make(chan struct{}), publisherStop: make(chan struct{}), cleanupDone: make(chan struct{}), closeErrors: make(chan error, 1), runner: execRunner{}, probe: probeNetwork, ports: checkPorts, healthInterval: 30 * time.Second}
	s.snapshot.Store(Status{State: "Disconnected", ProcessState: "stopped", NetworkHealth: "unknown", Message: "Local proxy is stopped", XrayVersion: XrayVersion})
	s.logSnapshot.Store([]LogEntry{})
	go s.publishLoop()
	go s.loop()
	go func() {
		<-s.loopDone
		s.storeMu.Lock()
		_ = s.store.close()
		s.storeMu.Unlock()
		close(s.cleanupDone)
	}()
	return s, nil
}
func (s *Service) SetEventCallback(cb func(Status)) {
	s.callbackMu.Lock()
	s.callback = cb
	s.callbackMu.Unlock()
}
func (s *Service) publishLoop() {
	defer close(s.publisherDone)
	for {
		// A callback cannot be forcibly canceled. Only this publisher invokes it,
		// so at most one callback can outlive Close; no replacements are spawned.
		select {
		case <-s.publisherStop:
			return
		default:
		}
		var st Status
		select {
		case <-s.publisherStop:
			return
		case next, ok := <-s.notifications:
			if !ok {
				return
			}
			st = next
		}
		s.callbackMu.RLock()
		cb := s.callback
		s.callbackMu.RUnlock()
		if cb != nil {
			cb(st)
		}
	}
}
func (s *Service) request(kind, id string) commandResult {
	reply := make(chan commandResult, 1)
	select {
	case s.commands <- command{kind, id, reply}:
	case <-s.loopDone:
		return commandResult{s.GetStatus(), errors.New("service is closed")}
	}
	select {
	case r := <-reply:
		return r
	case <-s.loopDone:
		select {
		case r := <-reply:
			return r
		default:
			return commandResult{s.GetStatus(), errors.New("service is closed")}
		}
	}
}
func (s *Service) GetStatus() Status { return s.snapshot.Load().(Status) }
func (s *Service) GetRecentLogs(limit int) []LogEntry {
	logs := s.logSnapshot.Load().([]LogEntry)
	if limit < 0 {
		limit = 0
	}
	if limit > len(logs) {
		limit = len(logs)
	}
	return append([]LogEntry{}, logs[len(logs)-limit:]...)
}
func (s *Service) GetLogs() []LogEntry                               { return s.GetRecentLogs(200) }
func (s *Service) PreviewImport(input string) (ImportPreview, error) { return PreviewImport(input) }
func (s *Service) ListProfiles() ([]Profile, error) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	if s.closed.Load() {
		return nil, errors.New("service is closed")
	}
	return s.store.list()
}
func (s *Service) ImportProfile(name, input string) (Profile, error) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	if s.closed.Load() {
		return Profile{}, errors.New("service is closed")
	}
	return s.store.add(name, []byte(input))
}
func (s *Service) RenameProfile(id, name string) (Profile, error) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	if s.closed.Load() {
		return Profile{}, errors.New("service is closed")
	}
	return s.store.rename(id, name)
}
func (s *Service) DeleteProfile(id string) error {
	if !validID(id) {
		return errors.New("invalid profile ID")
	}
	r := s.request("reserve-delete", id)
	if r.err != nil {
		return r.err
	}
	defer s.request("release-delete", id)
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	if s.closed.Load() {
		return errors.New("service is closed")
	}
	return s.store.remove(id)
}
func (s *Service) Connect(id string) (Status, error) {
	r := s.request("connect", id)
	return r.status, r.err
}
func (s *Service) Disconnect() Status { return s.request("disconnect", "").status }

// Close waits at most five seconds for session cleanup, then at most 250ms for
// notification delivery. A session cleanup error retains ownership; another Close explicitly
// retries a failed process stop or file cleanup. A blocked callback can outlive Close until it returns.
func (s *Service) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	s.closed.Store(true)
	select {
	case <-s.closeErrors:
	default:
	}
	s.request("close", "")
	var result error
	timer := time.NewTimer(sessionShutdownTimeout)
	defer timer.Stop()
	select {
	case <-s.cleanupDone:
	case result = <-s.closeErrors:
	case <-timer.C:
		result = errStopPending
	}
	if result != nil {
		s.publisherStopOnce.Do(func() { close(s.publisherStop) })
	}
	notificationTimer := time.NewTimer(notificationShutdownTimeout)
	defer notificationTimer.Stop()
	select {
	case <-s.publisherDone:
	case <-notificationTimer.C:
		s.publisherStopOnce.Do(func() { close(s.publisherStop) })
		result = errors.Join(result, errNotificationPending)
	}
	return result
}

// This goroutine is the only writer of session state, revisions and diagnostics.
// File, DPAPI, process and network work never runs in this loop.
func (s *Service) loop() {
	defer close(s.loopDone)
	defer close(s.notifications)
	status := s.GetStatus()
	logs := []LogEntry{}
	var active *session
	closing := false
	failures := 0
	deleting := map[string]bool{}
	retryCleanup := func() {
		if active.cleanupBlocked {
			active.cleanupBlocked = false
			active.retryCleanup <- struct{}{}
		}
	}
	reportCloseError := func(err error) {
		if closing {
			select {
			case s.closeErrors <- err:
			default:
			}
		}
	}
	publish := func(state, process, health, message string) {
		status.State = state
		status.ProcessState = process
		status.NetworkHealth = health
		status.Message = message
		status.Revision++
		level := "info"
		if state == "Error" {
			level = "error"
		} else if health == "unhealthy" {
			level = "warning"
		}
		logs = append(logs, LogEntry{time.Now().UTC().Format(time.RFC3339Nano), level, message})
		if len(logs) > 200 {
			logs = logs[len(logs)-200:]
		}
		s.logSnapshot.Store(append([]LogEntry{}, logs...))
		s.snapshot.Store(status)
		select {
		case s.notifications <- status:
		default:
			select {
			case <-s.notifications:
			default:
			}
			select {
			case s.notifications <- status:
			default:
			}
		}
	}
	requestStop := func() {
		repeated := active.ctx.Err() != nil
		active.cancel()
		retryCleanup()
		if repeated && !active.exited {
			// Retain an explicit repeat even if its command overtakes StopFailed.
			select {
			case active.retryStop <- struct{}{}:
			default:
			}
		}
		if !active.exited {
			publish("Disconnecting", "stopping", "unknown", "Stopping local proxy")
		}
	}
	for {
		select {
		case c := <-s.commands:
			var err error
			switch c.kind {
			case "connect":
				if closing || s.closed.Load() {
					err = errors.New("service is closed")
				} else if active != nil && active.cleanupBlocked {
					retryCleanup()
					err = errCleanupRetryRequested
					message := err.Error()
					if active.exitFailure != "" {
						message = active.exitFailure + "; " + message
					}
					publish("Error", "stopped", "unknown", message)
				} else if active != nil {
					err = errors.New("a session is active or still cleaning up")
				} else if !validID(c.id) {
					err = errors.New("invalid profile ID")
				} else if deleting[c.id] {
					err = errors.New("profile is being deleted")
				} else {
					var id [16]byte
					if _, e := rand.Read(id[:]); e != nil {
						err = errors.New("cannot create session ID")
					} else {
						ctx, cancel := context.WithCancel(context.Background())
						active = &session{ctx: ctx, cancel: cancel, id: hex.EncodeToString(id[:]), profileID: c.id, retryCleanup: make(chan struct{}, 1), retryStop: make(chan struct{}, 1)}
						status.ProfileID = c.id
						status.SessionID = active.id
						status.LastCheckedAt = ""
						failures = 0
						publish("Connecting", "preparing", "unknown", "Preparing local proxy")
						go s.run(active)
					}
				}
			case "disconnect":
				if active != nil {
					requestStop()
				}
			case "reserve-delete":
				if closing {
					err = errors.New("service is closed")
				} else if active != nil && active.profileID == c.id {
					err = errors.New("disconnect before deleting the active profile")
				} else if deleting[c.id] {
					err = errors.New("profile is being deleted")
				} else {
					deleting[c.id] = true
				}
			case "release-delete":
				delete(deleting, c.id)
			case "close":
				closing = true
				if active != nil {
					requestStop()
				}
			}
			c.reply <- commandResult{status, err}
			if closing && active == nil {
				return
			}
		case ev := <-s.events:
			if active == nil || ev.sessionID != active.id {
				continue
			}
			if ev.kind == "CleanupFailed" {
				active.cleanupBlocked = true
				message := errCleanupPending.Error()
				if active.exitFailure != "" {
					message = active.exitFailure + "; " + message
				}
				publish("Error", "stopped", "unknown", message)
				reportCloseError(errCleanupPending)
				continue
			}
			if ev.kind == "StopFailed" {
				if active.exited {
					continue
				}
				publish("Error", "stopping", "unknown", errStopPending.Error())
				reportCloseError(errStopPending)
				continue
			}
			if ev.kind == "StopCompleted" {
				active.cancel()
				if active.exitFailure != "" {
					ev.message = active.exitFailure
				}
				active = nil
				status.SessionID = ""
				if ev.message != "" {
					publish("Error", "error", "unknown", ev.message)
				} else {
					status.ProfileID = ""
					publish("Disconnected", "stopped", "unknown", "Local proxy is stopped")
				}
				if closing {
					return
				}
				continue
			}
			if ev.kind == "ProcessExited" {
				if !active.exited {
					active.exited = true
					if ev.ok {
						active.exitFailure = "Xray exited unexpectedly; no automatic restart was attempted"
					}
					active.cancel()
					if active.exitFailure != "" {
						publish("Error", "stopped", "unknown", active.exitFailure)
					} else {
						publish("Disconnecting", "stopped", "unknown", "Xray stopped; cleaning up session resources")
					}
				}
				continue
			}
			if (active.exited || active.ctx.Err() != nil) && ev.kind != "Diagnostic" {
				continue
			}
			switch ev.kind {
			case "Diagnostic":
				switch ev.message {
				case "xray-output", "xray-warning", "xray-error", "xray-stdout-observed", "xray-stderr-observed":
					logs = append(logs, LogEntry{ev.at.UTC().Format(time.RFC3339Nano), "info", ev.message})
					if len(logs) > 200 {
						logs = logs[len(logs)-200:]
					}
					s.logSnapshot.Store(append([]LogEntry{}, logs...))
				}
			case "ConfigValidated":
				publish("Connecting", "preparing", "unknown", "Xray configuration validated; network not yet checked")
			case "ProcessStarted":
				publish("Connecting", "running", "checking", "Xray is running; checking the local HTTP proxy")
			case "ProxyReady", "HealthFailed":
				status.LastCheckedAt = ev.at.UTC().Format(time.RFC3339Nano)
				if ev.ok {
					failures = 0
					publish("Connected", "running", "healthy", "HTTP proxy check passed; other destinations may differ")
				} else {
					failures++
					state := status.State
					if failures >= 2 {
						state = "Degraded"
					}
					publish(state, "running", "unhealthy", "HTTP proxy check failed; the control resource or network may be unavailable")
				}
			}
		}
	}
}
func (s *Service) event(run *session, kind, message string, ok bool) {
	s.events <- workerEvent{run.id, kind, message, ok, time.Now()}
}
func (s *Service) run(run *session) {
	failure := ""
	tempDir := ""
	defer func() {
		if tempDir != "" {
			for os.RemoveAll(tempDir) != nil {
				s.event(run, "CleanupFailed", "", false)
				// Exactly one attempt per explicit Disconnect/Close recovery request.
				<-run.retryCleanup
			}
		}
		s.event(run, "StopCompleted", failure, false)
	}()
	s.storeMu.Lock()
	p, e := s.store.read(run.profileID)
	s.storeMu.Unlock()
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
	if e = secureDirectory(tempDir); e != nil {
		failure = "Could not protect runtime directory"
		return
	}
	configPath := filepath.Join(tempDir, "config.json")
	if e = os.WriteFile(configPath, cfg, 0600); e != nil {
		failure = "Could not write runtime configuration"
		return
	}
	path := bundledRuntimePath()
	testCtx, cancel := context.WithTimeout(run.ctx, 15*time.Second)
	e = s.runner.Test(testCtx, path, configPath)
	cancel()
	if run.ctx.Err() != nil {
		return
	}
	if e != nil {
		failure = "Trusted Xray preflight failed: check bundled runtime integrity and profile"
		if errors.Is(e, errBundledRuntimeMissing) {
			failure = "Bundled Xray runtime is missing beside this app. Start bin/XrayMini.exe or use wails3 task dev; go run . is not supported."
		}
		return
	}
	s.event(run, "ConfigValidated", "", false)
	child, e := s.runner.Start(run.ctx, path, configPath)
	if e != nil {
		if run.ctx.Err() == nil {
			failure = "Could not start verified Xray inside its process container"
		}
		return
	}
	exited := make(chan error, 1)
	go func() {
		err := child.Wait()
		// Capture the exit fact before queueing: a later stop command may be
		// consumed first, but cannot change why this Wait completed.
		unexpected := run.ctx.Err() == nil
		s.event(run, "ProcessExited", "", unexpected)
		exited <- err
	}()
	seenDiagnostics := map[string]bool{}
	collectDiagnostics := func() {
		if d, ok := child.(interface{ Diagnostics() []string }); ok {
			for _, category := range d.Diagnostics() {
				if !seenDiagnostics[category] {
					seenDiagnostics[category] = true
					s.event(run, "Diagnostic", category, false)
				}
			}
		}
	}
	diagnosticTicker := time.NewTicker(time.Second)
	defer diagnosticTicker.Stop()
	defer collectDiagnostics()

	s.event(run, "ProcessStarted", "", false)
	healthCtx, healthCancel := context.WithCancel(run.ctx)
	healthDone := make(chan struct{})
	health := make(chan bool, 1)
	go func() {
		defer close(healthDone)
		ticker := time.NewTicker(s.healthInterval)
		defer ticker.Stop()
		for {
			if healthCtx.Err() != nil {
				return
			}
			probeCtx, stop := context.WithTimeout(healthCtx, 5*time.Second)
			ok := s.probe(probeCtx)
			stop()
			select {
			case health <- ok:
			case <-healthCtx.Done():
				return
			}
			select {
			case <-ticker.C:
			case <-healthCtx.Done():
				return
			}
		}
	}()
	defer func() { healthCancel(); <-healthDone }()
	for {
		select {
		case <-run.ctx.Done():
			for {
				select {
				case <-exited:
					return
				default:
				}
				if err := child.Kill(); err == nil {
					<-exited
					return
				}
				s.event(run, "StopFailed", "", false)
				// A failed attempt waits for actual exit or one explicit retry.
				// Do not re-select the already canceled session context.
				select {
				case <-exited:
					return
				case <-run.retryStop:
				}
			}
		case <-exited:
			if run.ctx.Err() == nil {
				failure = "Xray exited unexpectedly; no automatic restart was attempted"
			}
			return
		case <-diagnosticTicker.C:
			collectDiagnostics()
		case ok := <-health:
			select {
			case <-exited:
				if run.ctx.Err() == nil {
					failure = "Xray exited unexpectedly"
				}
				return
			default:
			}
			if ok {
				s.event(run, "ProxyReady", "", true)
			} else {
				s.event(run, "HealthFailed", "", false)
			}
		}
	}
}
