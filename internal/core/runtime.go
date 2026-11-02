package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// These pins are derived from the official, unmodified Windows x64 release.
const XrayVersion = "26.10.31"
const XrayExecutableSHA256 = "d5b558191cccb1e92a5772b68b8cab67717b26f634b0f15153215f65ae982a7e"

var errBundledRuntimeMissing = errors.New("bundled Xray runtime is missing")

func bundledRuntimePath() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(executable), "runtime", "xray.exe")
}

// Validate before every invocation, including preflight. There is no UI path override.
func validateBinary(path string) error {
	if runtime.GOOS != "windows" {
		return errors.New("the bundled Xray runtime requires Windows x64")
	}
	if runtime.GOARCH != "amd64" {
		return errors.New("the bundled Xray runtime requires Windows x64")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != bundledRuntimePath() {
		return errors.New("only the bundled Xray runtime is permitted")
	}
	return verifyExecutableDigest(path, XrayExecutableSHA256)
}

func verifyExecutableDigest(path, expected string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return errBundledRuntimeMissing
	}
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("bundled Xray executable is missing or is not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return errors.New("cannot read bundled Xray executable")
	}
	defer f.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, f); err != nil {
		return errors.New("cannot verify bundled Xray executable")
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return errors.New("bundled Xray integrity check failed; reinstall the verified package")
	}
	return nil
}
