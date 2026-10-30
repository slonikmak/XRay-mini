//go:build windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var createJob = kernel32.NewProc("CreateJobObjectW")
var setJobInfo = kernel32.NewProc("SetInformationJobObject")
var assignJob = kernel32.NewProc("AssignProcessToJobObject")

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

func configureCommand(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
func containProcess(p *os.Process) (func(), error) {
	job, _, _ := createJob.Call(0, 0)
	if job == 0 {
		return nil, errors.New("cannot create process job")
	}
	closeJob := func() { _ = syscall.CloseHandle(syscall.Handle(job)) }
	info := jobExtendedLimits{}
	info.BasicLimitInformation.LimitFlags = 0x2000 // KILL_ON_JOB_CLOSE
	ok, _, _ := setJobInfo.Call(job, 9, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	if ok == 0 {
		closeJob()
		return nil, errors.New("cannot configure process job")
	}
	handle, e := syscall.OpenProcess(0x0001|0x0100, false, uint32(p.Pid))
	if e != nil {
		closeJob()
		return nil, errors.New("cannot open child process")
	}
	defer syscall.CloseHandle(handle)
	ok, _, _ = assignJob.Call(job, uintptr(handle))
	if ok == 0 {
		closeJob()
		return nil, errors.New("cannot contain child process")
	}
	return closeJob, nil
}
