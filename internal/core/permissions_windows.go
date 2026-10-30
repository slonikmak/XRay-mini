//go:build windows

package core

import (
	"errors"
	"syscall"
	"unsafe"
)

var advapi32 = syscall.NewLazyDLL("advapi32.dll")
var convertSDDL = advapi32.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
var setFileSecurity = advapi32.NewProc("SetFileSecurityW")

// Replace inherited ACLs with inheritable full access for only the current user and SYSTEM.
func secureDirectory(path string) error {
	token, e := syscall.OpenCurrentProcessToken()
	if e != nil {
		return e
	}
	defer token.Close()
	user, e := token.GetTokenUser()
	if e != nil {
		return e
	}
	sid, e := user.User.Sid.String()
	if e != nil {
		return e
	}
	sddl, e := syscall.UTF16PtrFromString("D:P(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;SY)")
	if e != nil {
		return e
	}
	var descriptor uintptr
	ok, _, _ := convertSDDL.Call(uintptr(unsafe.Pointer(sddl)), 1, uintptr(unsafe.Pointer(&descriptor)), 0)
	if ok == 0 {
		return errors.New("cannot create private DACL")
	}
	defer localFree.Call(descriptor)
	name, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return e
	}
	ok, _, _ = setFileSecurity.Call(uintptr(unsafe.Pointer(name)), 0x80000004, descriptor)
	if ok == 0 {
		return errors.New("cannot apply private DACL")
	}
	return nil
}
