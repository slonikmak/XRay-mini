package core

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectUnverifiedExecutable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "xray.exe")
	if err := os.WriteFile(p, []byte("not an executable"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := validateBinary(p); err == nil {
		t.Fatal("arbitrary executable accepted")
	}
	if err := verifyExecutableDigest(p, XrayExecutableSHA256); err == nil {
		t.Fatal("tampered executable accepted")
	}
	if err := verifyExecutableDigest(p+"missing", XrayExecutableSHA256); err == nil {
		t.Fatal("missing executable accepted")
	}
}
func TestExactExecutableDigest(t *testing.T) {
	data := []byte("digest test only; never executed")
	sum := sha256.Sum256(data)
	p := filepath.Join(t.TempDir(), "fixture")
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyExecutableDigest(p, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(data, '!'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyExecutableDigest(p, hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("mutation not detected")
	}
}
