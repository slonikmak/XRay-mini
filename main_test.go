package main

import (
	"strings"
	"testing"

	"github.com/slonikmak/XRay-mini/internal/core"
)

func TestTrayMenuUsesOwnedSessionState(t *testing.T) {
	tests := []struct {
		name   string
		status core.Status
		want   trayMenuState
	}{
		{"idle", core.Status{State: "Disconnected"}, trayMenuState{status: "Disconnected"}},
		{"startup error", core.Status{State: "Error", Message: "private startup detail"}, trayMenuState{status: "Error"}},
		{"connecting", core.Status{State: "Connecting", ProcessState: "preparing", SessionID: "owned"}, trayMenuState{"Connecting", "Cancel connection", trayCancel}},
		{"connected", core.Status{State: "Connected", ProcessState: "running", SessionID: "owned"}, trayMenuState{"Connected", "Disconnect", trayDisconnect}},
		{"degraded", core.Status{State: "Degraded", ProcessState: "running", SessionID: "owned"}, trayMenuState{"Degraded", "Disconnect", trayDisconnect}},
		{"stopping", core.Status{State: "Disconnecting", ProcessState: "stopping", SessionID: "owned"}, trayMenuState{status: "Disconnecting"}},
		{"failed stop retains child", core.Status{State: "Error", ProcessState: "stopping", SessionID: "owned"}, trayMenuState{"Error (session retained)", "Retry stop", trayRetryStop}},
		{"failed cleanup retains session", core.Status{State: "Error", ProcessState: "stopped", SessionID: "owned"}, trayMenuState{"Error (session retained)", "Retry stop", trayRetryStop}},
		{"finished error has no session", core.Status{State: "Error", ProcessState: "error", ProfileID: "old"}, trayMenuState{status: "Error"}},
		{"inconsistent connected without session", core.Status{State: "Connected"}, trayMenuState{status: "Connected"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trayMenuForStatus(tt.status)
			if got != tt.want {
				t.Fatalf("trayMenuForStatus() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTrayMenuDoesNotExposeStatusDetails(t *testing.T) {
	secret := "secret-profile-and-url"
	status := core.Status{State: "Error", SessionID: "owned", ProfileID: secret, Message: secret}
	menu := trayMenuForStatus(status)
	if strings.Contains(menu.status, secret) || strings.Contains(menu.action, secret) {
		t.Fatalf("tray menu exposed profile or message: %+v", menu)
	}
}
