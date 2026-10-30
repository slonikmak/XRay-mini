package core

import (
	"context"
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
		if os.Args[2] == "-test" {
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
	runner := execRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = runner.Test(ctx, binary, config); e != nil {
		t.Fatalf("helper preflight: %v", e)
	}
	runCtx, stop := context.WithCancel(context.Background())
	defer stop()
	child, e := runner.Start(runCtx, binary, config)
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
