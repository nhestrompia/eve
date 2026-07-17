//go:build !unix && !windows

package main

import "os/exec"

func configureVerificationProcess(cmd *exec.Cmd) {}

func verificationProcessActive(pid int) bool { return true }
