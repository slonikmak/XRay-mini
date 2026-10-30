package core

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This test-only child mimics Xray's CLI shape. It never opens a network socket.
func TestMain(m *testing.M) {
	if os.Getenv("XRAY_MINI_TEST_CHILD") == "1" {
		if len(os.Args) < 4 || os.Args[1] != "run" {
			os.Exit(10)
		}
		config := os.Args[len(os.Args)-1]
		if _, e := os.ReadFile(config); e != nil {
			os.Exit(11)
		}
		if os.Args[2] == "-test" && os.Getenv("XRAY_MINI_TEST_HOLD_PREFLIGHT") != "1" {
			os.Exit(0)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestExecRunnerWithHelperProcess(t *testing.T) {
	t.Setenv("XRAY_MINI_TEST_CHILD", "1")
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	config := filepath.Join(t.TempDir(), "config.json")
	if e = os.WriteFile(config, []byte(testConfig), 0600); e != nil {
		t.Fatal(e)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	preflight, e := startContained(ctx, binary, []string{"run", "-test", "-config", config})
	if e == nil {
		e = preflight.Wait()
	}
	if e != nil {
		t.Fatalf("helper preflight: %v", e)
	}
	runCtx, stop := context.WithCancel(context.Background())
	defer stop()
	child, e := startContained(runCtx, binary, []string{"run", "-config", config})
	if e != nil {
		t.Fatalf("helper start: %v", e)
	}
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = child.Kill()
		<-done
		t.Fatal("helper did not stop after cancellation")
	}
}

func TestOutputDrainNeverRetainsChildText(t *testing.T) {
	var drain outputDrain
	secret := bytes.Repeat([]byte("secret-token=https://host/?password=private\n"), 100000)
	drain.start(io.NopCloser(bytes.NewReader(secret)), 0)
	drain.start(io.NopCloser(bytes.NewReader(secret)), 1)
	drain.wg.Wait()
	got := drain.Diagnostics()
	if len(got) != 2 || got[0] != "xray-stdout-observed" || got[1] != "xray-stderr-observed" {
		t.Fatalf("unsafe categories: %q", got)
	}
}
func TestContainedStartRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	if child, e := startContained(ctx, binary, []string{"run"}); e == nil {
		child.Kill()
		child.Wait()
		t.Fatal("started canceled process")
	}
}
