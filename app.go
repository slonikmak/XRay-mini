package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/slonikmak/XRay-mini/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// App exposes only application operations, never raw profile contents or core output.
type App struct {
	service           *core.Service
	startupError      error
	trayInitialStatus func(core.Status)
	trayStatusChanged func()
	closing           atomic.Bool
	previewMu         sync.Mutex
	lifetime          context.Context
	stopLifetime      context.CancelFunc
	previewCancel     context.CancelFunc
	previewTimer      *time.Timer
	previewGeneration uint64
	previewExpires    time.Time
	candidates        map[string]core.SubscriptionProfile
}

// SubscriptionCandidate is metadata only. Source URLs and configurations never
// cross the bridge in the preview response.
type SubscriptionCandidate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Transport string `json:"transport"`
}

func NewApp() *App { return &App{} }
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	defer func() {
		if a.trayInitialStatus != nil {
			a.trayInitialStatus(a.GetStatus())
		}
		a.notifyTrayStatus()
	}()
	a.previewMu.Lock()
	a.lifetime, a.stopLifetime = context.WithCancel(ctx)
	a.previewMu.Unlock()
	dir, err := os.UserConfigDir()
	if err != nil {
		a.startupError = errors.New("Cannot locate the user configuration directory")
		return nil // Keep the façade/window alive to display the safe failure.
	}
	a.service, err = core.NewService(filepath.Join(dir, "XrayMini"))
	if err != nil {
		// Core startup errors are fixed, sanitized categories (including instance ownership).
		a.startupError = err
		return nil
	}
	a.service.SetEventCallback(func(status core.Status) {
		if !a.closing.Load() {
			a.notifyTrayStatus()
			application.Get().Event.Emit("connection:state-changed", status)
		}
	})
	return nil
}
func (a *App) notifyTrayStatus() {
	if a.trayStatusChanged != nil {
		a.trayStatusChanged()
	}
}
func (a *App) ServiceShutdown() error {
	a.closing.Store(true)
	a.previewMu.Lock()
	a.clearSubscriptionLocked()
	if a.stopLifetime != nil {
		a.stopLifetime()
	}
	a.previewMu.Unlock()
	if a.service != nil {
		return a.service.Close()
	}
	return nil
}
func (a *App) ready() error {
	if a.closing.Load() {
		return errors.New("Desktop backend is closing")
	}
	if a.startupError != nil {
		return a.startupError
	}
	if a.service == nil {
		return errors.New("Desktop backend is not ready")
	}
	return nil
}

func (a *App) clearSubscriptionLocked() {
	a.previewGeneration++
	if a.previewCancel != nil {
		a.previewCancel()
		a.previewCancel = nil
	}
	if a.previewTimer != nil {
		a.previewTimer.Stop()
		a.previewTimer = nil
	}
	a.candidates = nil
	a.previewExpires = time.Time{}
}

func (a *App) ClearSubscriptionPreview() error {
	a.previewMu.Lock()
	defer a.previewMu.Unlock()
	a.clearSubscriptionLocked()
	return nil
}

func (a *App) PreviewSubscription(source string) ([]SubscriptionCandidate, error) {
	return a.previewSubscription(source, core.LoadSubscription)
}

func (a *App) previewSubscription(source string, load func(context.Context, string) ([]core.SubscriptionProfile, error)) ([]SubscriptionCandidate, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	a.previewMu.Lock()
	a.clearSubscriptionLocked()
	if a.closing.Load() {
		a.previewMu.Unlock()
		return nil, errors.New("Desktop backend is closing")
	}
	lifetime := a.lifetime
	if lifetime == nil {
		lifetime = context.Background()
	}
	ctx, cancel := context.WithCancel(lifetime)
	a.previewCancel = cancel
	generation := a.previewGeneration
	a.previewMu.Unlock()
	defer cancel()
	profiles, err := load(ctx, source)
	a.previewMu.Lock()
	defer a.previewMu.Unlock()
	if a.closing.Load() || generation != a.previewGeneration || ctx.Err() != nil {
		return nil, errors.New("Subscription preview canceled or replaced")
	}
	a.previewCancel = nil
	if err != nil {
		return nil, err
	}
	metadata := make([]SubscriptionCandidate, 0, len(profiles))
	candidates := make(map[string]core.SubscriptionProfile, len(profiles))
	for _, profile := range profiles {
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil {
			return nil, errors.New("Cannot create subscription preview")
		}
		id := hex.EncodeToString(token[:])
		candidates[id] = profile
		metadata = append(metadata, SubscriptionCandidate{ID: id, Name: profile.Name, Transport: profile.Transport})
	}
	a.candidates = candidates
	a.previewExpires = time.Now().Add(5 * time.Minute)
	a.previewTimer = time.AfterFunc(5*time.Minute, func() {
		a.previewMu.Lock()
		defer a.previewMu.Unlock()
		if generation == a.previewGeneration {
			a.clearSubscriptionLocked()
		}
	})
	return metadata, nil
}

func (a *App) ImportSubscriptionCandidate(id, name string) (core.Profile, error) {
	return a.importSubscriptionCandidate(id, name, a.service.ImportProfile)
}

func (a *App) importSubscriptionCandidate(id, name string, save func(string, string) (core.Profile, error)) (core.Profile, error) {
	if err := a.ready(); err != nil {
		return core.Profile{}, err
	}
	a.previewMu.Lock()
	if a.closing.Load() {
		a.previewMu.Unlock()
		return core.Profile{}, errors.New("Desktop backend is closing")
	}
	if !time.Now().Before(a.previewExpires) {
		a.clearSubscriptionLocked()
	}
	candidate, ok := a.candidates[id]
	if !ok {
		a.previewMu.Unlock()
		return core.Profile{}, errors.New("Subscription preview expired or changed; load it again")
	}
	// Consume before persistence so IDs remain single use even if saving fails.
	// Clear/shutdown and replacement previews must not wait on disk or DPAPI.
	a.clearSubscriptionLocked()
	a.previewMu.Unlock()
	return save(name, candidate.ConfigJSON)
}
func (a *App) ListProfiles() ([]core.Profile, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.service.ListProfiles()
}
func (a *App) ImportProfile(name, configJSON string) (core.Profile, error) {
	if err := a.ready(); err != nil {
		return core.Profile{}, err
	}
	profile, err := a.service.ImportProfile(name, configJSON)
	if err == nil {
		_ = a.ClearSubscriptionPreview()
	}
	return profile, err
}
func (a *App) DeleteProfile(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.service.DeleteProfile(id)
}
func (a *App) Connect(id string) (core.Status, error) {
	if err := a.ready(); err != nil {
		return core.Status{}, err
	}
	return a.service.Connect(id)
}
func (a *App) Disconnect() core.Status {
	if a.service == nil {
		return a.GetStatus()
	}
	return a.service.Disconnect()
}
func (a *App) GetStatus() core.Status {
	if a.service == nil {
		message := "Desktop backend unavailable"
		if a.startupError != nil {
			message = a.startupError.Error()
		}
		return core.Status{State: "Error", ProcessState: "error", NetworkHealth: "unknown", Message: message, XrayVersion: core.XrayVersion}
	}
	return a.service.GetStatus()
}
func (a *App) GetRecentLogs(limit int) []core.LogEntry {
	if a.service == nil {
		return []core.LogEntry{}
	}
	return a.service.GetRecentLogs(limit)
}

func (a *App) RenameProfile(id, name string) (core.Profile, error) {
	if err := a.ready(); err != nil {
		return core.Profile{}, err
	}
	return a.service.RenameProfile(id, name)
}
func (a *App) PreviewImport(configJSON string) (core.ImportPreview, error) {
	if err := a.ready(); err != nil {
		return core.ImportPreview{}, err
	}
	return core.PreviewImport(configJSON)
}
