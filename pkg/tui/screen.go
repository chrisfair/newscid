package tui

import (
	"fmt"
	"io"
	"math"
	"strings"

	"newscid/pkg/chess"
)

// RGB represents a 24-bit TrueColor value.
type RGB struct {
	R, G, B uint8
	Set     bool
}

func RGBColor(r, g, b uint8) RGB {
	return RGB{R: r, G: g, B: b, Set: true}
}

func HexColor(hexStr string) RGB {
	r, g, b := ParseHexRGB(hexStr)
	return RGB{R: r, G: g, B: b, Set: true}
}

// Color palette for the NewScid terminal workstation.
var (
	ColBgBase       = RGB{R: 15, G: 20, B: 28, Set: true}
	ColBgPanel      = RGB{R: 21, G: 28, B: 39, Set: true}
	ColBgHeader     = RGB{R: 26, G: 36, B: 52, Set: true}
	ColBgElevated   = RGB{R: 32, G: 44, B: 62, Set: true}
	ColBgActiveRow  = RGB{R: 37, G: 62, B: 92, Set: true}
	ColBgActiveMove = RGB{R: 36, G: 92, B: 74, Set: true}
	ColBgStatus     = RGB{R: 19, G: 26, B: 37, Set: true}

	ColBorder      = RGB{R: 54, G: 70, B: 93, Set: true}
	ColBorderFocus = RGB{R: 56, G: 189, B: 248, Set: true}

	ColTextPrimary = RGB{R: 232, G: 238, B: 246, Set: true}
	ColTextMuted   = RGB{R: 142, G: 158, B: 180, Set: true}
	ColTextDim     = RGB{R: 96, G: 112, B: 135, Set: true}

	ColAccentCyan    = RGB{R: 56, G: 189, B: 248, Set: true}
	ColAccentEmerald = RGB{R: 52, G: 211, B: 153, Set: true}
	ColAccentAmber   = RGB{R: 251, G: 191, B: 36, Set: true}
	ColAccentRose    = RGB{R: 251, G: 113, B: 133, Set: true}
	ColAccentViolet  = RGB{R: 167, G: 139, B: 250, Set: true}

	ColEvalWhite = RGB{R: 236, G: 240, B: 245, Set: true}
	ColEvalBlack = RGB{R: 38, G: 45, B: 58, Set: true}
)

// Style describes cell foreground/background colors and text attributes.
type Style struct {
	FG        RGB
	BG        RGB
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
}

// Rect describes a 0-based rectangular region in terminal character cells.
type Rect struct {
	X, Y, W, H int
}

func (r Rect) Contains(col0, row0 int) bool {
	return r.W > 0 && r.H > 0 &&
		col0 >= r.X && col0 < r.X+r.W &&
		row0 >= r.Y && row0 < r.Y+r.H
}

// Cell represents a single terminal character cell.
type Cell struct {
	Ch          rune
	Style       Style
	Continuation bool
}

// Screen buffers a 2D grid of styled cells and emits diffed ANSI + Kitty Graphics frames
// wrapped in DEC Mode 2026 Synchronized Output.
type Screen struct {
	Width  int
	Height int
	Cells  []Cell
	Prev   []Cell
	Cutout Rect
}

func NewScreen(w, h int) *Screen {
	if w < 20 {
		w = 20
	}
	if h < 10 {
		h = 10
	}
	s := &Screen{
		Width:  w,
		Height: h,
		Cells:  make([]Cell, w*h),
		Prev:   make([]Cell, w*h),
	}
	s.Clear(Style{FG: ColTextPrimary, BG: ColBgBase})
	return s
}

func (s *Screen) Resize(w, h int) {
	if w < 20 {
		w = 20
	}
	if h < 10 {
		h = 10
	}
	if w == s.Width && h == s.Height {
		return
	}
	s.Width = w
	s.Height = h
	s.Cells = make([]Cell, w*h)
	s.Prev = make([]Cell, w*h)
}

func (s *Screen) Clear(st Style) {
	blank := Cell{Ch: ' ', Style: st}
	for i := range s.Cells {
		s.Cells[i] = blank
	}
}

func (s *Screen) Invalidate() {
	for i := range s.Prev {
		s.Prev[i] = Cell{}
	}
}

func runeCellWidth(r rune) int {
	if r < 32 {
		return 0
	}
	// Standard Unicode chess symbols, box drawing, block elements, geometric shapes, and arrows are 1 cell wide
	if (r >= 0x2500 && r <= 0x27bf) || (r >= 0x2190 && r <= 0x21ff) {
		return 1
	}
	// CJK / fullwidth / supplementary emoji planes
	if (r >= 0x1100 && r <= 0x115f) ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) {
		return 2
	}
	return 1
}

// DisplayWidth computes terminal column width of a string.
func DisplayWidth(str string) int {
	w := 0
	for _, r := range str {
		w += runeCellWidth(r)
	}
	return w
}

// TruncateString trims str to fit within maxCols columns, appending "…" if truncated.
func TruncateString(str string, maxCols int) string {
	if maxCols <= 0 {
		return ""
	}
	if DisplayWidth(str) <= maxCols {
		return str
	}
	if maxCols == 1 {
		return "…"
	}
	var sb strings.Builder
	cols := 0
	for _, r := range str {
		rw := runeCellWidth(r)
		if cols+rw > maxCols-1 {
			break
		}
		sb.WriteRune(r)
		cols += rw
	}
	sb.WriteRune('…')
	return sb.String()
}

func (s *Screen) SetCell(x, y int, ch rune, st Style) {
	if x < 0 || x >= s.Width || y < 0 || y >= s.Height {
		return
	}
	if ch < 32 {
		ch = ' '
	}
	idx := y*s.Width + x
	rw := runeCellWidth(ch)
	if rw == 2 && x+1 >= s.Width {
		s.Cells[idx] = Cell{Ch: ' ', Style: st}
		return
	}
	s.Cells[idx] = Cell{Ch: ch, Style: st}
	if rw == 2 && x+1 < s.Width {
		s.Cells[idx+1] = Cell{Ch: ' ', Style: st, Continuation: true}
	}
}

func (s *Screen) FillRect(r Rect, ch rune, st Style) {
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			s.SetCell(x, y, ch, st)
		}
	}
}

// WriteText writes str starting at (x, y) clipped to maxCols, returning the number of columns written.
func (s *Screen) WriteText(x, y, maxCols int, str string, st Style) int {
	if y < 0 || y >= s.Height || maxCols <= 0 {
		return 0
	}
	clipped := TruncateString(str, maxCols)
	col := 0
	for _, r := range clipped {
		rw := runeCellWidth(r)
		if rw == 0 {
			continue
		}
		if col+rw > maxCols || x+col >= s.Width {
			break
		}
		s.SetCell(x+col, y, r, st)
		col += rw
	}
	return col
}

// WritePadded writes str padded with spaces up to width columns.
func (s *Screen) WritePadded(x, y, width int, str string, st Style) {
	written := s.WriteText(x, y, width, str, st)
	for c := written; c < width; c++ {
		s.SetCell(x+c, y, ' ', st)
	}
}

// DrawBox draws a rounded Unicode border around r with an optional title and right-aligned badge.
func (s *Screen) DrawBox(r Rect, title, badge string, focused bool, bg RGB) {
	if r.W < 2 || r.H < 2 {
		return
	}
	s.FillRect(r, ' ', Style{FG: ColTextPrimary, BG: bg})

	borderCol := ColBorder
	titleCol := ColTextPrimary
	if focused {
		borderCol = ColBorderFocus
		titleCol = ColAccentCyan
	}
	bst := Style{FG: borderCol, BG: bg}

	s.SetCell(r.X, r.Y, '╭', bst)
	s.SetCell(r.X+r.W-1, r.Y, '╮', bst)
	s.SetCell(r.X, r.Y+r.H-1, '╰', bst)
	s.SetCell(r.X+r.W-1, r.Y+r.H-1, '╯', bst)

	for x := r.X + 1; x < r.X+r.W-1; x++ {
		s.SetCell(x, r.Y, '─', bst)
		s.SetCell(x, r.Y+r.H-1, '─', bst)
	}
	for y := r.Y + 1; y < r.Y+r.H-1; y++ {
		s.SetCell(r.X, y, '│', bst)
		s.SetCell(r.X+r.W-1, y, '│', bst)
	}

	if title != "" && r.W > 6 {
		label := " " + title + " "
		s.WriteText(r.X+2, r.Y, r.W-4, label, Style{FG: titleCol, BG: bg, Bold: true})
	}
	if badge != "" && r.W > 12 {
		bLabel := " " + badge + " "
		bw := DisplayWidth(bLabel)
		bx := r.X + r.W - 2 - bw
		if bx > r.X+DisplayWidth(title)+4 {
			s.WriteText(bx, r.Y, bw, bLabel, Style{FG: ColTextMuted, BG: bg})
		}
	}
}

// DrawVerticalEvalBar draws a high-contrast vertical evaluation bar (White at bottom or top matching orientation).
func (s *Screen) DrawVerticalEvalBar(x, y, height int, scoreCp, mateIn int, flipped bool) {
	if height < 4 || x < 0 || x >= s.Width {
		return
	}

	// White win fraction in [0.04, 0.96]
	whiteFrac := 0.5
	if mateIn != 0 {
		if mateIn > 0 {
			whiteFrac = 0.97
		} else {
			whiteFrac = 0.03
		}
	} else {
		// Smooth logistic curve matching standard tournament evaluation bars
		whiteFrac = 1.0 / (1.0 + math.Exp(-float64(scoreCp)/260.0))
		if whiteFrac < 0.04 {
			whiteFrac = 0.04
		}
		if whiteFrac > 0.96 {
			whiteFrac = 0.96
		}
	}

	// Top portion vs bottom portion
	topIsBlack := !flipped
	topFrac := 1.0 - whiteFrac
	if !topIsBlack {
		topFrac = whiteFrac
	}
	splitRows := int(math.Round(topFrac * float64(height)))

	topBG := ColEvalBlack
	botBG := ColEvalWhite
	if !topIsBlack {
		topBG = ColEvalWhite
		botBG = ColEvalBlack
	}

	for i := 0; i < height; i++ {
		bg := botBG
		if i < splitRows {
			bg = topBG
		}
		s.SetCell(x, y+i, ' ', Style{FG: ColTextPrimary, BG: bg})
		if x+1 < s.Width {
			s.SetCell(x+1, y+i, ' ', Style{FG: ColTextPrimary, BG: bg})
		}
	}
}

// DrawWDLBar renders a compact horizontal Win / Draw / Loss percentage bar.
func (s *Screen) DrawWDLBar(x, y, width, wWins, draws, bWins int, rowBG RGB) {
	total := wWins + draws + bWins
	if width <= 0 || total <= 0 {
		return
	}
	wCols := int(math.Round(float64(wWins) / float64(total) * float64(width)))
	dCols := int(math.Round(float64(draws) / float64(total) * float64(width)))
	if wCols+dCols > width {
		dCols = width - wCols
	}
	bCols := width - wCols - dCols
	if bCols < 0 {
		bCols = 0
	}

	pos := 0
	for i := 0; i < wCols && pos < width; i++ {
		s.SetCell(x+pos, y, '━', Style{FG: RGBColor(238, 242, 248), BG: rowBG, Bold: true})
		pos++
	}
	for i := 0; i < dCols && pos < width; i++ {
		s.SetCell(x+pos, y, '━', Style{FG: RGBColor(148, 163, 184), BG: rowBG})
		pos++
	}
	for i := 0; i < bCols && pos < width; i++ {
		s.SetCell(x+pos, y, '━', Style{FG: RGBColor(71, 85, 105), BG: rowBG, Bold: true})
		pos++
	}
}

// DrawUnicodeBoard renders a 24-bit TrueColor Unicode chessboard inside r when Kitty Graphics is disabled or unsupported.
// Returns the inner 8x8 square grid Rect (for accurate mouse click-to-square mapping).
func (s *Screen) DrawUnicodeBoard(r Rect, b *chess.Board, opt BoardRenderOptions) Rect {
	if b == nil {
		b = chess.NewBoard()
	}
	theme := LookupBoardTheme(opt.ThemeID)
	lr, lg, lb := ParseHexRGB(theme.Light)
	dr, dg, db := ParseHexRGB(theme.Dark)
	hr, hg, hb := ParseHexRGB(theme.Highlight)

	cellW := (r.W - 3) / 8
	if cellW < 2 {
		cellW = 2
	}
	if cellW > 6 {
		cellW = 6
	}
	cellH := (r.H - 1) / 8
	if cellH < 1 {
		cellH = 1
	}
	if cellH > 3 {
		cellH = 3
	}

	gridW := cellW * 8
	gridH := cellH * 8
	gx := r.X + 2 + maxInt(0, (r.W-2-gridW)/2)
	gy := r.Y + maxInt(0, (r.H-1-gridH)/2)

	legalSet := make(map[chess.Square]bool, len(opt.LegalTargets))
	for _, sq := range opt.LegalTargets {
		legalSet[sq] = true
	}
	checkSq := opt.CheckKingSq
	if checkSq >= 64 && b.InCheck(b.Turn) {
		checkSq = b.KingSquare(b.Turn)
	}

	for row := 0; row < 8; row++ {
		rankSq := DisplayCoordsToSquare(0, row, opt.Flipped)
		rankLabel := string(byte('1' + rankSq.Rank()))
		s.WriteText(gx-2, gy+row*cellH+cellH/2, 1, rankLabel, Style{FG: ColTextMuted, BG: ColBgPanel})

		for col := 0; col < 8; col++ {
			sq := DisplayCoordsToSquare(col, row, opt.Flipped)
			isLight := (sq.File()+sq.Rank())%2 == 1
			sr, sg, sb := dr, dg, db
			if isLight {
				sr, sg, sb = lr, lg, lb
			}
			if sq == opt.LastFrom || sq == opt.LastTo {
				sr = blendByte(sr, hr, 0.58)
				sg = blendByte(sg, hg, 0.58)
				sb = blendByte(sb, hb, 0.58)
			}
			if sq == opt.SelectedSq {
				sr = blendByte(sr, 76, 0.50)
				sg = blendByte(sg, 194, 0.50)
				sb = blendByte(sb, 148, 0.50)
			}
			if sq == checkSq {
				sr = blendByte(sr, 225, 0.65)
				sg = blendByte(sg, 38, 0.65)
				sb = blendByte(sb, 38, 0.65)
			}

			sqBG := RGBColor(sr, sg, sb)
			for dy := 0; dy < cellH; dy++ {
				for dx := 0; dx < cellW; dx++ {
					s.SetCell(gx+col*cellW+dx, gy+row*cellH+dy, ' ', Style{FG: ColTextPrimary, BG: sqBG})
				}
			}

			p := b.PieceAt(sq)
			centerX := gx + col*cellW + (cellW-1)/2
			centerY := gy + row*cellH + cellH/2

			if p != chess.Empty {
				glyph, fg := unicodePieceGlyph(p)
				st := Style{FG: fg, BG: sqBG, Bold: true}
				if legalSet[sq] {
					st.Underline = true
				}
				s.SetCell(centerX, centerY, glyph, st)
			} else if legalSet[sq] {
				dotFG := RGBColor(blendByte(sr, 20, 0.55), blendByte(sg, 30, 0.55), blendByte(sb, 24, 0.55))
				s.SetCell(centerX, centerY, '●', Style{FG: dotFG, BG: sqBG})
			}
		}
	}

	if gy+gridH < r.Y+r.H {
		for col := 0; col < 8; col++ {
			fileSq := DisplayCoordsToSquare(col, 7, opt.Flipped)
			fileLabel := string(byte('a' + fileSq.File()))
			s.WriteText(gx+col*cellW+(cellW-1)/2, gy+gridH, 1, fileLabel, Style{FG: ColTextMuted, BG: ColBgPanel})
		}
	}

	return Rect{X: gx, Y: gy, W: gridW, H: gridH}
}

func unicodePieceGlyph(p chess.Piece) (rune, RGB) {
	// Solid chess glyphs (♚♛♜♝♞♟) tinted crisp ivory for White and deep obsidian for Black
	// offer dramatically higher legibility on colored squares than hollow wireframe glyphs.
	whiteFG := RGBColor(255, 255, 255)
	blackFG := RGBColor(16, 18, 24)
	switch p {
	case chess.WK:
		return '♚', whiteFG
	case chess.WQ:
		return '♛', whiteFG
	case chess.WR:
		return '♜', whiteFG
	case chess.WB:
		return '♝', whiteFG
	case chess.WN:
		return '♞', whiteFG
	case chess.WP:
		return '♟', whiteFG
	case chess.BK:
		return '♚', blackFG
	case chess.BQ:
		return '♛', blackFG
	case chess.BR:
		return '♜', blackFG
	case chess.BB:
		return '♝', blackFG
	case chess.BN:
		return '♞', blackFG
	case chess.BP:
		return '♟', blackFG
	default:
		return ' ', whiteFG
	}
}

// Flush renders cell diffs plus optional Kitty Graphics commands atomically within DEC Mode 2026 Synchronized Output.
func (s *Screen) Flush(w io.Writer, forceFull bool, skipCutout bool, kittySeq string) error {
	var sb strings.Builder
	sb.Grow(8192 + len(kittySeq))

	// Begin DEC Mode 2026 Synchronized Update + hide cursor during composite
	sb.WriteString("\x1b[?2026h\x1b[?25l")

	var curStyle Style
	haveStyle := false
	curX, curY := -1, -1

	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			if skipCutout && s.Cutout.Contains(x, y) {
				continue
			}
			idx := y*s.Width + x
			cell := s.Cells[idx]
			if cell.Continuation {
				continue
			}
			if !forceFull && s.Prev[idx] == cell {
				continue
			}
			s.Prev[idx] = cell

			if x != curX || y != curY {
				fmt.Fprintf(&sb, "\x1b[%d;%dH", y+1, x+1)
				curX = x
				curY = y
			}
			if !haveStyle || curStyle != cell.Style {
				writeSGR(&sb, cell.Style)
				curStyle = cell.Style
				haveStyle = true
			}
			ch := cell.Ch
			if ch == 0 {
				ch = ' '
			}
			sb.WriteRune(ch)
			curX += runeCellWidth(ch)
		}
	}

	sb.WriteString("\x1b[0m")

	if kittySeq != "" {
		sb.WriteString(kittySeq)
	}

	// End DEC Mode 2026 Synchronized Update
	sb.WriteString("\x1b[?2026l")

	_, err := io.WriteString(w, sb.String())
	return err
}

func writeSGR(sb *strings.Builder, st Style) {
	sb.WriteString("\x1b[0")
	if st.Bold {
		sb.WriteString(";1")
	}
	if st.Dim {
		sb.WriteString(";2")
	}
	if st.Italic {
		sb.WriteString(";3")
	}
	if st.Underline {
		sb.WriteString(";4")
	}
	if st.FG.Set {
		fmt.Fprintf(sb, ";38;2;%d;%d;%d", st.FG.R, st.FG.G, st.FG.B)
	}
	if st.BG.Set {
		fmt.Fprintf(sb, ";48;2;%d;%d;%d", st.BG.R, st.BG.G, st.BG.B)
	}
	sb.WriteByte('m')
}
