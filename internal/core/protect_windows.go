//go:build windows

package core

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	Size uint32
	Data *byte
}

var crypt32 = syscall.NewLazyDLL("crypt32.dll")
var cryptProtect = crypt32.NewProc("CryptProtectData")
var cryptUnprotect = crypt32.NewProc("CryptUnprotectData")
var localFree = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")

func crypt(b []byte, decrypt bool) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("empty protected data")
	}
	input := dataBlob{uint32(len(b)), &b[0]}
	var output dataBlob
	proc := cryptProtect
	if decrypt {
		proc = cryptUnprotect
	}
	// CRYPTPROTECT_UI_FORBIDDEN, current-user scope (no LOCAL_MACHINE flag).
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&output)))
	runtime.KeepAlive(b)
	if ok == 0 {
		return nil, errors.New("DPAPI operation failed")
	}
	defer localFree.Call(uintptr(unsafe.Pointer(output.Data)))
	result := append([]byte(nil), unsafe.Slice(output.Data, int(output.Size))...)
	return result, nil
}
func protect(b []byte) ([]byte, error)   { return crypt(b, false) }
func unprotect(b []byte) ([]byte, error) { return crypt(b, true) }
