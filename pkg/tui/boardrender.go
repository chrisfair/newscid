package tui

import (
	"bytes"
	"embed"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"
	"sync"

	"newscid/pkg/chess"
	"newscid/pkg/sets"
)

//go:embed assets/*.png
var embeddedPieceAssets embed.FS

// AvailableTUIPieceSets lists the embedded high-DPI vector-rasterized piece sets available in the TUI.
var AvailableTUIPieceSets = []string{
	"cburnett",
	"merida",
	"staunty",
	"alpha",
	"leipzig",
	"cardinal",
}

const spriteTileSize = 96

var (
	spriteCacheMu sync.Mutex
	spriteCache   = make(map[string]*image.NRGBA)
)

// ArrowKind classifies visual arrow overlays rendered on the Kitty chessboard.
type ArrowKind uint8

const (
	ArrowEnginePV1 ArrowKind = iota
	ArrowEnginePV2
	ArrowOpeningBook
	ArrowUserHint
)

// BoardArrow represents a translucent directional arrow between two squares.
type BoardArrow struct {
	From chess.Square
	To   chess.Square
	Kind ArrowKind
}

// BoardRenderOptions controls high-DPI PNG chessboard rendering for Kitty Graphics placement.
type BoardRenderOptions struct {
	ThemeID       string
	PieceSetID    string
	SquarePx      int
	Flipped       bool
	SelectedSq    chess.Square
	LastFrom      chess.Square
	LastTo        chess.Square
	LegalTargets  []chess.Square
	CaptureMask   uint64 // bitmask indexed by Square (0..63) for capture rings
	Arrows        []BoardArrow
	ShowCoords    bool
	CheckKingSq   chess.Square
}

// Hash computes a deterministic 64-bit digest of the board state and visual overlays
// so Kitty image placements are only re-encoded and re-transmitted when pixels change.
func (opt BoardRenderOptions) Hash(b *chess.Board) uint64 {
	h := fnv.New64a()
	if b != nil {
		for sq := 0; sq < 64; sq++ {
			_, _ = h.Write([]byte{byte(b.Squares[sq])})
		}
		_, _ = h.Write([]byte{byte(b.Turn)})
	}
	_, _ = h.Write([]byte(opt.ThemeID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(opt.PieceSetID))
	_, _ = h.Write([]byte{
		byte(opt.SquarePx >> 8), byte(opt.SquarePx),
		boolByte(opt.Flipped),
		byte(opt.SelectedSq),
		byte(opt.LastFrom),
		byte(opt.LastTo),
		boolByte(opt.ShowCoords),
		byte(opt.CheckKingSq),
	})
	for _, sq := range opt.LegalTargets {
		_, _ = h.Write([]byte{byte(sq)})
	}
	var capBuf [8]byte
	for i := 0; i < 8; i++ {
		capBuf[i] = byte(opt.CaptureMask >> (uint(i) * 8))
	}
	_, _ = h.Write(capBuf[:])
	for _, ar := range opt.Arrows {
		_, _ = h.Write([]byte{byte(ar.From), byte(ar.To), byte(ar.Kind)})
	}
	return h.Sum64()
}

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}

func loadSpriteStrip(pieceSetID string) (*image.NRGBA, error) {
	id := strings.ToLower(strings.TrimSpace(pieceSetID))
	if id == "" {
		id = "cburnett"
	}

	spriteCacheMu.Lock()
	if img, ok := spriteCache[id]; ok {
		spriteCacheMu.Unlock()
		return img, nil
	}
	spriteCacheMu.Unlock()

	data, err := embeddedPieceAssets.ReadFile("assets/" + id + ".png")
	if err != nil {
		data, err = embeddedPieceAssets.ReadFile("assets/cburnett.png")
		if err != nil {
			return nil, fmt.Errorf("load piece spritesheet: %w", err)
		}
	}

	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode piece spritesheet: %w", err)
	}

	bounds := decoded.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba.Set(x-bounds.Min.X, y-bounds.Min.Y, decoded.At(x, y))
		}
	}

	spriteCacheMu.Lock()
	spriteCache[id] = nrgba
	spriteCacheMu.Unlock()
	return nrgba, nil
}

// LookupBoardTheme returns the BoardThemeInfo matching id, defaulting to tournament-green.
func LookupBoardTheme(id string) sets.BoardThemeInfo {
	themes := sets.GetAvailableBoardThemes()
	norm := strings.ToLower(strings.TrimSpace(id))
	for _, t := range themes {
		if strings.ToLower(t.ID) == norm {
			return t
		}
	}
	for _, t := range themes {
		if t.ID == "tournament-green" {
			return t
		}
	}
	return sets.BoardThemeInfo{
		ID:        "tournament-green",
		Name:      "Tournament Green",
		Light:     "#eeeed2",
		Dark:      "#769656",
		Highlight: "#baca44",
	}
}

// ParseHexRGB parses "#rrggbb" into R, G, B bytes.
func ParseHexRGB(hexStr string) (uint8, uint8, uint8) {
	s := strings.TrimPrefix(strings.TrimSpace(hexStr), "#")
	if len(s) != 6 {
		return 128, 128, 128
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 128, 128, 128
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v)
}

func pieceSpriteIndex(p chess.Piece) int {
	switch p {
	case chess.WK:
		return 0
	case chess.WQ:
		return 1
	case chess.WR:
		return 2
	case chess.WB:
		return 3
	case chess.WN:
		return 4
	case chess.WP:
		return 5
	case chess.BK:
		return 6
	case chess.BQ:
		return 7
	case chess.BR:
		return 8
	case chess.BB:
		return 9
	case chess.BN:
		return 10
	case chess.BP:
		return 11
	default:
		return -1
	}
}

// SquareToDisplayCoords maps a chess.Square (A1=0..H8=63) to 0-indexed (col, row) on the 8x8 visual grid.
func SquareToDisplayCoords(sq chess.Square, flipped bool) (col, row int) {
	file := sq.File()
	rank := sq.Rank()
	if flipped {
		return 7 - file, rank
	}
	return file, 7 - rank
}

// DisplayCoordsToSquare maps 0-indexed (col, row) on the 8x8 visual grid to a chess.Square.
func DisplayCoordsToSquare(col, row int, flipped bool) chess.Square {
	if col < 0 || col > 7 || row < 0 || row > 7 {
		return chess.NoSquare
	}
	if flipped {
		return chess.MakeSquare(7-col, row)
	}
	return chess.MakeSquare(col, 7-row)
}

// RenderBoardImage renders an 8x8 RGBA chessboard image with pieces, highlights, legal moves, coordinates, and arrows.
func RenderBoardImage(b *chess.Board, opt BoardRenderOptions) (*image.RGBA, error) {
	if b == nil {
		b = chess.NewBoard()
	}
	sqPx := opt.SquarePx
	if sqPx < 32 {
		sqPx = 72
	}
	if sqPx > 160 {
		sqPx = 160
	}

	boardPx := sqPx * 8
	canvas := image.NewRGBA(image.Rect(0, 0, boardPx, boardPx))

	theme := LookupBoardTheme(opt.ThemeID)
	lr, lg, lb := ParseHexRGB(theme.Light)
	dr, dg, db := ParseHexRGB(theme.Dark)
	hr, hg, hb := ParseHexRGB(theme.Highlight)

	legalSet := make(map[chess.Square]bool, len(opt.LegalTargets))
	for _, sq := range opt.LegalTargets {
		legalSet[sq] = true
	}

	checkSq := opt.CheckKingSq
	if checkSq >= 64 && b.InCheck(b.Turn) {
		checkSq = b.KingSquare(b.Turn)
	}

	// 1. Draw base squares, highlights, and king-in-check radial glow
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := DisplayCoordsToSquare(col, row, opt.Flipped)
			isLight := (sq.File()+sq.Rank())%2 == 1
			r, g, bl := dr, dg, db
			if isLight {
				r, g, bl = lr, lg, lb
			}

			if sq == opt.LastFrom || sq == opt.LastTo {
				r = blendByte(r, hr, 0.58)
				g = blendByte(g, hg, 0.58)
				bl = blendByte(bl, hb, 0.58)
			}
			if sq == opt.SelectedSq {
				r = blendByte(r, 76, 0.48)
				g = blendByte(g, 194, 0.48)
				bl = blendByte(bl, 148, 0.48)
			}

			x0 := col * sqPx
			y0 := row * sqPx
			fillRectRGBA(canvas, x0, y0, sqPx, sqPx, color.NRGBA{R: r, G: g, B: bl, A: 255})

			if sq == checkSq {
				drawCheckGlow(canvas, x0, y0, sqPx)
			}
		}
	}

	// 2. Draw rank/file coordinates in outer squares
	if opt.ShowCoords {
		coordScale := sqPx / 36
		if coordScale < 1 {
			coordScale = 1
		}
		if coordScale > 3 {
			coordScale = 3
		}
		pad := maxInt(3, sqPx/18)

		for row := 0; row < 8; row++ {
			sq := DisplayCoordsToSquare(0, row, opt.Flipped)
			isLight := (sq.File()+sq.Rank())%2 == 1
			fg := color.NRGBA{R: lr, G: lg, B: lb, A: 225}
			if isLight {
				fg = color.NRGBA{R: dr, G: dg, B: db, A: 225}
			}
			rankChar := byte('1' + sq.Rank())
			drawTinyGlyph(canvas, pad, row*sqPx+pad, rankChar, coordScale, fg)
		}

		for col := 0; col < 8; col++ {
			sq := DisplayCoordsToSquare(col, 7, opt.Flipped)
			isLight := (sq.File()+sq.Rank())%2 == 1
			fg := color.NRGBA{R: lr, G: lg, B: lb, A: 225}
			if isLight {
				fg = color.NRGBA{R: dr, G: dg, B: db, A: 225}
			}
			fileChar := byte('a' + sq.File())
			gw := 5 * coordScale
			gh := 7 * coordScale
			drawTinyGlyph(canvas, (col+1)*sqPx-gw-pad, 8*sqPx-gh-pad, fileChar, coordScale, fg)
		}
	}

	// 3. Draw pieces using bilinear resampling from embedded 96x96 spritesheets
	strip, err := loadSpriteStrip(opt.PieceSetID)
	if err != nil {
		return nil, err
	}

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := DisplayCoordsToSquare(col, row, opt.Flipped)
			p := b.PieceAt(sq)
			idx := pieceSpriteIndex(p)
			if idx >= 0 {
				blitPieceBilinear(canvas, strip, idx, col*sqPx, row*sqPx, sqPx)
			}
		}
	}

	// 4. Draw legal move dots (quiet moves) and rings (captures)
	for sq := range legalSet {
		if sq >= 64 {
			continue
		}
		col, row := SquareToDisplayCoords(sq, opt.Flipped)
		cx := float64(col*sqPx) + float64(sqPx)*0.5
		cy := float64(row*sqPx) + float64(sqPx)*0.5
		isCapture := (opt.CaptureMask&(uint64(1)<<uint(sq))) != 0 || b.PieceAt(sq) != chess.Empty
		if isCapture {
			outerR := float64(sqPx) * 0.44
			innerR := float64(sqPx) * 0.34
			drawRingAA(canvas, cx, cy, innerR, outerR, color.NRGBA{R: 20, G: 30, B: 24, A: 135})
		} else {
			dotR := float64(sqPx) * 0.155
			drawDiscAA(canvas, cx, cy, dotR, color.NRGBA{R: 20, G: 30, B: 24, A: 120})
		}
	}

	// 5. Draw directional arrows (PV lines, book moves) on top
	// Draw secondary arrows first so primary PV1 arrow renders cleanly on top
	for i := len(opt.Arrows) - 1; i >= 0; i-- {
		ar := opt.Arrows[i]
		if ar.From >= 64 || ar.To >= 64 || ar.From == ar.To {
			continue
		}
		drawBoardArrow(canvas, ar, sqPx, opt.Flipped)
	}

	return canvas, nil
}

// RenderBoardPNG encodes the rendered chessboard directly to PNG bytes for Kitty Graphics transmission.
func RenderBoardPNG(b *chess.Board, opt BoardRenderOptions) ([]byte, error) {
	img, err := RenderBoardImage(b, opt)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func blendByte(base, over uint8, alpha float64) uint8 {
	v := float64(base)*(1.0-alpha) + float64(over)*alpha
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func fillRectRGBA(dst *image.RGBA, x0, y0, w, h int, c color.NRGBA) {
	bounds := dst.Bounds()
	x1 := minInt(bounds.Max.X, x0+w)
	y1 := minInt(bounds.Max.Y, y0+h)
	x0 = maxInt(bounds.Min.X, x0)
	y0 = maxInt(bounds.Min.Y, y0)
	for y := y0; y < y1; y++ {
		rowOff := dst.PixOffset(x0, y)
		for x := x0; x < x1; x++ {
			dst.Pix[rowOff] = c.R
			dst.Pix[rowOff+1] = c.G
			dst.Pix[rowOff+2] = c.B
			dst.Pix[rowOff+3] = c.A
			rowOff += 4
		}
	}
}

func blendPixel(dst *image.RGBA, x, y int, src color.NRGBA) {
	if src.A == 0 {
		return
	}
	b := dst.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	off := dst.PixOffset(x, y)
	sa := uint32(src.A)
	inv := 255 - sa
	dst.Pix[off] = uint8((uint32(src.R)*sa + uint32(dst.Pix[off])*inv + 127) / 255)
	dst.Pix[off+1] = uint8((uint32(src.G)*sa + uint32(dst.Pix[off+1])*inv + 127) / 255)
	dst.Pix[off+2] = uint8((uint32(src.B)*sa + uint32(dst.Pix[off+2])*inv + 127) / 255)
	dst.Pix[off+3] = 255
}

func drawCheckGlow(dst *image.RGBA, x0, y0, sqPx int) {
	cx := float64(x0) + float64(sqPx)*0.5
	cy := float64(y0) + float64(sqPx)*0.5
	maxR := float64(sqPx) * 0.68
	for y := y0; y < y0+sqPx; y++ {
		for x := x0; x < x0+sqPx; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			dist := math.Hypot(dx, dy)
			if dist < maxR {
				t := 1.0 - (dist / maxR)
				alpha := uint8(math.Min(220, 225.0*t*t))
				blendPixel(dst, x, y, color.NRGBA{R: 225, G: 36, B: 36, A: alpha})
			}
		}
	}
}

func blitPieceBilinear(dst *image.RGBA, strip *image.NRGBA, tileIdx, dstX, dstY, sqPx int) {
	srcX0 := tileIdx * spriteTileSize
	scale := float64(spriteTileSize) / float64(sqPx)

	for y := 0; y < sqPx; y++ {
		sy := (float64(y)+0.5)*scale - 0.5
		if sy < 0 {
			sy = 0
		}
		y0 := int(sy)
		y1 := y0 + 1
		if y1 >= spriteTileSize {
			y1 = spriteTileSize - 1
		}
		fy := sy - float64(y0)

		for x := 0; x < sqPx; x++ {
			sx := (float64(x)+0.5)*scale - 0.5
			if sx < 0 {
				sx = 0
			}
			x0 := int(sx)
			x1 := x0 + 1
			if x1 >= spriteTileSize {
				x1 = spriteTileSize - 1
			}
			fx := sx - float64(x0)

			c00 := strip.NRGBAAt(srcX0+x0, y0)
			c10 := strip.NRGBAAt(srcX0+x1, y0)
			c01 := strip.NRGBAAt(srcX0+x0, y1)
			c11 := strip.NRGBAAt(srcX0+x1, y1)

			if c00.A == 0 && c10.A == 0 && c01.A == 0 && c11.A == 0 {
				continue
			}

			w00 := (1 - fx) * (1 - fy)
			w10 := fx * (1 - fy)
			w01 := (1 - fx) * fy
			w11 := fx * fy

			aSum := float64(c00.A)*w00 + float64(c10.A)*w10 + float64(c01.A)*w01 + float64(c11.A)*w11
			if aSum < 1.0 {
				continue
			}

			// Alpha-premultiplied interpolation prevents dark fringes on vector anti-aliased edges
			rPremul := float64(c00.R)*float64(c00.A)*w00 +
				float64(c10.R)*float64(c10.A)*w10 +
				float64(c01.R)*float64(c01.A)*w01 +
				float64(c11.R)*float64(c11.A)*w11
			gPremul := float64(c00.G)*float64(c00.A)*w00 +
				float64(c10.G)*float64(c10.A)*w10 +
				float64(c01.G)*float64(c01.A)*w01 +
				float64(c11.G)*float64(c11.A)*w11
			bPremul := float64(c00.B)*float64(c00.A)*w00 +
				float64(c10.B)*float64(c10.A)*w10 +
				float64(c01.B)*float64(c01.A)*w01 +
				float64(c11.B)*float64(c11.A)*w11

			outR := clampUint8(rPremul / aSum)
			outG := clampUint8(gPremul / aSum)
			outB := clampUint8(bPremul / aSum)
			outA := clampUint8(aSum)

			blendPixel(dst, dstX+x, dstY+y, color.NRGBA{R: outR, G: outG, B: outB, A: outA})
		}
	}
}

func drawDiscAA(dst *image.RGBA, cx, cy, radius float64, c color.NRGBA) {
	x0 := int(math.Floor(cx - radius - 1.5))
	x1 := int(math.Ceil(cx + radius + 1.5))
	y0 := int(math.Floor(cy - radius - 1.5))
	y1 := int(math.Ceil(cy + radius + 1.5))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			cov := clamp01(radius + 0.5 - d)
			if cov > 0 {
				sc := c
				sc.A = uint8(float64(c.A)*cov + 0.5)
				blendPixel(dst, x, y, sc)
			}
		}
	}
}

func drawRingAA(dst *image.RGBA, cx, cy, innerR, outerR float64, c color.NRGBA) {
	x0 := int(math.Floor(cx - outerR - 1.5))
	x1 := int(math.Ceil(cx + outerR + 1.5))
	y0 := int(math.Floor(cy - outerR - 1.5))
	y1 := int(math.Ceil(cy + outerR + 1.5))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			outerCov := clamp01(outerR + 0.5 - d)
			innerCov := clamp01(d - (innerR - 0.5))
			cov := outerCov * innerCov
			if cov > 0 {
				sc := c
				sc.A = uint8(float64(c.A)*cov + 0.5)
				blendPixel(dst, x, y, sc)
			}
		}
	}
}

func arrowStyleColor(kind ArrowKind) color.NRGBA {
	switch kind {
	case ArrowEnginePV1:
		return color.NRGBA{R: 34, G: 197, B: 94, A: 185}
	case ArrowEnginePV2:
		return color.NRGBA{R: 56, G: 189, B: 248, A: 155}
	case ArrowOpeningBook:
		return color.NRGBA{R: 245, G: 158, B: 11, A: 175}
	default:
		return color.NRGBA{R: 168, G: 85, B: 247, A: 175}
	}
}

func drawBoardArrow(dst *image.RGBA, ar BoardArrow, sqPx int, flipped bool) {
	c0, r0 := SquareToDisplayCoords(ar.From, flipped)
	c1, r1 := SquareToDisplayCoords(ar.To, flipped)

	x0 := (float64(c0) + 0.5) * float64(sqPx)
	y0 := (float64(r0) + 0.5) * float64(sqPx)
	x1 := (float64(c1) + 0.5) * float64(sqPx)
	y1 := (float64(r1) + 0.5) * float64(sqPx)

	dx := x1 - x0
	dy := y1 - y0
	length := math.Hypot(dx, dy)
	if length < 4 {
		return
	}
	ux := dx / length
	uy := dy / length
	px := -uy
	py := ux

	startInset := float64(sqPx) * 0.14
	endInset := float64(sqPx) * 0.08
	headLen := float64(sqPx) * 0.36
	headHalfW := float64(sqPx) * 0.22
	shaftHalfW := float64(sqPx) * 0.085

	sx := x0 + ux*startInset
	sy := y0 + uy*startInset
	tx := x1 - ux*endInset
	ty := y1 - uy*endInset

	totalSpan := math.Hypot(tx-sx, ty-sy)
	if totalSpan <= headLen*1.1 {
		headLen = totalSpan * 0.55
	}
	neckLen := totalSpan - headLen

	col := arrowStyleColor(ar.Kind)

	minX := int(math.Floor(math.Min(sx, tx) - headHalfW - 2))
	maxX := int(math.Ceil(math.Max(sx, tx) + headHalfW + 2))
	minY := int(math.Floor(math.Min(sy, ty) - headHalfW - 2))
	maxY := int(math.Ceil(math.Max(sy, ty) + headHalfW + 2))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			vx := float64(x) + 0.5 - sx
			vy := float64(y) + 0.5 - sy
			along := vx*ux + vy*uy
			perp := math.Abs(vx*px + vy*py)

			var cov float64
			if along >= -0.5 && along <= neckLen+0.5 {
				cAlong := clamp01(along+0.5) * clamp01(neckLen+0.5-along)
				cPerp := clamp01(shaftHalfW + 0.5 - perp)
				cov = cAlong * cPerp
			}
			if along >= neckLen-0.5 && along <= totalSpan+0.5 {
				t := (totalSpan - along) / headLen
				if t < 0 {
					t = 0
				}
				if t > 1 {
					t = 1
				}
				maxW := headHalfW * t
				cAlong := clamp01(along-(neckLen-0.5)) * clamp01(totalSpan+0.5-along)
				cPerp := clamp01(maxW + 0.5 - perp)
				headCov := cAlong * cPerp
				if headCov > cov {
					cov = headCov
				}
			}

			if cov > 0 {
				sc := col
				sc.A = uint8(float64(col.A)*cov + 0.5)
				blendPixel(dst, x, y, sc)
			}
		}
	}
}

// Crisp 5x7 bitmap glyphs for coordinates a-h and 1-8.
var coordGlyph5x7 = map[byte][7]uint8{
	'1': {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'2': {0x0E, 0x11, 0x01, 0x06, 0x08, 0x10, 0x1F},
	'3': {0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'a': {0x00, 0x00, 0x0E, 0x01, 0x0F, 0x11, 0x0F},
	'b': {0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x1E},
	'c': {0x00, 0x00, 0x0E, 0x11, 0x10, 0x11, 0x0E},
	'd': {0x01, 0x01, 0x0F, 0x11, 0x11, 0x11, 0x0F},
	'e': {0x00, 0x00, 0x0E, 0x11, 0x1F, 0x10, 0x0E},
	'f': {0x06, 0x08, 0x1E, 0x08, 0x08, 0x08, 0x08},
	'g': {0x00, 0x0F, 0x11, 0x11, 0x0F, 0x01, 0x0E},
	'h': {0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x11},
}

func drawTinyGlyph(dst *image.RGBA, x0, y0 int, ch byte, scale int, fg color.NRGBA) {
	rows, ok := coordGlyph5x7[ch]
	if !ok {
		return
	}
	for r := 0; r < 7; r++ {
		bits := rows[r]
		for c := 0; c < 5; c++ {
			if (bits & (1 << uint(4-c))) != 0 {
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						blendPixel(dst, x0+c*scale+sx, y0+r*scale+sy, fg)
					}
				}
			}
		}
	}
}

func clamp01(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

func clampUint8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
