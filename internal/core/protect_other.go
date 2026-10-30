//go:build !windows

package core

// Development only: non-Windows storage is plaintext with 0600 files / 0700 directory.
func protect(b []byte) ([]byte, error)   { return b, nil }
func unprotect(b []byte) ([]byte, error) { return b, nil }
