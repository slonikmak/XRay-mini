package main

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"

	"github.com/slonikmak/XRay-mini/internal/core"
)

func TestTrayIconFollowsConnectionHealth(t *testing.T) {
	tests := []struct {
		name   string
		status core.Status
		want   trayIconKind
	}{
		{"startup", core.Status{}, trayIconIdle},
		{"disconnected", core.Status{State: "Disconnected", ProcessState: "stopped"}, trayIconIdle},
		{"connecting", core.Status{State: "Connecting", ProcessState: "running", NetworkHealth: "checking"}, trayIconIdle},
		{"connected and checked", core.Status{State: "Connected", ProcessState: "running", NetworkHealth: "healthy"}, trayIconConnected},
		{"first failed check", core.Status{State: "Connected", ProcessState: "running", NetworkHealth: "unhealthy"}, trayIconWarning},
		{"startup check failed", core.Status{State: "Connecting", ProcessState: "running", NetworkHealth: "unhealthy"}, trayIconWarning},
		{"degraded", core.Status{State: "Degraded", ProcessState: "running", NetworkHealth: "unhealthy"}, trayIconWarning},
		{"recovered", core.Status{State: "Connected", ProcessState: "running", NetworkHealth: "healthy"}, trayIconConnected},
		{"disconnecting", core.Status{State: "Disconnecting", ProcessState: "stopping", NetworkHealth: "healthy"}, trayIconIdle},
		{"error with retained session", core.Status{State: "Error", ProcessState: "stopping", SessionID: "owned"}, trayIconWarning},
		{"unknown network", core.Status{State: "Connected", ProcessState: "running", NetworkHealth: "unknown"}, trayIconIdle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trayIconForStatus(tt.status); got != tt.want {
				t.Fatalf("trayIconForStatus(%+v) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestTrayIconsPreserveMarkAndUseDistinctBackgrounds(t *testing.T) {
	icons, err := trayIcons(trayIcon)
	if err != nil {
		t.Fatal(err)
	}
	backgrounds := [3]color.NRGBA{
		{R: 151, G: 163, B: 177, A: 255},
		{R: 129, G: 235, B: 205, A: 255},
		{R: 226, G: 189, B: 119, A: 255},
	}
	for kind, data := range icons {
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("icon %d: %v", kind, err)
		}
		if got := color.NRGBAModel.Convert(img.At(128, 20)).(color.NRGBA); got != backgrounds[kind] {
			t.Errorf("icon %d background = %v, want %v", kind, got, backgrounds[kind])
		}
		if got := color.NRGBAModel.Convert(img.At(128, 128)).(color.NRGBA); got != (color.NRGBA{R: 16, G: 42, B: 37, A: 255}) {
			t.Errorf("icon %d lost its mark: %v", kind, got)
		}
		if _, _, _, alpha := img.At(0, 0).RGBA(); alpha != 0 {
			t.Errorf("icon %d lost transparency", kind)
		}
	}
	if _, err := trayIcons([]byte("not a PNG")); err == nil {
		t.Fatal("invalid icon was accepted")
	}
}
