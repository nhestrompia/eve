//go:build windows

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"time"
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
