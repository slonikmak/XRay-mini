package main

import (
	"embed"
	"log"
	"sync"
	"sync/atomic"

	"github.com/slonikmak/XRay-mini/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func init() { application.RegisterEvent[core.Status]("connection:state-changed") }

type trayAction uint8

const (
	trayNoAction trayAction = iota
	trayDisconnect
	trayCancel
	trayRetryStop
)

type trayMenuState struct {
	status string
	action string
	kind   trayAction
}

// Tray text uses only fixed words. Status messages and profile data may contain
// sensitive connection details and must never become native menu or tooltip text.
func trayMenuForStatus(status core.Status) trayMenuState {
	state := trayMenuState{status: "Starting"}
	switch status.State {
	case "Disconnected", "Connecting", "Connected", "Degraded", "Disconnecting", "Error":
		state.status = status.State
	}
	if (status.State == "Connecting" || status.State == "Connected") && status.ProcessState == "running" && status.NetworkHealth == "unhealthy" {
		state.status = "Check failed"
	}
	if status.SessionID == "" {
		return state
	}
	switch status.State {
	case "Connecting":
		state.action, state.kind = "Cancel connection", trayCancel
	case "Connected", "Degraded":
		state.action, state.kind = "Disconnect", trayDisconnect
	case "Error":
		state.status = "Error (session retained)"
		state.action, state.kind = "Retry stop", trayRetryStop
	}
	return state
}

func main() {
	icons, err := trayIcons(trayIcon)
	if err != nil {
		log.Fatal("Unable to prepare system tray icons: ", err)
	}
	desktop := NewApp()
	trayUpdates := make(chan struct{}, 1)
	windowShown := make(chan struct{})
	done := make(chan struct{})
	var shownOnce, doneOnce sync.Once
	var exiting atomic.Bool
	desktop.trayStatusChanged = func() {
		select {
		case trayUpdates <- struct{}{}:
		default:
		}
	}

	app := application.New(application.Options{
		Name: "Xray Mini", Description: "Local Xray proxy client",
		Services: []application.Service{application.NewService(desktop)},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: false},
	})
	app.OnShutdown(func() {
		exiting.Store(true)
		doneOnce.Do(func() { close(done) })
	})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Xray Mini", Width: 380, Height: 640, MinWidth: 360, MinHeight: 560,
		BackgroundColour: application.NewRGB(13, 17, 25), URL: "/",
		Windows: application.WindowsWindow{Theme: application.Dark},
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !exiting.Load() {
			window.Hide()
			e.Cancel()
		}
	})
	window.OnWindowEvent(events.Common.WindowShow, func(_ *application.WindowEvent) {
		shownOnce.Do(func() { close(windowShown) })
	})
	openWindow := func() {
		application.InvokeAsync(func() { window.Show().Focus() })
	}

	tray := app.SystemTray.New()
	currentIcon := trayIconIdle
	tray.SetIcon(icons[currentIcon])
	// Wails starts services before it initializes pending trays and windows.
	// Set the initial icon from that snapshot before the native tray exists.
	desktop.trayInitialStatus = func(status core.Status) {
		currentIcon = trayIconForStatus(status)
		tray.SetIcon(icons[currentIcon])
	}
	tray.SetTooltip("Xray Mini — Starting")
	tray.OnClick(openWindow)
	menu := app.NewMenu()
	menu.Add("Starting").SetEnabled(false)
	menu.Add("Open").OnClick(func(_ *application.Context) { openWindow() })
	stopItem := menu.Add("Disconnect").SetHidden(true)
	stopItem.OnClick(func(_ *application.Context) {
		status := desktop.GetStatus()
		if status.SessionID == "" || trayMenuForStatus(status).kind == trayNoAction {
			return
		}
		go func(sessionID string) {
			if desktop.GetStatus().SessionID == sessionID {
				desktop.Disconnect()
			}
		}(status.SessionID)
	})
	menu.AddSeparator()
	menu.Add("Exit").OnClick(func(_ *application.Context) {
		exiting.Store(true)
		go app.Quit() // Wails invokes ServiceShutdown during normal application cleanup.
	})
	tray.SetMenu(menu)

	updateTray := func() {
		status := desktop.GetStatus()
		state := trayMenuForStatus(status)
		if nextIcon := trayIconForStatus(status); nextIcon != currentIcon {
			tray.SetIcon(icons[nextIcon])
			currentIcon = nextIcon
		}
		menu.ItemAt(0).SetLabel(state.status)
		stopItem.SetLabel(state.action)
		stopItem.SetHidden(state.kind == trayNoAction)
		stopItem.SetEnabled(state.kind != trayNoAction)
		menu.Update()
		tray.SetTooltip("Xray Mini — " + state.status)
	}
	go func() {
		select {
		case <-windowShown:
		case <-done:
			return
		}
		for {
			application.InvokeAsync(func() {
				if !exiting.Load() {
					updateTray()
				}
			})
			select {
			case <-trayUpdates:
			case <-done:
				return
			}
		}
	}()
	if err := app.Run(); err != nil {
		log.Fatal("Unable to launch desktop window: ", err)
	}
	doneOnce.Do(func() { close(done) })
}
