//go:build windows

package core

import "syscall"

// A non-shared handle is a kernel-owned lock: crash releases it, without stale PID guesses.
func acquireStoreLock(path string) (func() error, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return func() error { return syscall.CloseHandle(h) }, nil
}
