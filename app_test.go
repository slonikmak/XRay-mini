//go:build windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/slonikmak/XRay-mini/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const facadeSyntheticLink = "vless://11111111-1111-4111-8111-111111111111@example.invalid:443?type=xhttp&security=tls&path=%2Fsynthetic#Candidate"

func subscriptionTestApp(t *testing.T) *App {
	t.Helper()
	// Native DPAPI/store tests always use a private APPDATA, never personal data.
	t.Setenv("APPDATA", t.TempDir())
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	service, err := core.NewService(filepath.Join(dir, "XrayMini"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{service: service, lifetime: ctx, stopLifetime: cancel}
	t.Cleanup(func() {
		if err := a.ServiceShutdown(); err != nil {
			t.Error(err)
		}
	})
	return a
}

func TestSubscriptionFacadeMetadataExplicitImportAndExpiry(t *testing.T) {
	a := subscriptionTestApp(t)
	metadata, err := a.PreviewSubscription(facadeSyntheticLink)
	if err != nil || len(metadata) != 1 {
		t.Fatalf("preview: %v", err)
	}
	serialized, _ := json.Marshal(metadata)
	for _, secret := range []string{"vless://", "11111111-1111", "example.invalid", "outbounds", "ConfigJSON", "source"} {
		if strings.Contains(string(serialized), secret) {
			t.Fatal("secret in metadata DTO")
		}
	}
	if len(metadata[0].ID) != 32 || metadata[0].Transport != "XHTTP/TLS" {
		t.Fatal("opaque metadata contract")
	}
	if profiles, _ := a.ListProfiles(); len(profiles) != 0 {
		t.Fatal("preview persisted a profile")
	}
	if _, err := a.ImportSubscriptionCandidate("old-id", "Saved"); err == nil {
		t.Fatal("unknown ID accepted")
	}
	profile, err := a.ImportSubscriptionCandidate(metadata[0].ID, "Saved")
	if err != nil {
		t.Fatal(err)
	}
	if profiles, _ := a.ListProfiles(); len(profiles) != 1 || profiles[0].ID != profile.ID {
		t.Fatal("selected profile not persisted exactly once")
	}
	data, err := os.ReadFile(filepath.Join(os.Getenv("APPDATA"), "XrayMini", profile.ID+".profile"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "11111111-1111-4111") || strings.Contains(string(data), "example.invalid") {
		t.Error("profile credentials persisted in plaintext")
	}
	if a.candidates != nil {
		t.Fatal("successful import retained transient config")
	}
	if _, err := a.ImportSubscriptionCandidate(metadata[0].ID, "Duplicate"); err == nil {
		t.Fatal("replayed preview imported")
	}
	metadata, err = a.PreviewSubscription(facadeSyntheticLink)
	if err != nil {
		t.Fatal(err)
	}
	a.previewMu.Lock()
	a.previewExpires = time.Now().Add(-time.Second)
	a.previewMu.Unlock()
	if _, err := a.ImportSubscriptionCandidate(metadata[0].ID, "Expired"); err == nil {
		t.Fatal("expired preview imported")
	}
	metadata, err = a.PreviewSubscription(facadeSyntheticLink)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.PreviewSubscription(facadeSyntheticLink); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ImportSubscriptionCandidate(metadata[0].ID, "Replaced"); err == nil {
		t.Fatal("replaced ID accepted")
	}
	if err := a.ClearSubscriptionPreview(); err != nil || a.candidates != nil {
		t.Fatal("clear retained preview")
	}
}

func TestSubscriptionFacadeRejectsLateCanceledAndShutdownResults(t *testing.T) {
	for _, action := range []string{"cancel", "replace", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			a := subscriptionTestApp(t)
			started, release := make(chan struct{}), make(chan struct{})
			result := make(chan error, 1)
			go func() {
				_, err := a.previewSubscription("synthetic", func(ctx context.Context, _ string) ([]core.SubscriptionProfile, error) {
					close(started)
					<-release
					return []core.SubscriptionProfile{{Name: "Late", Transport: "XHTTP/TLS", ConfigJSON: "secret"}}, nil
				})
				result <- err
			}()
			<-started
			switch action {
			case "cancel":
				_ = a.ClearSubscriptionPreview()
			case "replace":
				if _, err := a.PreviewSubscription(facadeSyntheticLink); err != nil {
					t.Fatal(err)
				}
			case "shutdown":
				if err := a.ServiceShutdown(); err != nil {
					t.Fatal(err)
				}
			}
			close(release)
			if err := <-result; err == nil {
				t.Fatal("late result accepted")
			}
			a.previewMu.Lock()
			for _, candidate := range a.candidates {
				if candidate.Name == "Late" {
					t.Error("late result restored preview")
				}
			}
			a.previewMu.Unlock()
			if action == "shutdown" {
				if _, err := a.PreviewSubscription(facadeSyntheticLink); err == nil {
					t.Fatal("closing facade accepted load")
				}
			}
		})
	}
}

func TestSubscriptionFacadeDelayedSaveDoesNotBlockPreviewOrShutdown(t *testing.T) {
	for _, action := range []string{"clear", "replace", "shutdown", "save failure"} {
		t.Run(action, func(t *testing.T) {
			a := subscriptionTestApp(t)
			metadata, err := a.PreviewSubscription(facadeSyntheticLink)
			if err != nil {
				t.Fatal(err)
			}
			started, release := make(chan struct{}), make(chan struct{})
			defer close(release)
			saved := make(chan error, 1)
			go func() {
				_, err := a.importSubscriptionCandidate(metadata[0].ID, "Saved", func(name, config string) (core.Profile, error) {
					close(started)
					<-release
					if action == "save failure" {
						return core.Profile{}, errors.New("synthetic save failure")
					}
					return core.Profile{ID: "saved", Name: name}, nil
				})
				saved <- err
			}()
			<-started
			var replacement []SubscriptionCandidate
			actionDone := make(chan error, 1)
			go func() {
				switch action {
				case "clear", "save failure":
					actionDone <- a.ClearSubscriptionPreview()
				case "replace":
					var err error
					replacement, err = a.PreviewSubscription(facadeSyntheticLink)
					actionDone <- err
				case "shutdown":
					actionDone <- a.ServiceShutdown()
				}
			}()
			select {
			case err := <-actionDone:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("clear/replacement/shutdown waited for slow profile persistence")
			}
			if _, err := a.ImportSubscriptionCandidate(metadata[0].ID, "Duplicate"); err == nil {
				t.Fatal("reserved candidate remained reusable")
			}
			// Release the blocked persistence call without letting deferred cleanup
			// close the channel twice.
			release <- struct{}{}
			if err := <-saved; (err != nil) != (action == "save failure") {
				t.Fatalf("unexpected save result: %v", err)
			}
			a.previewMu.Lock()
			defer a.previewMu.Unlock()
			if action == "replace" {
				if _, ok := a.candidates[replacement[0].ID]; !ok {
					t.Fatal("completed old save cleared a newer preview")
				}
			} else if a.candidates != nil {
				t.Fatal("save completion resurrected consumed preview")
			}
		})
	}
}

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
