//go:build linux || darwin || freebsd || openbsd || netbsd

package tui

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

// TermGeometry holds terminal dimensions in character cells and pixels.
type TermGeometry struct {
	Cols   int
	Rows   int
	XPixel int
	YPixel int
}

// GetTermGeometry queries terminal dimensions via TIOCGWINSZ, falling back to 120x36.
func GetTermGeometry(fd int) TermGeometry {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || ws == nil {
		return TermGeometry{Cols: 120, Rows: 36}
	}
	cols := int(ws.Col)
	rows := int(ws.Row)
	if cols <= 0 {
		cols = 120
	}
	if rows <= 0 {
		rows = 36
	}
	return TermGeometry{
		Cols:   cols,
		Rows:   rows,
		XPixel: int(ws.Xpixel),
		YPixel: int(ws.Ypixel),
	}
}

// IsTerminalFD returns true if fd refers to an interactive TTY.
func IsTerminalFD(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, ioctlGetTermiosReq)
	return err == nil
}

// EnterRawMode switches fd into raw terminal mode and returns a restore callback.
func EnterRawMode(fd int) (func(), error) {
	orig, err := unix.IoctlGetTermios(fd, ioctlGetTermiosReq)
	if err != nil {
		return nil, err
	}

	raw := *orig
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, ioctlSetTermiosReq, &raw); err != nil {
		return nil, err
	}

	restore := func() {
		_ = unix.IoctlSetTermios(fd, ioctlSetTermiosReq, orig)
	}
	return restore, nil
}

// SubscribeResize registers a SIGWINCH listener and returns the notification channel and cleanup func.
func SubscribeResize() (<-chan os.Signal, func()) {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, unix.SIGWINCH)
	cleanup := func() {
		signal.Stop(ch)
		close(ch)
	}
	return ch, cleanup
}
