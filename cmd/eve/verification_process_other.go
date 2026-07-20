//go:build !unix && !windows

package main

import (
	"errors"
	"os"
	"os/exec"
)

func configureVerificationProcess(cmd *exec.Cmd) {}

func verificationProcessActive(pid int) bool { return true }

func tryVerificationFileLock(file *os.File) error {
	claim, err := os.OpenFile(file.Name()+".claim", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return errVerificationLockBusy
	}
	if err != nil {
		return err
	}
	return claim.Close()
}

func unlockVerificationFile(file *os.File) error {
	err := os.Remove(file.Name() + ".claim")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
