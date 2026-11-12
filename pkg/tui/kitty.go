package tui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	KittyBoardImageID     uint32 = 42
	KittyBoardPlacementID uint32 = 1
	kittyMaxChunkBytes           = 4096
)

// DetectKittySupport inspects the active environment for terminals implementing the Kitty Graphics Protocol
// (Kitty, Ghostty, WezTerm, Konsole 22.04+, Wayst).
func DetectKittySupport() bool {
	return DetectKittySupportFromEnv(os.Getenv)
}

// DetectKittySupportFromEnv evaluates Kitty Graphics Protocol capability using a custom environment lookup.
func DetectKittySupportFromEnv(getenv func(string) string) bool {
	if v := strings.ToLower(strings.TrimSpace(getenv("NEWSCID_KITTY"))); v != "" {
		switch v {
		case "1", "true", "yes", "on", "force":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}

	if getenv("KITTY_WINDOW_ID") != "" || getenv("KITTY_PID") != "" {
		return true
	}
	if getenv("GHOSTTY_RESOURCES_DIR") != "" || getenv("GHOSTTY_BIN_DIR") != "" {
		return true
	}
	if getenv("WEZTERM_EXECUTABLE") != "" || getenv("WEZTERM_PANE") != "" {
		return true
	}

	term := strings.ToLower(getenv("TERM"))
	if strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") || strings.Contains(term, "wezterm") {
		return true
	}

	termProg := strings.ToLower(getenv("TERM_PROGRAM"))
	switch termProg {
	case "kitty", "ghostty", "wezterm", "konsole", "wayst", "rio":
		return true
	}

	if getenv("KONSOLE_VERSION") != "" {
		return true
	}

	return false
}

// EncodeKittyPNGPlacement encodes a PNG payload into Kitty Graphics Protocol APC escape chunks
// positioned at 1-based terminal cell coordinates (row, col) spanning (cols x rows) cells.
// Uses fixed image ID & placement ID so subsequent renders atomically replace the board in-place without flicker,
// C=1 so cursor does not move, and q=2 to suppress terminal acknowledgement noise on stdin.
func EncodeKittyPNGPlacement(imageID, placementID uint32, row, col, cols, rows int, pngBytes []byte) string {
	if len(pngBytes) == 0 {
		return ""
	}
	if row < 1 {
		row = 1
	}
	if col < 1 {
		col = 1
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	encoded := base64.StdEncoding.EncodeToString(pngBytes)
	var sb strings.Builder
	sb.Grow(len(encoded) + 128*(1+len(encoded)/kittyMaxChunkBytes))

	// Move cursor to top-left cell of board placement
	fmt.Fprintf(&sb, "\x1b[%d;%dH", row, col)

	total := len(encoded)
	offset := 0
	first := true
	for offset < total {
		end := offset + kittyMaxChunkBytes
		more := 1
		if end >= total {
			end = total
			more = 0
		}
		chunk := encoded[offset:end]
		if first {
			fmt.Fprintf(
				&sb,
				"\x1b_Ga=T,f=100,i=%d,p=%d,c=%d,r=%d,C=1,q=2,m=%d;%s\x1b\\",
				imageID, placementID, cols, rows, more, chunk,
			)
			first = false
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
		offset = end
	}

	return sb.String()
}

// EncodeKittyDeleteImage deletes a specific Kitty image ID and all its placements.
func EncodeKittyDeleteImage(imageID uint32) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", imageID)
}

// EncodeKittyDeleteAll deletes all visible Kitty graphics placements on the active screen.
func EncodeKittyDeleteAll() string {
	return "\x1b_Ga=d,d=A,q=2\x1b\\"
}

// KeyKind identifies keyboard and mouse events decoded from the terminal input stream.
type KeyKind uint8

const (
	KeyNone KeyKind = iota
	KeyRune
	KeyEnter
	KeyEsc
	KeyBackspace
	KeyTab
	KeyBackTab
	KeySpace
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDn
	KeyDelete
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyCtrlC
	KeyCtrlQ
	KeyCtrlL
	KeyCtrlU
	KeyCtrlW
	KeyMouse
)

// MouseAction classifies SGR mouse events.
type MouseAction uint8

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseDrag
	MouseWheelUp
	MouseWheelDown
)

// InputEvent represents a parsed keyboard or mouse event.
type InputEvent struct {
	Kind        KeyKind
	Rune        rune
	Alt         bool
	Ctrl        bool
	Shift       bool
	MouseBtn    int // 0=left, 1=middle, 2=right
	MouseCol    int // 1-based terminal column
	MouseRow    int // 1-based terminal row
	MouseAction MouseAction
}

// ParseInputEvents parses as many complete InputEvents as possible from buf,
// returning the events and the number of bytes consumed.
func ParseInputEvents(buf []byte) ([]InputEvent, int) {
	var events []InputEvent
	i := 0
	n := len(buf)

	for i < n {
		b := buf[i]

		if b == 0x1b {
			// Standalone ESC if this is the sole byte in buffer
			if i+1 == n {
				events = append(events, InputEvent{Kind: KeyEsc})
				i++
				continue
			}

			next := buf[i+1]

			// Kitty APC response (\x1b_G ... \x1b\\): discard cleanly
			if next == '_' {
				endIdx := bytes.Index(buf[i+2:], []byte("\x1b\\"))
				if endIdx == -1 {
					break
				}
				i += 2 + endIdx + 2
				continue
			}

			// CSI sequence (\x1b[ ...)
			if next == '[' {
				ev, consumed, ok, incomplete := parseCSISequence(buf[i:])
				if incomplete {
					break
				}
				if ok {
					if ev.Kind != KeyNone {
						events = append(events, ev)
					}
					i += consumed
					continue
				}
				events = append(events, InputEvent{Kind: KeyEsc})
				i++
				continue
			}

			// SS3 sequence (\x1bO ...)
			if next == 'O' {
				if i+2 >= n {
					break
				}
				ch := buf[i+2]
				i += 3
				switch ch {
				case 'A':
					events = append(events, InputEvent{Kind: KeyUp})
				case 'B':
					events = append(events, InputEvent{Kind: KeyDown})
				case 'C':
					events = append(events, InputEvent{Kind: KeyRight})
				case 'D':
					events = append(events, InputEvent{Kind: KeyLeft})
				case 'H':
					events = append(events, InputEvent{Kind: KeyHome})
				case 'F':
					events = append(events, InputEvent{Kind: KeyEnd})
				case 'P':
					events = append(events, InputEvent{Kind: KeyF1})
				case 'Q':
					events = append(events, InputEvent{Kind: KeyF2})
				case 'R':
					events = append(events, InputEvent{Kind: KeyF3})
				case 'S':
					events = append(events, InputEvent{Kind: KeyF4})
				}
				continue
			}

			// Alt + key
			r, rw := utf8.DecodeRune(buf[i+1:])
			if r != utf8.RuneError || rw > 0 {
				events = append(events, InputEvent{Kind: KeyRune, Rune: r, Alt: true})
				i += 1 + rw
				continue
			}

			events = append(events, InputEvent{Kind: KeyEsc})
			i++
			continue
		}

		// Control characters
		switch b {
		case '\r', '\n':
			events = append(events, InputEvent{Kind: KeyEnter})
			i++
			continue
		case '\t':
			events = append(events, InputEvent{Kind: KeyTab})
			i++
			continue
		case 0x7f, 0x08:
			events = append(events, InputEvent{Kind: KeyBackspace})
			i++
			continue
		case 0x03:
			events = append(events, InputEvent{Kind: KeyCtrlC, Ctrl: true})
			i++
			continue
		case 0x11:
			events = append(events, InputEvent{Kind: KeyCtrlQ, Ctrl: true})
			i++
			continue
		case 0x0c:
			events = append(events, InputEvent{Kind: KeyCtrlL, Ctrl: true})
			i++
			continue
		case 0x15:
			events = append(events, InputEvent{Kind: KeyCtrlU, Ctrl: true})
			i++
			continue
		case 0x17:
			events = append(events, InputEvent{Kind: KeyCtrlW, Ctrl: true})
			i++
			continue
		case ' ':
			events = append(events, InputEvent{Kind: KeySpace, Rune: ' '})
			i++
			continue
		}

		if b < 0x20 {
			// Generic Ctrl+letter
			events = append(events, InputEvent{
				Kind: KeyRune,
				Rune: rune('a' + (b - 1)),
				Ctrl: true,
			})
			i++
			continue
		}

		if !utf8.FullRune(buf[i:]) {
			break
		}
		r, rw := utf8.DecodeRune(buf[i:])
		events = append(events, InputEvent{Kind: KeyRune, Rune: r})
		i += rw
	}

	return events, i
}

func parseCSISequence(seq []byte) (ev InputEvent, consumed int, ok bool, incomplete bool) {
	if len(seq) < 3 {
		return InputEvent{}, 0, false, true
	}

	// Find CSI final byte (0x40..0x7E)
	finalIdx := -1
	for j := 2; j < len(seq); j++ {
		c := seq[j]
		if c >= 0x40 && c <= 0x7e {
			finalIdx = j
			break
		}
		if j > 48 {
			return InputEvent{}, 2, false, false
		}
	}
	if finalIdx == -1 {
		return InputEvent{}, 0, false, true
	}

	params := string(seq[2:finalIdx])
	final := seq[finalIdx]
	consumed = finalIdx + 1

	// SGR mouse: \x1b[<btn;col;rowM or m
	if strings.HasPrefix(params, "<") && (final == 'M' || final == 'm') {
		parts := strings.Split(params[1:], ";")
		if len(parts) == 3 {
			cb, err1 := strconv.Atoi(parts[0])
			cx, err2 := strconv.Atoi(parts[1])
			cy, err3 := strconv.Atoi(parts[2])
			if err1 == nil && err2 == nil && err3 == nil {
				ev := InputEvent{
					Kind:     KeyMouse,
					MouseCol: cx,
					MouseRow: cy,
					Shift:    (cb & 4) != 0,
					Alt:      (cb & 8) != 0,
					Ctrl:     (cb & 16) != 0,
				}
				if (cb & 64) != 0 {
					if (cb & 1) != 0 {
						ev.MouseAction = MouseWheelDown
					} else {
						ev.MouseAction = MouseWheelUp
					}
					return ev, consumed, true, false
				}
				ev.MouseBtn = cb & 3
				if final == 'm' {
					ev.MouseAction = MouseRelease
				} else if (cb & 32) != 0 {
					ev.MouseAction = MouseDrag
				} else {
					ev.MouseAction = MousePress
				}
				return ev, consumed, true, false
			}
		}
		return InputEvent{Kind: KeyNone}, consumed, true, false
	}

	// Kitty CSI-u keyboard protocol: \x1b[<codepoint>u or \x1b[<codepoint>;<modifiers>u
	if final == 'u' {
		parts := strings.Split(params, ";")
		cpPart := strings.Split(parts[0], ":")[0]
		cp, err := strconv.Atoi(cpPart)
		if err != nil {
			return InputEvent{Kind: KeyNone}, consumed, true, false
		}
		mod := 1
		if len(parts) >= 2 {
			modPart := strings.Split(parts[1], ":")[0]
			if m, err := strconv.Atoi(modPart); err == nil && m >= 1 {
				mod = m
			}
		}
		shift := ((mod - 1) & 1) != 0
		alt := ((mod - 1) & 2) != 0
		ctrl := ((mod - 1) & 4) != 0

		switch cp {
		case 27:
			return InputEvent{Kind: KeyEsc, Shift: shift, Alt: alt, Ctrl: ctrl}, consumed, true, false
		case 13, 57414:
			return InputEvent{Kind: KeyEnter, Shift: shift, Alt: alt, Ctrl: ctrl}, consumed, true, false
		case 9:
			if shift {
				return InputEvent{Kind: KeyBackTab, Shift: true}, consumed, true, false
			}
			return InputEvent{Kind: KeyTab, Alt: alt, Ctrl: ctrl}, consumed, true, false
		case 127, 8:
			return InputEvent{Kind: KeyBackspace, Shift: shift, Alt: alt, Ctrl: ctrl}, consumed, true, false
		case 32:
			return InputEvent{Kind: KeySpace, Rune: ' ', Shift: shift, Alt: alt, Ctrl: ctrl}, consumed, true, false
		default:
			if ctrl && (cp == 'c' || cp == 'C') {
				return InputEvent{Kind: KeyCtrlC, Ctrl: true}, consumed, true, false
			}
			if ctrl && (cp == 'q' || cp == 'Q') {
				return InputEvent{Kind: KeyCtrlQ, Ctrl: true}, consumed, true, false
			}
			if ctrl && (cp == 'l' || cp == 'L') {
				return InputEvent{Kind: KeyCtrlL, Ctrl: true}, consumed, true, false
			}
			if ctrl && (cp == 'u' || cp == 'U') {
				return InputEvent{Kind: KeyCtrlU, Ctrl: true}, consumed, true, false
			}
			if ctrl && (cp == 'w' || cp == 'W') {
				return InputEvent{Kind: KeyCtrlW, Ctrl: true}, consumed, true, false
			}
			if cp >= 32 && cp <= 0x10ffff {
				r := rune(cp)
				if shift && r >= 'a' && r <= 'z' {
					r = r - 'a' + 'A'
				}
				return InputEvent{Kind: KeyRune, Rune: r, Shift: shift, Alt: alt, Ctrl: ctrl}, consumed, true, false
			}
			return InputEvent{Kind: KeyNone}, consumed, true, false
		}
	}

	switch final {
	case 'A':
		return InputEvent{Kind: KeyUp}, consumed, true, false
	case 'B':
		return InputEvent{Kind: KeyDown}, consumed, true, false
	case 'C':
		return InputEvent{Kind: KeyRight}, consumed, true, false
	case 'D':
		return InputEvent{Kind: KeyLeft}, consumed, true, false
	case 'H':
		return InputEvent{Kind: KeyHome}, consumed, true, false
	case 'F':
		return InputEvent{Kind: KeyEnd}, consumed, true, false
	case 'Z':
		return InputEvent{Kind: KeyBackTab, Shift: true}, consumed, true, false
	case '~':
		numPart := strings.Split(params, ";")[0]
		switch numPart {
		case "1", "7":
			return InputEvent{Kind: KeyHome}, consumed, true, false
		case "3":
			return InputEvent{Kind: KeyDelete}, consumed, true, false
		case "4", "8":
			return InputEvent{Kind: KeyEnd}, consumed, true, false
		case "5":
			return InputEvent{Kind: KeyPgUp}, consumed, true, false
		case "6":
			return InputEvent{Kind: KeyPgDn}, consumed, true, false
		case "11":
			return InputEvent{Kind: KeyF1}, consumed, true, false
		case "12":
			return InputEvent{Kind: KeyF2}, consumed, true, false
		case "13":
			return InputEvent{Kind: KeyF3}, consumed, true, false
		case "14":
			return InputEvent{Kind: KeyF4}, consumed, true, false
		case "15":
			return InputEvent{Kind: KeyF5}, consumed, true, false
		case "17":
			return InputEvent{Kind: KeyF6}, consumed, true, false
		}
	}

	return InputEvent{Kind: KeyNone}, consumed, true, false
}
