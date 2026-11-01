//go:build windows

package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestWindowsMissingRuntimeIsClassifiedBeforeChildCreation(t *testing.T) {
	for _, relative := range []string{"xray.exe", filepath.Join("runtime", "xray.exe")} {
		path := filepath.Join(t.TempDir(), "private-location", relative)
		child, err := startVerifiedContained(context.Background(), path, []string{"run", "-test"})
		if child != nil || !errors.Is(err, errBundledRuntimeMissing) {
			t.Fatalf("missing runtime did not produce a safe category: %v", err)
		}
		if strings.Contains(err.Error(), "private-location") {
			t.Fatal("runtime error exposed its path")
		}
	}
}

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

// A real deny-delete handle exercises RemoveAll failure, rather than replacing
// filesystem cleanup with a fake. The worker owns this exact runtime file.
func TestWindowsRuntimeCleanupFailureRetainsAdmissionAndAllowsExplicitRetry(t *testing.T) {
	for _, recovery := range []string{"disconnect", "connect", "close"} {
		t.Run(recovery, func(t *testing.T) {
			s, p, _, r := fixture(t)
			var handle syscall.Handle
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { syscall.CloseHandle(handle) }) }
			t.Cleanup(release)
			r.preflight = func(_ context.Context, config string) error {
				name, err := syscall.UTF16PtrFromString(config)
				if err != nil {
					return err
				}
				handle, err = syscall.CreateFile(name, syscall.GENERIC_READ,
					syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, 0, 0)
				return err
			}
			if _, err := s.Connect(p.ID); err != nil {
				t.Fatal(err)
			}
			waitState(t, s, "running")
			s.Disconnect()
			deadline := time.Now().Add(2 * time.Second)
			for s.GetStatus().Message != errCleanupPending.Error() && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			status := s.GetStatus()
			if status.State != "Error" || status.ProcessState != "stopped" || status.Message != errCleanupPending.Error() {
				t.Fatalf("cleanup failure not safely reported: %+v", status)
			}
			previousRevision := status.Revision
			if retryStatus, err := s.Connect(p.ID); !errors.Is(err, errCleanupRetryRequested) ||
				retryStatus.Message != errCleanupRetryRequested.Error() || r.starts.Load() != 1 {
				t.Fatalf("Connect did not request cleanup only: %+v %v", retryStatus, err)
			}
			deadline = time.Now().Add(2 * time.Second)
			for s.GetStatus().Message != errCleanupPending.Error() && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			status = s.GetStatus()
			if status.Message != errCleanupPending.Error() || status.Revision <= previousRevision {
				t.Fatalf("explicit Connect retry did not report deletion refusal: %+v", status)
			}
			if err := s.DeleteProfile(p.ID); err == nil {
				t.Fatal("failed-cleanup profile ownership dropped")
			}
			time.Sleep(25 * time.Millisecond)
			if s.GetStatus().Revision != status.Revision {
				t.Fatal("cleanup is retrying without an explicit request")
			}
			if recovery == "close" {
				if err := s.Close(); !errors.Is(err, errCleanupPending) {
					t.Fatalf("Close cleanup result: %v", err)
				}
				if replacement, err := NewService(s.store.dir); err == nil {
					replacement.Close()
					t.Fatal("Close released lock before cleanup")
				}
			}
			release()
			if recovery == "close" {
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				if recovery == "connect" {
					if _, err := s.Connect(p.ID); !errors.Is(err, errCleanupRetryRequested) || r.starts.Load() != 1 {
						t.Fatalf("Connect started a child instead of retrying cleanup: %v", err)
					}
				} else {
					s.Disconnect() // documented explicit, single cleanup retry
				}
				waitState(t, s, "stopped")
				deadline = time.Now().Add(2 * time.Second)
				for s.GetStatus().State != "Disconnected" && time.Now().Before(deadline) {
					time.Sleep(time.Millisecond)
				}
				if s.GetStatus().State != "Disconnected" {
					t.Fatal("retry did not finish cleanup")
				}
				r.preflight = nil
				r.child = &fakeChild{done: make(chan error, 1)}
				if _, err := s.Connect(p.ID); err != nil {
					t.Fatal(err)
				}
				waitState(t, s, "running")
				s.Disconnect()
				waitState(t, s, "stopped")
			}
			assertNoRuntime(t, s)
		})
	}
}

func TestWindowsJobTerminationDoesNotRequireProcessTerminateRight(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XRAY_MINI_TEST_CHILD", "1")
	t.Setenv("XRAY_MINI_TEST_HOLD_PREFLIGHT", "1")
	config := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(config, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	child, err := startContained(context.Background(), binary, []string{"run", "-config", config})
	if err != nil {
		t.Fatal(err)
	}
	c := child.(*windowsChild)
	// Job termination and the single Wait remain available on every assertion path.
	t.Cleanup(func() { _ = c.Kill(); _ = c.Wait() })
	pid, _, _ := kernel32.NewProc("GetProcessId").Call(uintptr(c.process))
	limited, err := syscall.OpenProcess(syscall.SYNCHRONIZE|0x1000, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.TerminateProcess(limited, 1); err != syscall.ERROR_ACCESS_DENIED {
		syscall.CloseHandle(limited)
		t.Fatalf("expected native direct-process access denial, got %v", err)
	}
	// Emulate loss of direct terminate access without mocking the process/job.
	syscall.CloseHandle(c.process)
	c.process = limited
	waited := make(chan error, 1)
	go func() { waited <- c.Wait() }()
	if err := c.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-waited: // Nonzero exit is expected for forced termination.
	case <-time.After(5 * time.Second):
		t.Fatal("native job termination did not unblock Wait")
	}
	if c.process != 0 || c.job != 0 {
		t.Fatal("Wait did not release native ownership")
	}
}
