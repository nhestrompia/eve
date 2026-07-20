//go:build windows

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32DLL  = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx   = kernel32DLL.NewProc("LockFileEx")
	unlockFileEx = kernel32DLL.NewProc("UnlockFileEx")
)

const (
	lockFileFailImmediately = 0x00000001
	lockFileExclusive       = 0x00000002
	errorLockViolation      = syscall.Errno(33)
	errorIOPending          = syscall.Errno(997)
)

func configureVerificationProcess(cmd *exec.Cmd) {
	cmd.WaitDelay = 3 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
}

func verificationProcessActive(pid int) bool {
	output, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/FO", "CSV", "/NH").Output()
	return err == nil && bytes.Contains(output, []byte(strconv.Itoa(pid)))
}

func tryVerificationFileLock(file *os.File) error {
	var overlapped syscall.Overlapped
	result, _, callErr := lockFileEx.Call(
		file.Fd(), lockFileExclusive|lockFileFailImmediately, 0, 1, 0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if result != 0 {
		return nil
	}
	if errno, ok := callErr.(syscall.Errno); ok && (errno == errorLockViolation || errno == errorIOPending) {
		return errVerificationLockBusy
	}
	return callErr
}

func unlockVerificationFile(file *os.File) error {
	var overlapped syscall.Overlapped
	result, _, callErr := unlockFileEx.Call(file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if result != 0 {
		return nil
	}
	return callErr
}
