package main

import (
	"context"
	"errors"
	"github.com/slonikmak/XRay-mini/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
	"os"
	"path/filepath"
	"sync/atomic"
)

// App exposes only application operations, never raw profile contents or core output.
type App struct {
	service      *core.Service
	startupError error
	closing      atomic.Bool
}

func NewApp() *App { return &App{} }
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		a.startupError = errors.New("Cannot locate the user configuration directory")
		return a.startupError
	}
	a.service, err = core.NewService(filepath.Join(dir, "XrayMini"))
	if err != nil {
		// Core startup errors are fixed, sanitized categories (including instance ownership).
		a.startupError = err
		return a.startupError
	}
	a.service.SetEventCallback(func(status core.Status) {
		if !a.closing.Load() {
			application.Get().Event.Emit("connection:state-changed", status)
		}
	})
	return nil
}
func (a *App) ServiceShutdown() error {
	a.closing.Store(true)
	if a.service != nil {
		return a.service.Close()
	}
	return nil
}
func (a *App) ready() error {
	if a.startupError != nil {
		return a.startupError
	}
	if a.service == nil {
		return errors.New("Desktop backend is not ready")
	}
	return nil
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
	return a.service.ImportProfile(name, configJSON)
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
		return core.Status{State: "Error", ProcessState: "error", NetworkHealth: "unknown", Message: "Desktop backend unavailable"}
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
