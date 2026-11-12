//go:build darwin || freebsd || openbsd || netbsd

package tui

import "golang.org/x/sys/unix"

const (
	ioctlGetTermiosReq = unix.TIOCGETA
	ioctlSetTermiosReq = unix.TIOCSETA
)
