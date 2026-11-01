//go:build windows

package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var terminateJob = kernel32.NewProc("TerminateJobObject")
var createJob = kernel32.NewProc("CreateJobObjectW")
var setJobInfo = kernel32.NewProc("SetInformationJobObject")
var initAttributes = kernel32.NewProc("InitializeProcThreadAttributeList")
var updateAttributes = kernel32.NewProc("UpdateProcThreadAttribute")
var deleteAttributes = kernel32.NewProc("DeleteProcThreadAttributeList")
var createProcess = kernel32.NewProc("CreateProcessW")

type jobBasicLimits struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}
type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}
type jobExtendedLimits struct {
	BasicLimitInformation jobBasicLimits
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}
type startupInfoEx struct {
	syscall.StartupInfo
	attributes unsafe.Pointer
}
type windowsChild struct {
	mu      sync.Mutex
	process syscall.Handle
	job     syscall.Handle
	done    chan struct{}
	once    sync.Once
	err     error
	drain   outputDrain
}

func (c *windowsChild) Kill() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.process == 0 {
		return nil
	}
	// The job handle has its own termination rights, independent of the process
	// handle. Keep both handles until the single Wait confirms process exit.
	ok, _, _ := terminateJob.Call(uintptr(c.job), 1)
	if ok == 0 {
		return errors.New("cannot terminate Xray process job")
	}
	return nil
}
func (c *windowsChild) Wait() error {
	c.once.Do(func() {
		_, e := syscall.WaitForSingleObject(c.process, syscall.INFINITE)
		var code uint32
		if e == nil {
			e = syscall.GetExitCodeProcess(c.process, &code)
		}
		c.mu.Lock()
		syscall.CloseHandle(c.process)
		c.process = 0
		syscall.CloseHandle(c.job)
		c.job = 0
		c.mu.Unlock()
		close(c.done)
		c.drain.wg.Wait()
		if e != nil {
			c.err = errors.New("cannot wait for Xray process")
		} else if code != 0 {
			c.err = errors.New("Xray exited with a nonzero status")
		}
	})
	return c.err
}
func (c *windowsChild) Diagnostics() []string { return c.drain.Diagnostics() }

// Both preflight and live Xray enter the kill-on-close job atomically during
// CreateProcessW. No post-creation assignment or suspended-process gap exists.
// Microsoft: UpdateProcThreadAttribute, PROC_THREAD_ATTRIBUTE_JOB_LIST (Win10+).
func startContained(ctx context.Context, path string, args []string) (childProcess, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	job, _, _ := createJob.Call(0, 0)
	if job == 0 {
		return nil, errors.New("cannot create process job")
	}
	owned := false
	defer func() {
		if !owned {
			syscall.CloseHandle(syscall.Handle(job))
		}
	}()
	limits := jobExtendedLimits{}
	limits.BasicLimitInformation.LimitFlags = 0x2000
	ok, _, _ := setJobInfo.Call(job, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits))
	if ok == 0 {
		return nil, errors.New("cannot configure process job")
	}
	var size uintptr
	initAttributes.Call(0, 2, 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return nil, errors.New("cannot size process attributes")
	}
	storage := make([]byte, size)
	attrs := unsafe.Pointer(&storage[0])
	ok, _, _ = initAttributes.Call(uintptr(attrs), 2, 0, uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return nil, errors.New("cannot initialize process attributes")
	}
	jobHandle := syscall.Handle(job)
	var handles []syscall.Handle
	defer func() {
		deleteAttributes.Call(uintptr(attrs))
		runtime.KeepAlive(storage)
		runtime.KeepAlive(handles)
		runtime.KeepAlive(jobHandle)
	}()
	ok, _, _ = updateAttributes.Call(uintptr(attrs), 0, 0x0002000D, uintptr(unsafe.Pointer(&jobHandle)), unsafe.Sizeof(jobHandle), 0, 0)
	if ok == 0 {
		return nil, errors.New("atomic process job assignment is unavailable")
	}
	// Only these three handles may be inherited; notably the job is NOT inherited.
	sa := syscall.SecurityAttributes{Length: uint32(unsafe.Sizeof(syscall.SecurityAttributes{})), InheritHandle: 1}
	var outR, outW, errR, errW syscall.Handle
	if e := syscall.CreatePipe(&outR, &outW, &sa, 0); e != nil {
		return nil, errors.New("cannot create output pipe")
	}
	defer syscall.CloseHandle(outW)
	if e := syscall.CreatePipe(&errR, &errW, &sa, 0); e != nil {
		syscall.CloseHandle(outR)
		return nil, errors.New("cannot create error pipe")
	}
	defer syscall.CloseHandle(errW)
	readOwned := false
	defer func() {
		if !readOwned {
			syscall.CloseHandle(outR)
			syscall.CloseHandle(errR)
		}
	}()
	if syscall.SetHandleInformation(outR, syscall.HANDLE_FLAG_INHERIT, 0) != nil || syscall.SetHandleInformation(errR, syscall.HANDLE_FLAG_INHERIT, 0) != nil {
		return nil, errors.New("cannot restrict pipe inheritance")
	}
	nul, _ := syscall.UTF16PtrFromString("NUL")
	stdin, e := syscall.CreateFile(nul, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, &sa, syscall.OPEN_EXISTING, 0, 0)
	if e != nil {
		return nil, errors.New("cannot open null input")
	}
	defer syscall.CloseHandle(stdin)
	handles = []syscall.Handle{stdin, outW, errW}
	ok, _, _ = updateAttributes.Call(uintptr(attrs), 0, 0x00020002, uintptr(unsafe.Pointer(&handles[0])), uintptr(len(handles))*unsafe.Sizeof(handles[0]), 0, 0)
	if ok == 0 {
		return nil, errors.New("cannot restrict child handle list")
	}
	app, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return nil, errors.New("invalid executable path")
	}
	parts := []string{syscall.EscapeArg(path)}
	for _, arg := range args {
		parts = append(parts, syscall.EscapeArg(arg))
	}
	command, e := syscall.UTF16PtrFromString(strings.Join(parts, " "))
	if e != nil {
		return nil, errors.New("invalid process arguments")
	}
	dir, e := syscall.UTF16PtrFromString(filepath.Dir(path))
	if e != nil {
		return nil, errors.New("invalid executable directory")
	}
	si := startupInfoEx{}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = syscall.STARTF_USESTDHANDLES
	si.StdInput = stdin
	si.StdOutput = outW
	si.StdErr = errW
	si.attributes = attrs
	var pi syscall.ProcessInformation
	// EXTENDED_STARTUPINFO_PRESENT | CREATE_NO_WINDOW; environment inherited.
	ok, _, _ = createProcess.Call(uintptr(unsafe.Pointer(app)), uintptr(unsafe.Pointer(command)), 0, 0, 1, 0x00080000|0x08000000, 0, uintptr(unsafe.Pointer(dir)), uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	runtime.KeepAlive(storage)
	runtime.KeepAlive(handles)
	runtime.KeepAlive(jobHandle)
	if ok == 0 {
		return nil, errors.New("cannot create Xray in process job")
	}
	syscall.CloseHandle(pi.Thread)
	c := &windowsChild{process: pi.Process, job: jobHandle, done: make(chan struct{})}
	owned = true
	readOwned = true
	c.drain.start(os.NewFile(uintptr(outR), "xray-stdout"), 0)
	c.drain.start(os.NewFile(uintptr(errR), "xray-stderr"), 1)
	go func() {
		select {
		case <-ctx.Done():
			_ = c.Kill()
		case <-c.done:
		}
	}()
	return c, nil
}

func lockExecutable(path string) (func(), error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, errors.New("invalid executable path")
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errBundledRuntimeMissing
		}
		return nil, errors.New("cannot lock bundled Xray for verified execution")
	}
	return func() { syscall.CloseHandle(handle) }, nil
}
