//go:build windows

package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Run on a real Windows 10/11 host. The intermediate parent intentionally exits
// without Wait, Kill, context cancellation or any defer-based cleanup.
func TestWindowsJobKillsChildOnParentExit(t *testing.T) {
	for _, stage := range []string{"preflight", "runtime"} {
		t.Run(stage, func(t *testing.T) {
			binary, e := os.Executable()
			if e != nil {
				t.Fatal(e)
			}
			cmd := exec.Command(binary, "-test.run=^TestWindowsJobCrashParentHelper$")
			config := filepath.Join(t.TempDir(), "config.json")
			if e = os.WriteFile(config, []byte("{}"), 0600); e != nil {
				t.Fatal(e)
			}
			cmd.Env = append(os.Environ(), "XRAY_MINI_JOB_CRASH_HELPER="+stage, "XRAY_MINI_JOB_TEST_CONFIG="+config)
			output, e := cmd.Output()
			if e != nil {
				t.Fatalf("parent helper: %v", e)
			}
			pid, e := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32)
			if e != nil {
				t.Fatalf("child PID: %q", output)
			}
			handle, e := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
			if e == syscall.Errno(87) {
				return
			} // Already reaped.
			if e != nil {
				t.Fatal(e)
			}
			defer syscall.CloseHandle(handle)
			result, e := syscall.WaitForSingleObject(handle, uint32((5*time.Second)/time.Millisecond))
			if e != nil || result != syscall.WAIT_OBJECT_0 {
				t.Fatalf("child survived parent exit: wait=%d err=%v", result, e)
			}
		})
	}
}
func TestWindowsJobCrashParentHelper(t *testing.T) {
	stage := os.Getenv("XRAY_MINI_JOB_CRASH_HELPER")
	if stage == "" {
		t.Skip("subprocess helper")
	}
	binary, e := os.Executable()
	if e != nil {
		os.Exit(20)
	}
	config := os.Getenv("XRAY_MINI_JOB_TEST_CONFIG")
	os.Setenv("XRAY_MINI_TEST_CHILD", "1")
	os.Setenv("XRAY_MINI_TEST_HOLD_PREFLIGHT", "1")
	args := []string{"run", "-config", config}
	if stage == "preflight" {
		args = []string{"run", "-test", "-config", config}
	}
	child, e := startContained(context.Background(), binary, args)
	if e != nil {
		os.Exit(23)
	}
	c := child.(*windowsChild)
	pid, _, _ := kernel32.NewProc("GetProcessId").Call(uintptr(c.process))
	if pid == 0 {
		os.Exit(24)
	}
	// A crash must terminate the child even before any Wait registration.
	fmt.Println(pid)
	os.Exit(0)
}

func TestWindowsExecutableLockDeniesReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xray.exe")
	if err := os.WriteFile(path, []byte("not executable; lock test only"), 0600); err != nil {
		t.Fatal(err)
	}
	release, err := lockExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600); err == nil {
		f.Close()
		t.Fatal("verified executable remained writable")
	}
	if err := os.Remove(path); err == nil {
		t.Fatal("verified executable remained removable")
	}
}
