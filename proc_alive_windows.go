//go:build windows

package main

import "syscall"

const processQueryLimitedInfo = 0x1000

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := syscall.OpenProcess(processQueryLimitedInfo, false, uint32(pid))
	if err != nil || h == 0 {
		return false
	}
	_ = syscall.CloseHandle(h)
	return true
}
