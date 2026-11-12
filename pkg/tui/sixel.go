package tui

import (
	"fmt"
	"image"
	"sort"
	"strconv"
	"strings"
)

// GraphicsMode identifies the active chessboard rendering backend in the terminal.
type GraphicsMode uint8

const (
	GraphicsUnicode GraphicsMode = iota
	GraphicsKitty
	GraphicsSixel
)

func (m GraphicsMode) String() string {
	switch m {
	case GraphicsKitty:
		return "KITTY HD"
	case GraphicsSixel:
		return "SIXEL HD"
	default:
		return "UNICODE RGB"
	}
}

// IsPixelMode returns true when rendering high-DPI bitmap boards (Kitty or Sixel).
func (m GraphicsMode) IsPixelMode() bool {
	return m == GraphicsKitty || m == GraphicsSixel
}

// IsFootTerminal returns true when TERM indicates the Wayland foot terminal emulator.
func IsFootTerminal(getenv func(string) string) bool {
	term := strings.ToLower(strings.TrimSpace(getenv("TERM")))
	if strings.HasPrefix(term, "foot") {
		return true
	}
	return strings.ToLower(strings.TrimSpace(getenv("TERM_PROGRAM"))) == "foot"
}

// DetectSixelSupportFromEnv inspects environment variables for terminals with native DEC Sixel graphics support
// (foot, contour, mlterm, mintty, yaft, Windows Terminal, xterm/wezterm).
func DetectSixelSupportFromEnv(getenv func(string) string) bool {
	if v := strings.ToLower(strings.TrimSpace(getenv("NEWSCID_SIXEL"))); v != "" {
		switch v {
		case "1", "true", "yes", "on", "force":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}

	if IsFootTerminal(getenv) {
		return true
	}

	term := strings.ToLower(strings.TrimSpace(getenv("TERM")))
	for _, prefix := range []string{"mlterm", "yaft", "contour", "sixel"} {
		if strings.Contains(term, prefix) {
			return true
		}
	}

	termProg := strings.ToLower(strings.TrimSpace(getenv("TERM_PROGRAM")))
	switch termProg {
	case "foot", "contour", "mintty", "blackbox":
		return true
	}

	if getenv("WT_SESSION") != "" {
		return true
	}

	return false
}

// DetectBestGraphicsModeFromEnv selects the highest-fidelity graphics protocol supported by the active terminal.
func DetectBestGraphicsModeFromEnv(getenv func(string) string) GraphicsMode {
	if g := strings.ToLower(strings.TrimSpace(getenv("NEWSCID_GRAPHICS"))); g != "" {
		switch g {
		case "kitty":
			return GraphicsKitty
		case "sixel":
			return GraphicsSixel
		case "unicode", "rgb", "ansi", "ascii", "none", "off":
			return GraphicsUnicode
		}
	}

	if v := strings.ToLower(strings.TrimSpace(getenv("NEWSCID_KITTY"))); v == "0" || v == "false" || v == "off" {
		if DetectSixelSupportFromEnv(getenv) {
			return GraphicsSixel
		}
		return GraphicsUnicode
	}

	if DetectKittySupportFromEnv(getenv) {
		return GraphicsKitty
	}
	if DetectSixelSupportFromEnv(getenv) {
		return GraphicsSixel
	}
	return GraphicsUnicode
}

type rgbBucket struct {
	key   int
	count int
	r, g, b uint8
}

// EncodeSixelPlacement encodes an RGBA image into a DEC Sixel DCS sequence positioned at 1-based cell (row, col).
func EncodeSixelPlacement(row, col int, img *image.RGBA) string {
	if img == nil {
		return ""
	}
	var sb strings.Builder
	if row > 0 && col > 0 {
		fmt.Fprintf(&sb, "\x1b[%d;%dH", row, col)
	}
	sb.WriteString(EncodeSixelImage(img))
	return sb.String()
}

// EncodeSixelImage encodes an RGBA image into a raw DEC Sixel DCS payload (`\x1bP0;1;0q... \x1b\\`).
func EncodeSixelImage(img *image.RGBA) string {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return ""
	}

	// 1. Build 15-bit RGB555 popularity histogram over all pixels
	var counts [32768]int32
	var sampleR, sampleG, sampleB [32768]uint8

	for y := 0; y < h; y++ {
		off := img.PixOffset(bounds.Min.X, bounds.Min.Y+y)
		for x := 0; x < w; x++ {
			r := img.Pix[off]
			g := img.Pix[off+1]
			b := img.Pix[off+2]
			key := (int(r>>3) << 10) | (int(g>>3) << 5) | int(b>>3)
			if counts[key] == 0 {
				sampleR[key] = r
				sampleG[key] = g
				sampleB[key] = b
			}
			counts[key]++
			off += 4
		}
	}

	buckets := make([]rgbBucket, 0, 256)
	for k, c := range counts {
		if c > 0 {
			buckets = append(buckets, rgbBucket{
				key:   k,
				count: int(c),
				r:     sampleR[k],
				g:     sampleG[k],
				b:     sampleB[k],
			})
		}
	}

	// 2. Build up to 256-register palette and O(1) RGB555 -> register LUT
	const maxRegs = 256
	var palR, palG, palB [maxRegs]uint8
	palLen := 0
	var lut [32768]uint8

	if len(buckets) <= maxRegs {
		for i, bk := range buckets {
			palR[i] = bk.r
			palG[i] = bk.g
			palB[i] = bk.b
			lut[bk.key] = uint8(i)
		}
		palLen = len(buckets)
	} else {
		sort.Slice(buckets, func(i, j int) bool {
			return buckets[i].count > buckets[j].count
		})

		picked := make([]bool, len(buckets))
		// First pass: pick popular distinct colors so arrows, highlights, and pieces never get starved
		for i, bk := range buckets {
			if palLen >= maxRegs {
				break
			}
			tooClose := false
			for p := 0; p < palLen; p++ {
				dr := int(bk.r) - int(palR[p])
				dg := int(bk.g) - int(palG[p])
				db := int(bk.b) - int(palB[p])
				if dr*dr+dg*dg+db*db < 144 {
					tooClose = true
					break
				}
			}
			if !tooClose {
				palR[palLen] = bk.r
				palG[palLen] = bk.g
				palB[palLen] = bk.b
				picked[i] = true
				palLen++
			}
		}
		// Second pass: fill remaining palette slots by highest pixel frequency
		for i, bk := range buckets {
			if palLen >= maxRegs {
				break
			}
			if !picked[i] {
				palR[palLen] = bk.r
				palG[palLen] = bk.g
				palB[palLen] = bk.b
				palLen++
			}
		}

		// Map every populated RGB555 bucket to its closest palette register
		for _, bk := range buckets {
			bestIdx := uint8(0)
			bestDist := 1 << 30
			for p := 0; p < palLen; p++ {
				dr := int(bk.r) - int(palR[p])
				dg := int(bk.g) - int(palG[p])
				db := int(bk.b) - int(palB[p])
				d := dr*dr + dg*dg + db*db
				if d < bestDist {
					bestDist = d
					bestIdx = uint8(p)
					if d == 0 {
						break
					}
				}
			}
			lut[bk.key] = bestIdx
		}
	}

	// Precompute indexed pixel buffer
	indexed := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		off := img.PixOffset(bounds.Min.X, bounds.Min.Y+y)
		rowBase := y * w
		for x := 0; x < w; x++ {
			key := (int(img.Pix[off]>>3) << 10) | (int(img.Pix[off+1]>>3) << 5) | int(img.Pix[off+2]>>3)
			indexed[rowBase+x] = lut[key]
			off += 4
		}
	}

	var sb strings.Builder
	sb.Grow(w * h / 4)

	// DCS Sixel introducer + 1:1 pixel aspect + exact raster dimensions
	sb.WriteString("\x1bP0;1;0q\"1;1;")
	sb.WriteString(strconv.Itoa(w))
	sb.WriteByte(';')
	sb.WriteString(strconv.Itoa(h))

	// Emit color register table (#reg;2;R%;G%;B%)
	for p := 0; p < palLen; p++ {
		r100 := (int(palR[p])*100 + 127) / 255
		g100 := (int(palG[p])*100 + 127) / 255
		b100 := (int(palB[p])*100 + 127) / 255
		sb.WriteByte('#')
		sb.WriteString(strconv.Itoa(p))
		sb.WriteString(";2;")
		sb.WriteString(strconv.Itoa(r100))
		sb.WriteByte(';')
		sb.WriteString(strconv.Itoa(g100))
		sb.WriteByte(';')
		sb.WriteString(strconv.Itoa(b100))
	}

	// 3. Encode 6-pixel-high bands with per-color span bounds and RLE compression
	var lastX [maxRegs]int
	activeColors := make([]uint8, 0, 64)
	rowBits := make([]uint8, w)

	for bandY := 0; bandY < h; bandY += 6 {
		for i := 0; i < palLen; i++ {
			lastX[i] = -1
		}
		activeColors = activeColors[:0]

		bandRows := h - bandY
		if bandRows > 6 {
			bandRows = 6
		}

		for dy := 0; dy < bandRows; dy++ {
			rowOff := (bandY + dy) * w
			for x := 0; x < w; x++ {
				ci := indexed[rowOff+x]
				if lastX[ci] == -1 {
					activeColors = append(activeColors, ci)
				}
				if x > lastX[ci] {
					lastX[ci] = x
				}
			}
		}

		for idx, ci := range activeColors {
			endX := lastX[ci]
			if endX < 0 {
				continue
			}

			// Clear rowBits[0..endX]
			for x := 0; x <= endX; x++ {
				rowBits[x] = 0
			}
			for dy := 0; dy < bandRows; dy++ {
				bit := uint8(1 << uint(dy))
				rowOff := (bandY + dy) * w
				for x := 0; x <= endX; x++ {
					if indexed[rowOff+x] == ci {
						rowBits[x] |= bit
					}
				}
			}

			sb.WriteByte('#')
			sb.WriteString(strconv.Itoa(int(ci)))

			// Run-length encode rowBits[0..endX]
			x := 0
			for x <= endX {
				val := rowBits[x]
				run := 1
				for x+run <= endX && rowBits[x+run] == val {
					run++
				}
				ch := byte(63 + val)
				if run >= 4 {
					sb.WriteByte('!')
					sb.WriteString(strconv.Itoa(run))
					sb.WriteByte(ch)
				} else {
					for k := 0; k < run; k++ {
						sb.WriteByte(ch)
					}
				}
				x += run
			}

			// '$' returns carriage to x=0 on same 6px band for next color
			if idx+1 < len(activeColors) {
				sb.WriteByte('$')
			}
		}

		// '-' advances down 6 pixels to next Sixel band
		if bandY+6 < h {
			sb.WriteByte('-')
		}
	}

	sb.WriteString("\x1b\\")
	return sb.String()
}
