//go:build linux

package tui

import "golang.org/x/sys/unix"

const (
	ioctlGetTermiosReq = unix.TCGETS
	ioctlSetTermiosReq = unix.TCSETS
)
