//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd

package tui

import "os"

// TermGeometry holds terminal dimensions in character cells and pixels.
type TermGeometry struct {
	Cols   int
	Rows   int
	XPixel int
	YPixel int
}

// GetTermGeometry returns fallback dimensions on non-POSIX targets.
func GetTermGeometry(_ int) TermGeometry {
	return TermGeometry{Cols: 120, Rows: 36}
}

// IsTerminalFD returns false on non-POSIX stub targets unless overridden.
func IsTerminalFD(_ int) bool {
	return false
}

// EnterRawMode is a no-op stub on non-POSIX targets.
func EnterRawMode(_ int) (func(), error) {
	return func() {}, nil
}

// SubscribeResize returns an idle channel on non-POSIX targets.
func SubscribeResize() (<-chan os.Signal, func()) {
	ch := make(chan os.Signal)
	return ch, func() { close(ch) }
}
