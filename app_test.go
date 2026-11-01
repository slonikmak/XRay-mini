//go:build windows

package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/slonikmak/XRay-mini/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestStartupFailureKeepsSafeStatusAndRejectsOperations(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APPDATA", dataDir)
	owner, err := core.NewService(filepath.Join(dataDir, "XrayMini"))
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	app := NewApp()
	t.Cleanup(func() {
		if err := app.ServiceShutdown(); err != nil {
			t.Error(err)
		}
	})
	if err := app.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("safe startup failure aborted the desktop: %v", err)
	}
	status := app.GetStatus()
	if status.State != "Error" || status.Message != "another Xray Mini instance owns this profile directory" || status.XrayVersion != core.XrayVersion {
		t.Fatalf("startup cause lost: %+v", status)
	}
	if _, err := app.ListProfiles(); err == nil {
		t.Fatal("unavailable backend listed profiles")
	}
	if _, err := app.Connect("00000000000000000000000000000000"); err == nil {
		t.Fatal("unavailable backend accepted Connect")
	}
	if err := app.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
}
