package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCrashRecoveryAndActiveInstance(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "runtime-old-crashed-session")
	if err := os.Mkdir(old, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "config.json"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := newStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("stale runtime not cleaned")
	}
	live := filepath.Join(dir, "runtime-live")
	if err := os.Mkdir(live, 0700); err != nil {
		t.Fatal(err)
	}
	if second, err := newStore(dir); err == nil {
		second.close()
		t.Fatal("second owner accepted")
	}
	if _, err := os.Stat(live); err != nil {
		t.Fatal("live runtime deleted by second instance")
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := newStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Fatal("released owner's runtime not recovered")
	}
}

func TestRenameProfilePreservesIdentity(t *testing.T) {
	store, err := newStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	profile, err := store.add("original", []byte(`{"outbounds":[{"protocol":"vless","settings":{"vnext":[{"address":"example.org","port":443,"users":[{"id":"00000000-0000-4000-8000-000000000000","encryption":"none"}]}]},"streamSettings":{"network":"raw","security":"reality","realitySettings":{"serverName":"example.org","publicKey":"example"}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := store.rename(profile.ID, " new name ")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.ID != profile.ID || renamed.CreatedAt != profile.CreatedAt || renamed.Name != "new name" {
		t.Fatal("rename changed identity")
	}
	persisted, err := store.read(profile.ID)
	if err != nil || persisted.Name != "new name" {
		t.Fatal("rename did not persist")
	}
	if _, err := store.rename(profile.ID, "  "); err == nil {
		t.Fatal("empty name accepted")
	}
	if _, err := store.rename("../bad", "name"); err == nil {
		t.Fatal("invalid ID accepted")
	}
}

func TestStoreLockOwnerCrash(t *testing.T) {
	if dir := os.Getenv("XRAY_MINI_LOCK_HELPER"); dir != "" {
		store, err := newStore(dir)
		if err != nil {
			os.Exit(2)
		}
		_ = store // Deliberately never Close: parent kills this process.
		if os.Mkdir(filepath.Join(dir, "runtime-helper"), 0700) != nil {
			os.Exit(3)
		}
		if os.WriteFile(filepath.Join(dir, "ready"), []byte("ready"), 0600) != nil {
			os.Exit(4)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestStoreLockOwnerCrash$")
	cmd.Env = append(os.Environ(), "XRAY_MINI_LOCK_HELPER="+dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if second, err := newStore(dir); err == nil {
		second.close()
		t.Fatal("live subprocess lock ignored")
	}
	if _, err := os.Stat(filepath.Join(dir, "runtime-helper")); err != nil {
		t.Fatal("live subprocess runtime removed")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	waited = true
	recovered, err := newStore(dir)
	if err != nil {
		t.Fatal("crashed owner's lock not released:", err)
	}
	defer recovered.close()
	if _, err := os.Stat(filepath.Join(dir, "runtime-helper")); !os.IsNotExist(err) {
		t.Fatal("crashed owner runtime not cleaned")
	}
}
