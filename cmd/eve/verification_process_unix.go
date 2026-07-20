//go:build unix

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureVerificationProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 3 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		pid := cmd.Process.Pid
		err := syscall.Kill(-pid, syscall.SIGTERM)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		go func() {
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()
			<-timer.C
			_ = syscall.Kill(-pid, syscall.SIGKILL)
		}()
		return nil
	}
}

func verificationProcessActive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func tryVerificationFileLock(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return errVerificationLockBusy
	}
	return err
}

func unlockVerificationFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
