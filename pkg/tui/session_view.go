package tui

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/engine"
	"newscid/pkg/version"
)

// Render composites the complete NewScid TUI frame onto scr and flushes ANSI + Kitty Graphics output to out.
func (s *Session) Render(scr *Screen, geom TermGeometry, out io.Writer, forceFull bool) error {
	scr.Resize(geom.Cols, geom.Rows)
	scr.Clear(Style{FG: ColTextPrimary, BG: ColBgBase})
	s.hitScorePlies = s.hitScorePlies[:0]
	s.hitListStartRow = 0
	s.hitListRowCount = 0

	w := scr.Width
	h := scr.Height

	// 1. Top Application Header Bar (row 0)
	s.renderTopBar(scr)

	// 2. Compute left (Board) vs right (Inspector) split
	contentY := 1
	contentH := maxInt(6, h-3) // leave bottom 2 rows for command/status + shortcut legend

	// Determine optimal Kitty/Unicode board dimensions inside left pane
	cellAspect := 2.0 // typical monospace cell height:width ratio
	if geom.XPixel > 0 && geom.YPixel > 0 && geom.Cols > 0 && geom.Rows > 0 {
		cw := float64(geom.XPixel) / float64(geom.Cols)
		ch := float64(geom.YPixel) / float64(geom.Rows)
		if cw > 2 && ch > 2 {
			ratio := ch / cw
			if ratio >= 1.3 && ratio <= 2.8 {
				cellAspect = ratio
			}
		}
	}

	maxBoardRows := contentH - 6 // borders (2) + top/bottom player plates (2) + ECO/info (2)
	if maxBoardRows < 8 {
		maxBoardRows = 8
	}
	if maxBoardRows > 24 {
		maxBoardRows = 24
	}
	// Keep left pane <= ~48% of terminal width so inspector has plenty of room
	maxLeftW := maxInt(30, (w*48)/100)
	maxBoardColsByW := maxLeftW - 6 // borders (2) + eval bar (3) + padding (1)
	boardRows := maxBoardRows
	boardCols := int(math.Round(float64(boardRows) * cellAspect))
	if boardCols > maxBoardColsByW {
		boardCols = maxInt(16, maxBoardColsByW)
		boardRows = maxInt(8, int(math.Round(float64(boardCols)/cellAspect)))
	}
	// Make boardCols a clean even integer >= 16 when space permits
	if boardCols >= 16 && boardCols%2 == 1 {
		boardCols--
	}

	leftW := minInt(w-24, boardCols+7)
	if leftW < 26 {
		leftW = minInt(w/2, 26)
	}
	rightX := leftW
	rightW := maxInt(20, w-rightX)

	leftBox := Rect{X: 0, Y: contentY, W: leftW, H: contentH}
	rightBox := Rect{X: rightX, Y: contentY, W: rightW, H: contentH}

	// 3. Render Left Chessboard Pane
	boardPlacementRect, sqPx := s.renderBoardPane(scr, leftBox, boardCols, boardRows, geom)

	// 4. Render Right Multi-Tab Inspector Pane
	s.renderRightPane(scr, rightBox)

	// 5. Render Bottom Prompt / Status & Key Legend (rows h-2, h-1)
	s.renderBottomBars(scr)

	// 6. Optional Help Overlay Modal
	modalActive := s.Mode == ModeHelpModal
	if modalActive {
		s.renderHelpModal(scr)
	}

	// 7. Manage Kitty Graphics Protocol Placement & Cutout
	var kittySeq string
	skipCutout := false

	if s.UseKitty && !modalActive && boardPlacementRect.W >= 8 && boardPlacementRect.H >= 4 {
		scr.Cutout = boardPlacementRect
		lastFrom, lastTo := s.LastMoveSquares()
		targets, capMask := s.LegalTargetsFrom(s.SelectedSq)
		checkSq := chess.NoSquare
		if s.Board.InCheck(s.Board.Turn) {
			checkSq = s.Board.KingSquare(s.Board.Turn)
		}

		opt := BoardRenderOptions{
			ThemeID:      s.ThemeID,
			PieceSetID:   s.PieceSetID,
			SquarePx:     sqPx,
			Flipped:      s.Flipped,
			SelectedSq:   s.SelectedSq,
			LastFrom:     lastFrom,
			LastTo:       lastTo,
			LegalTargets: targets,
			CaptureMask:  capMask,
			Arrows:       s.ActiveBoardArrows(),
			ShowCoords:   true,
			CheckKingSq:  checkSq,
		}
		hash := opt.Hash(s.Board)
		rectChanged := boardPlacementRect != s.lastBoardRect

		if forceFull || !s.kittyPlaced || rectChanged || hash != s.lastBoardHash || len(s.lastBoardPNG) == 0 {
			if hash != s.lastBoardHash || len(s.lastBoardPNG) == 0 {
				pngBytes, err := RenderBoardPNG(s.Board, opt)
				if err == nil {
					s.lastBoardPNG = pngBytes
					s.lastBoardHash = hash
				}
			}
			if len(s.lastBoardPNG) > 0 {
				kittySeq = EncodeKittyPNGPlacement(
					KittyBoardImageID,
					KittyBoardPlacementID,
					boardPlacementRect.Y+1,
					boardPlacementRect.X+1,
					boardPlacementRect.W,
					boardPlacementRect.H,
					s.lastBoardPNG,
				)
				s.lastBoardRect = boardPlacementRect
				s.kittyPlaced = true
			}
			// On full redraw, let background cells paint first; on incremental board updates, skip writing spaces over cutout
			skipCutout = !forceFull && !rectChanged
		} else {
			// Kitty image is already placed and unchanged — do not overwrite its cells
			skipCutout = true
		}
	} else {
		scr.Cutout = Rect{}
		if s.kittyPlaced {
			kittySeq = EncodeKittyDeleteImage(KittyBoardImageID)
			s.kittyPlaced = false
			s.lastBoardHash = 0
		}
	}

	return scr.Flush(out, forceFull, skipCutout, kittySeq)
}

func (s *Session) renderTopBar(scr *Screen) {
	w := scr.Width
	scr.FillRect(Rect{X: 0, Y: 0, W: w, H: 1}, ' ', Style{FG: ColTextPrimary, BG: ColBgHeader})

	x := 1
	title := fmt.Sprintf("♞ NewScid %s", version.Short())
	x += scr.WriteText(x, 0, w-x, title, Style{FG: ColAccentCyan, BG: ColBgHeader, Bold: true})
	x += scr.WriteText(x, 0, w-x, " │ ", Style{FG: ColBorder, BG: ColBgHeader})

	if s.UseKitty {
		x += scr.WriteText(x, 0, w-x, "KITTY HD", Style{FG: ColBgBase, BG: ColAccentEmerald, Bold: true})
	} else {
		x += scr.WriteText(x, 0, w-x, "UNICODE RGB", Style{FG: ColBgBase, BG: ColAccentAmber, Bold: true})
	}
	x += scr.WriteText(x, 0, w-x, " ", Style{FG: ColTextMuted, BG: ColBgHeader})
	x += scr.WriteText(x, 0, w-x, fmt.Sprintf("(%s · %s)", s.PieceSetID, s.ThemeID), Style{FG: ColTextMuted, BG: ColBgHeader})

	x += scr.WriteText(x, 0, w-x, " │ ", Style{FG: ColBorder, BG: ColBgHeader})
	dbBadge := "Scratch Board"
	if s.ActiveDBName != "" {
		total := s.TotalMatched
		if s.ScidDB != nil && s.ScidDB.IsOpen && s.FilterQuery == "" {
			total = s.ScidDB.NumGames()
		}
		if s.ActiveGameIndex > 0 {
			dbBadge = fmt.Sprintf("DB: %s [#%d/%d]", s.ActiveDBName, s.ActiveGameIndex, total)
		} else {
			dbBadge = fmt.Sprintf("DB: %s (%d games)", s.ActiveDBName, total)
		}
	}
	x += scr.WriteText(x, 0, w-x, dbBadge, Style{FG: ColTextPrimary, BG: ColBgHeader, Bold: true})

	// Right-aligned Engine + Help badge
	s.mu.Lock()
	engOn := s.EngineEnabled
	engName := s.EngineName
	eval := s.LastEval
	s.mu.Unlock()

	rightBadge := " [?] Help "
	if engOn {
		scoreTxt := formatEvalCompact(eval.ScoreCp, eval.MateIn)
		rightBadge = fmt.Sprintf(" ⚡ %s d%d %s │ [?] Help ", TruncateString(engName, 14), eval.Depth, scoreTxt)
	} else {
		rightBadge = " Engine: OFF [e] │ [?] Help "
	}
	rw := DisplayWidth(rightBadge)
	if w-rw > x+2 {
		col := ColTextMuted
		if engOn {
			col = ColAccentEmerald
		}
		scr.WriteText(w-rw, 0, rw, rightBadge, Style{FG: col, BG: ColBgHeader, Bold: engOn})
	}
}

func (s *Session) renderBoardPane(scr *Screen, box Rect, boardCols, boardRows int, geom TermGeometry) (Rect, int) {
	turnStr := "White to move"
	if s.Board.Turn == chess.Black {
		turnStr = "Black to move"
	}
	if s.Board.IsCheckmate() {
		if s.Board.Turn == chess.White {
			turnStr = "Checkmate · 0-1"
		} else {
			turnStr = "Checkmate · 1-0"
		}
	} else if s.Board.IsStalemate() {
		turnStr = "Stalemate · ½-½"
	} else if s.Board.InCheck(s.Board.Turn) {
		turnStr += " · CHECK!"
	}

	scr.DrawBox(box, "Chessboard", turnStr, true, ColBgPanel)

	innerX := box.X + 2
	innerW := maxInt(10, box.W-4)
	topPlateY := box.Y + 1

	topColor := chess.Black
	botColor := chess.White
	if s.Flipped {
		topColor = chess.White
		botColor = chess.Black
	}

	s.renderPlayerPlate(scr, innerX, topPlateY, innerW, topColor)

	boardAreaY := topPlateY + 1
	evalBarX := innerX
	boardX := innerX + 3

	s.mu.Lock()
	eval := s.LastEval
	engOn := s.EngineEnabled
	s.mu.Unlock()

	scoreCp := 0
	mateIn := 0
	if engOn {
		scoreCp = eval.ScoreCp
		mateIn = eval.MateIn
	} else if s.ImbalanceRep != nil {
		scoreCp = s.ImbalanceRep.EstimatedEvalCp
	}

	var placementRect Rect
	sqPx := 72

	if s.UseKitty {
		availCols := maxInt(8, box.W-6)
		bCols := minInt(boardCols, availCols)
		bRows := minInt(boardRows, maxInt(6, box.H-6))
		bx := boardX + maxInt(0, (availCols-bCols)/2)
		by := boardAreaY + maxInt(0, (box.H-6-bRows)/2)

		placementRect = Rect{X: bx, Y: by, W: bCols, H: bRows}
		s.hitBoardRect = placementRect
		s.hitBoardIsKitty = true

		scr.DrawVerticalEvalBar(evalBarX, by, bRows, scoreCp, mateIn, s.Flipped)

		// Choose crisp high-DPI pixel square size matching terminal pixel density if reported
		if geom.XPixel > 0 && geom.Cols > 0 {
			estBoardPx := (geom.XPixel * bCols) / geom.Cols
			cand := estBoardPx / 8
			if cand >= 48 && cand <= 144 {
				sqPx = (cand / 8) * 8
			}
		}

		botPlateY := by + bRows
		if botPlateY < box.Y+box.H-1 {
			s.renderPlayerPlate(scr, innerX, botPlateY, innerW, botColor)
		}
		ecoY := botPlateY + 1
		if ecoY < box.Y+box.H-1 {
			s.renderBoardFooterStrip(scr, innerX, ecoY, innerW)
		}
	} else {
		unicodeArea := Rect{X: boardX, Y: boardAreaY, W: maxInt(16, box.W-6), H: maxInt(8, box.H-5)}
		lastFrom, lastTo := s.LastMoveSquares()
		targets, capMask := s.LegalTargetsFrom(s.SelectedSq)
		uGrid := scr.DrawUnicodeBoard(unicodeArea, s.Board, BoardRenderOptions{
			ThemeID:      s.ThemeID,
			Flipped:      s.Flipped,
			SelectedSq:   s.SelectedSq,
			LastFrom:     lastFrom,
			LastTo:       lastTo,
			LegalTargets: targets,
			CaptureMask:  capMask,
			ShowCoords:   true,
		})
		s.hitBoardRect = uGrid
		s.hitBoardIsKitty = false

		scr.DrawVerticalEvalBar(evalBarX, uGrid.Y, uGrid.H, scoreCp, mateIn, s.Flipped)

		botPlateY := minInt(box.Y+box.H-2, uGrid.Y+uGrid.H+1)
		if botPlateY > boardAreaY && botPlateY < box.Y+box.H-1 {
			s.renderPlayerPlate(scr, innerX, botPlateY, innerW, botColor)
		}
		if botPlateY+1 < box.Y+box.H-1 {
			s.renderBoardFooterStrip(scr, innerX, botPlateY+1, innerW)
		}
	}

	return placementRect, sqPx
}

func (s *Session) renderPlayerPlate(scr *Screen, x, y, width int, side chess.Color) {
	name := s.Game.Tag("White")
	elo := s.Game.Tag("WhiteElo")
	title := s.Game.Tag("WhiteTitle")
	swatch := "⬜"
	if side == chess.Black {
		name = s.Game.Tag("Black")
		elo = s.Game.Tag("BlackElo")
		title = s.Game.Tag("BlackTitle")
		swatch = "⬛"
	}
	if name == "" || name == "?" {
		if side == chess.White {
			name = "White"
		} else {
			name = "Black"
		}
	}

	active := s.Board.Turn == side
	bullet := "  "
	if active {
		bullet = "▶ "
	}

	label := bullet + swatch + " "
	if title != "" && title != "?" {
		label += title + " "
	}
	label += name
	if elo != "" && elo != "?" && elo != "0" {
		label += fmt.Sprintf(" (%s)", elo)
	}

	// Material delta badge
	matBadge := ""
	if s.ImbalanceRep != nil {
		diff := s.ImbalanceRep.MaterialDelta
		if side == chess.Black {
			diff = -diff
		}
		if diff > 0 {
			matBadge = fmt.Sprintf("+%d", diff)
		}
	}

	fg := ColTextMuted
	if active {
		fg = ColTextPrimary
	}
	scr.WritePadded(x, y, width, "", Style{FG: fg, BG: ColBgPanel})
	used := scr.WriteText(x, y, width, label, Style{FG: fg, BG: ColBgPanel, Bold: active})
	if matBadge != "" && used+len(matBadge)+2 < width {
		scr.WriteText(x+used+2, y, width-used-2, matBadge, Style{FG: ColAccentEmerald, BG: ColBgPanel, Bold: true})
	}
}

func (s *Session) renderBoardFooterStrip(scr *Screen, x, y, width int) {
	ecoStr := ""
	if s.CurrentECO != nil && s.CurrentECO.Code != "" {
		ecoStr = fmt.Sprintf("%s · %s", s.CurrentECO.Code, s.CurrentECO.Name)
	}
	plyStr := fmt.Sprintf("Ply %d/%d", s.CurPly, len(s.Path))
	scr.WritePadded(x, y, width, "", Style{FG: ColTextMuted, BG: ColBgPanel})
	scr.WriteText(x, y, maxInt(0, width-len(plyStr)-2), ecoStr, Style{FG: ColAccentAmber, BG: ColBgPanel})
	pw := DisplayWidth(plyStr)
	if width > pw+2 {
		scr.WriteText(x+width-pw, y, pw, plyStr, Style{FG: ColTextMuted, BG: ColBgPanel})
	}
}

func (s *Session) renderRightPane(scr *Screen, box Rect) {
	resBadge := string(s.Game.Result)
	scr.DrawBox(box, "Workstation Inspector", resBadge, false, ColBgPanel)

	// Render clickable tab strip at box.Y + 1
	tabY := box.Y + 1
	tx := box.X + 2
	maxX := box.X + box.W - 2
	scr.WritePadded(tx, tabY, maxInt(0, maxX-tx), "", Style{FG: ColTextMuted, BG: ColBgPanel})

	for i := TabID(0); i < tabCount; i++ {
		lbl := " " + tabTitles[i] + " "
		lw := DisplayWidth(lbl)
		if tx+lw > maxX {
			s.hitTabRects[i] = Rect{}
			continue
		}
		st := Style{FG: ColTextMuted, BG: ColBgElevated}
		if i == s.ActiveTab {
			st = Style{FG: ColBgBase, BG: ColAccentCyan, Bold: true}
		}
		scr.WriteText(tx, tabY, lw, lbl, st)
		s.hitTabRects[i] = Rect{X: tx, Y: tabY, W: lw, H: 1}
		tx += lw + 1
	}

	// Horizontal divider below tabs
	divY := box.Y + 2
	if divY < box.Y+box.H-1 {
		for cx := box.X + 1; cx < box.X+box.W-1; cx++ {
			scr.SetCell(cx, divY, '─', Style{FG: ColBorder, BG: ColBgPanel})
		}
	}

	body := Rect{
		X: box.X + 2,
		Y: box.Y + 3,
		W: maxInt(10, box.W-4),
		H: maxInt(2, box.H-4),
	}

	switch s.ActiveTab {
	case TabScore:
		s.renderScoreAndEngineTab(scr, body)
	case TabTree:
		s.renderOpeningTreeTab(scr, body)
	case TabGames:
		s.renderGamesTab(scr, body)
	case TabBases:
		s.renderBasesTab(scr, body)
	case TabImbalance:
		s.renderImbalanceTab(scr, body)
	case TabForensics:
		s.renderForensicsTab(scr, body)
	}
}

func (s *Session) renderScoreAndEngineTab(scr *Screen, r Rect) {
	// Split vertically: top ~58% Scorebook, bottom ~42% Live UCI Engine
	engH := maxInt(6, (r.H*40)/100)
	if r.H < 12 {
		engH = maxInt(4, r.H/2)
	}
	scoreH := maxInt(3, r.H-engH-1)

	// Header metadata line (Event / Site / Date)
	ev := s.Game.Tag("Event")
	dt := s.Game.Tag("Date")
	hdr := fmt.Sprintf("%s  ·  %s  ·  Notation: %s [n]", ev, dt, strings.ToUpper(string(s.Notation)))
	scr.WritePadded(r.X, r.Y, r.W, hdr, Style{FG: ColTextMuted, BG: ColBgPanel})

	// Build wrapped scorebook tokens with ply indices so active move auto-scrolls into view
	type moveTok struct {
		Text   string
		Ply    int // >0 for clickable move token, 0 for move number / comment / variation text
		Active bool
		Style  Style
	}
	var toks []moveTok
	b := s.startBoardForGame()
	for i, pf := range s.Path {
		ply := i + 1
		if (i % 2) == 0 {
			toks = append(toks, moveTok{
				Text:  fmt.Sprintf("%d.", (i/2)+1),
				Style: Style{FG: ColTextDim, BG: ColBgPanel},
			})
		}
		moveStr := b.MoveToNotation(pf.Move, s.Notation)
		for _, nag := range pf.Node.NAGs {
			moveStr += chess.FormatNAG(nag)
		}
		active := ply == s.CurPly
		st := Style{FG: ColTextPrimary, BG: ColBgPanel, Bold: active}
		if active {
			st = Style{FG: ColTextPrimary, BG: ColBgActiveMove, Bold: true}
		}
		toks = append(toks, moveTok{
			Text:   moveStr,
			Ply:    ply,
			Active: active,
			Style:  st,
		})

		if len(pf.Node.Variations) > 0 {
			var vParts []string
			for _, v := range pf.Node.Variations {
				if v != nil && v.SAN != "" {
					vParts = append(vParts, v.SAN)
				}
			}
			if len(vParts) > 0 {
				toks = append(toks, moveTok{
					Text:  fmt.Sprintf("[%s]", strings.Join(vParts, "|")),
					Style: Style{FG: ColAccentViolet, BG: ColBgPanel},
				})
			}
		}
		if pf.Node.Comment != "" {
			cleanCmt := strings.Join(strings.Fields(pf.Node.Comment), " ")
			toks = append(toks, moveTok{
				Text:  "{" + TruncateString(cleanCmt, 28) + "}",
				Style: Style{FG: ColAccentCyan, BG: ColBgPanel, Italic: true},
			})
		}
		b.MakeMove(pf.Move)
	}

	if len(toks) == 0 {
		scr.WriteText(r.X, r.Y+2, r.W, "No moves played yet. Click/drag pieces or press 'm' to enter SAN/UCI moves.", Style{FG: ColTextDim, BG: ColBgPanel, Italic: true})
	} else {
		// Wrap tokens into visual lines
		type tokPlacement struct {
			tok moveTok
			x   int
			w   int
		}
		var lines [][]tokPlacement
		var curLine []tokPlacement
		cx := 0
		activeLine := 0

		for _, tk := range toks {
			tw := DisplayWidth(tk.Text)
			space := 1
			if len(curLine) == 0 {
				space = 0
			}
			if cx+space+tw > r.W && len(curLine) > 0 {
				lines = append(lines, curLine)
				curLine = nil
				cx = 0
				space = 0
			}
			px := cx + space
			curLine = append(curLine, tokPlacement{tok: tk, x: px, w: tw})
			if tk.Active {
				activeLine = len(lines)
			}
			cx = px + tw
		}
		if len(curLine) > 0 {
			lines = append(lines, curLine)
		}

		availRows := maxInt(1, scoreH-1)
		startLine := 0
		if activeLine >= availRows {
			startLine = activeLine - availRows + 2
			if startLine > len(lines)-availRows {
				startLine = maxInt(0, len(lines)-availRows)
			}
		}

		for rowOff := 0; rowOff < availRows && startLine+rowOff < len(lines); rowOff++ {
			sy := r.Y + 1 + rowOff
			for _, tp := range lines[startLine+rowOff] {
				txt := tp.tok.Text
				if tp.tok.Active {
					scr.WriteText(r.X+tp.x, sy, tp.w, txt, tp.tok.Style)
				} else {
					scr.WriteText(r.X+tp.x, sy, tp.w, txt, tp.tok.Style)
				}
				if tp.tok.Ply > 0 {
					s.hitScorePlies = append(s.hitScorePlies, scoreHitSpan{
						Row: sy,
						X0:  r.X + tp.x,
						X1:  r.X + tp.x + tp.w,
						Ply: tp.tok.Ply,
					})
				}
			}
		}
	}

	// Divider between Scorebook and UCI Engine panel
	sepY := r.Y + scoreH
	for cx := r.X; cx < r.X+r.W; cx++ {
		scr.SetCell(cx, sepY, '─', Style{FG: ColBorder, BG: ColBgPanel})
	}

	// Bottom sub-pane: Live UCI Engine Analysis
	ey := sepY + 1
	s.mu.Lock()
	engOn := s.EngineEnabled
	engName := s.EngineName
	eval := s.LastEval
	mpv := s.MultiPV
	s.mu.Unlock()

	if !engOn {
		scr.WriteText(r.X, ey, r.W, "⚡ UCI Engine Disabled — press [e] to start Stockfish / Scid Engine", Style{FG: ColTextMuted, BG: ColBgPanel, Bold: true})
		if ey+1 < r.Y+r.H {
			scr.WriteText(r.X, ey+1, r.W, "Use [=]/[-] to adjust Multi-PV (1-5) and [Space] to play best move on board.", Style{FG: ColTextDim, BG: ColBgPanel})
		}
		return
	}

	npsK := eval.NPS / 1000
	hdrLine := fmt.Sprintf("⚡ %s  ·  Depth %d  ·  %dk nps  ·  MultiPV %d [+/-]  ·  [Space] Play",
		engName, eval.Depth, npsK, mpv)
	scr.WritePadded(r.X, ey, r.W, hdrLine, Style{FG: ColAccentEmerald, BG: ColBgPanel, Bold: true})

	lines := eval.Lines
	if len(lines) == 0 && len(eval.PV) > 0 {
		san, formatted := engine.FormatPV(s.Board.FEN(), eval.PV)
		lines = []engine.PVLine{{
			MultiPV:     1,
			Depth:       eval.Depth,
			ScoreCp:     eval.ScoreCp,
			MateIn:      eval.MateIn,
			SAN:         san,
			PVFormatted: formatted,
		}}
	}

	for idx := 0; idx < engH-1 && idx < len(lines); idx++ {
		ln := lines[idx]
		ry := ey + 1 + idx
		if ry >= r.Y+r.H {
			break
		}
		scTxt := formatEvalCompact(ln.ScoreCp, ln.MateIn)
		scoreCol := ColAccentCyan
		if idx == 0 {
			scoreCol = ColAccentEmerald
		}
		prefix := fmt.Sprintf("#%d %-6s ", ln.MultiPV, scTxt)
		pw := scr.WriteText(r.X, ry, r.W, prefix, Style{FG: scoreCol, BG: ColBgPanel, Bold: idx == 0})
		pvText := ln.PVFormatted
		if pvText == "" {
			pvText = strings.Join(ln.PV, " ")
		}
		scr.WriteText(r.X+pw, ry, maxInt(0, r.W-pw), pvText, Style{FG: ColTextPrimary, BG: ColBgPanel})
	}
}

func (s *Session) renderOpeningTreeTab(scr *Screen, r Rect) {
	s.mu.Lock()
	res := s.TreeResult
	busy := s.TreeBusy
	cursor := s.TreeCursor
	s.mu.Unlock()

	total := 0
	if res != nil {
		total = res.TotalGames
	}
	status := fmt.Sprintf("Position Opening Tree  ·  %d matching games", total)
	if busy {
		status += "  (scanning…)"
	}
	scr.WritePadded(r.X, r.Y, r.W, status, Style{FG: ColAccentAmber, BG: ColBgPanel, Bold: true})

	hdrY := r.Y + 1
	scr.WritePadded(r.X, hdrY, r.W, "Key  Move      Games    Score%   W / D / L Distribution", Style{FG: ColTextMuted, BG: ColBgElevated, Bold: true})

	if res == nil || len(res.Moves) == 0 {
		msg := "No games in the active database reach this position."
		if s.ActiveDBName == "" {
			msg = "Open a Scid .si4 or .pgn database ([4:Bases] or :open <path>) to populate the Opening Tree."
		}
		scr.WriteText(r.X, r.Y+3, r.W, msg, Style{FG: ColTextDim, BG: ColBgPanel, Italic: true})
		return
	}

	avail := maxInt(1, r.H-2)
	if cursor >= len(res.Moves) {
		cursor = len(res.Moves) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	start := 0
	if cursor >= avail {
		start = cursor - avail + 1
	}

	s.hitListStartRow = r.Y + 2
	s.hitListRowCount = 0

	for i := 0; i < avail && start+i < len(res.Moves); i++ {
		idx := start + i
		mv := res.Moves[idx]
		ry := r.Y + 2 + i
		rowBG := ColBgPanel
		if idx == cursor {
			rowBG = ColBgActiveRow
		}
		keyHint := "   "
		if idx < 9 {
			keyHint = fmt.Sprintf("[%c]", 'a'+idx)
		}
		line := fmt.Sprintf("%-4s %-8s %6d   %5.1f%%  ", keyHint, mv.Move, mv.Count, mv.ScorePct)
		scr.WritePadded(r.X, ry, r.W, line, Style{FG: ColTextPrimary, BG: rowBG, Bold: idx == cursor})
		barX := r.X + 32
		barW := minInt(24, maxInt(0, r.W-34))
		if barW >= 6 {
			scr.DrawWDLBar(barX, ry, barW, mv.WhiteWins, mv.Draws, mv.BlackWins, rowBG)
		}
		s.hitListRowCount++
	}
}

func (s *Session) renderGamesTab(scr *Screen, r Rect) {
	filterInfo := "All games (press [/] to filter: player:Carlsen eco:B90 minelo:2600 pos)"
	if s.FilterQuery != "" {
		filterInfo = fmt.Sprintf("Filter: %q  ([/] edit, [:filter] clear)", s.FilterQuery)
	}
	hdr := fmt.Sprintf("Matched %d games  ·  %s", s.TotalMatched, filterInfo)
	scr.WritePadded(r.X, r.Y, r.W, hdr, Style{FG: ColAccentCyan, BG: ColBgPanel, Bold: true})

	scr.WritePadded(r.X, r.Y+1, r.W, "   #  White                  Black                  Res      ECO  Year  Mvs", Style{FG: ColTextMuted, BG: ColBgElevated, Bold: true})

	if len(s.GameSummaries) == 0 {
		msg := "No games loaded. Switch to [4:Bases] or run :open <database.si4|file.pgn>."
		if s.ActiveDBName != "" {
			msg = "No games match the current filter."
		}
		scr.WriteText(r.X, r.Y+3, r.W, msg, Style{FG: ColTextDim, BG: ColBgPanel, Italic: true})
		return
	}

	avail := maxInt(1, r.H-2)
	if s.GameListCursor >= len(s.GameSummaries) {
		s.GameListCursor = len(s.GameSummaries) - 1
	}
	if s.GameListCursor < 0 {
		s.GameListCursor = 0
	}
	start := 0
	if s.GameListCursor >= avail {
		start = s.GameListCursor - avail + 1
	}

	s.hitListStartRow = r.Y + 2
	s.hitListRowCount = 0

	for i := 0; i < avail && start+i < len(s.GameSummaries); i++ {
		idx := start + i
		gs := s.GameSummaries[idx]
		ry := r.Y + 2 + i
		rowBG := ColBgPanel
		if idx == s.GameListCursor {
			rowBG = ColBgActiveRow
		}
		year := gs.Date
		if len(year) >= 4 {
			year = year[:4]
		}
		rowTxt := fmt.Sprintf("%4d  %-21s  %-21s  %-7s  %-3s  %-4s  %3d",
			gs.ID,
			TruncateString(gs.White, 21),
			TruncateString(gs.Black, 21),
			string(gs.Result),
			gs.ECO,
			year,
			gs.Moves,
		)
		fg := ColTextPrimary
		if gs.ID == s.ActiveGameIndex {
			fg = ColAccentEmerald
		}
		scr.WritePadded(r.X, ry, r.W, rowTxt, Style{FG: fg, BG: rowBG, Bold: idx == s.GameListCursor || gs.ID == s.ActiveGameIndex})
		s.hitListRowCount++
	}
}

func (s *Session) renderBasesTab(scr *Screen, r Rect) {
	scr.WritePadded(r.X, r.Y, r.W, "Discovered Scid (.si4) & PGN Databases — [↑/↓] Select, [Enter] Open, [:open <path>]", Style{FG: ColAccentCyan, BG: ColBgPanel, Bold: true})
	scr.WritePadded(r.X, r.Y+1, r.W, "Type  Name                    Games     Size      Category / Path", Style{FG: ColTextMuted, BG: ColBgElevated, Bold: true})

	if len(s.DiscoveredBases) == 0 {
		scr.WriteText(r.X, r.Y+3, r.W, "No databases discovered yet. Pass a path on CLI or type :open /path/to/games.si4", Style{FG: ColTextDim, BG: ColBgPanel, Italic: true})
		return
	}

	avail := maxInt(1, r.H-2)
	if s.BasesCursor >= len(s.DiscoveredBases) {
		s.BasesCursor = len(s.DiscoveredBases) - 1
	}
	if s.BasesCursor < 0 {
		s.BasesCursor = 0
	}
	start := 0
	if s.BasesCursor >= avail {
		start = s.BasesCursor - avail + 1
	}

	s.hitListStartRow = r.Y + 2
	s.hitListRowCount = 0

	for i := 0; i < avail && start+i < len(s.DiscoveredBases); i++ {
		idx := start + i
		db := s.DiscoveredBases[idx]
		ry := r.Y + 2 + i
		rowBG := ColBgPanel
		if idx == s.BasesCursor {
			rowBG = ColBgActiveRow
		}
		gamesCol := "-"
		if db.NumGames > 0 {
			gamesCol = strconv.Itoa(db.NumGames)
		}
		line := fmt.Sprintf("%-4s  %-22s  %8s  %-8s  %s",
			strings.ToUpper(db.Type),
			TruncateString(db.Name, 22),
			gamesCol,
			db.SizeStr,
			db.BasePath,
		)
		scr.WritePadded(r.X, ry, r.W, line, Style{FG: ColTextPrimary, BG: rowBG, Bold: idx == s.BasesCursor})
		s.hitListRowCount++
	}
}

func (s *Session) renderImbalanceTab(scr *Screen, r Rect) {
	rep := s.ImbalanceRep
	if rep == nil {
		scr.WriteText(r.X, r.Y, r.W, "Positional imbalance unavailable.", Style{FG: ColTextDim, BG: ColBgPanel})
		return
	}

	scr.WritePadded(r.X, r.Y, r.W, fmt.Sprintf("Positional & Material Imbalance  ·  Est. Eval %s", formatEvalCompact(rep.EstimatedEvalCp, 0)), Style{FG: ColAccentEmerald, BG: ColBgPanel, Bold: true})
	scr.WritePadded(r.X, r.Y+1, r.W, rep.PositionalSummary, Style{FG: ColAccentAmber, BG: ColBgPanel})

	scr.WritePadded(r.X, r.Y+3, r.W, "Metric                        White             Black             Advantage", Style{FG: ColTextMuted, BG: ColBgElevated, Bold: true})

	rows := []struct {
		Label string
		WVal  string
		BVal  string
		Adv   string
	}{
		{
			"Material Points",
			strconv.Itoa(rep.White.MaterialPoints),
			strconv.Itoa(rep.Black.MaterialPoints),
			deltaLabel(rep.White.MaterialPoints - rep.Black.MaterialPoints),
		},
		{
			"Bishop Pair",
			yesNo(rep.White.HasBishopPair),
			yesNo(rep.Black.HasBishopPair),
			boolAdv(rep.White.HasBishopPair, rep.Black.HasBishopPair),
		},
		{
			"King Shelter Safety",
			fmt.Sprintf("%d / 100", rep.White.KingSafety),
			fmt.Sprintf("%d / 100", rep.Black.KingSafety),
			deltaLabel(rep.White.KingSafety - rep.Black.KingSafety),
		},
		{
			"Passed Pawns",
			strconv.Itoa(rep.White.PawnStructure.PassedPawns),
			strconv.Itoa(rep.Black.PawnStructure.PassedPawns),
			deltaLabel(rep.White.PawnStructure.PassedPawns - rep.Black.PawnStructure.PassedPawns),
		},
		{
			"Isolated Pawns",
			strconv.Itoa(rep.White.PawnStructure.IsolatedPawns),
			strconv.Itoa(rep.Black.PawnStructure.IsolatedPawns),
			deltaLabel(rep.Black.PawnStructure.IsolatedPawns - rep.White.PawnStructure.IsolatedPawns),
		},
		{
			"Doubled Pawns",
			strconv.Itoa(rep.White.PawnStructure.DoubledPawns),
			strconv.Itoa(rep.Black.PawnStructure.DoubledPawns),
			deltaLabel(rep.Black.PawnStructure.DoubledPawns - rep.White.PawnStructure.DoubledPawns),
		},
		{
			"Enemy Half Territory",
			fmt.Sprintf("%d sq", rep.White.ControlledTerritory),
			fmt.Sprintf("%d sq", rep.Black.ControlledTerritory),
			deltaLabel(rep.White.ControlledTerritory - rep.Black.ControlledTerritory),
		},
		{
			"Open Files Controlled",
			strconv.Itoa(rep.White.OpenFilesControlled),
			strconv.Itoa(rep.Black.OpenFilesControlled),
			deltaLabel(rep.White.OpenFilesControlled - rep.Black.OpenFilesControlled),
		},
	}

	for i, row := range rows {
		ry := r.Y + 4 + i
		if ry >= r.Y+r.H {
			break
		}
		txt := fmt.Sprintf("%-28s  %-16s  %-16s  %s", row.Label, row.WVal, row.BVal, row.Adv)
		scr.WritePadded(r.X, ry, r.W, txt, Style{FG: ColTextPrimary, BG: ColBgPanel})
	}

	fenY := r.Y + 4 + len(rows) + 1
	if fenY < r.Y+r.H {
		scr.WriteText(r.X, fenY, r.W, "FEN: "+s.Board.FEN(), Style{FG: ColTextDim, BG: ColBgPanel})
	}
}

func (s *Session) renderForensicsTab(scr *Screen, r Rect) {
	s.mu.Lock()
	busy := s.ForensicsBusy
	mode := s.ForensicsMode
	fp := s.FairPlayReport
	sc := s.ScoutReport
	scroll := s.ForensicsScroll
	s.mu.Unlock()

	scr.WritePadded(r.X, r.Y, r.W, "Forensics & Preparation — [F] FairPlay Dossier  [S] Opponent Scout  [:fairplay <name>]  [:scout <name>]", Style{FG: ColAccentViolet, BG: ColBgPanel, Bold: true})

	if busy {
		scr.WriteText(r.X, r.Y+2, r.W, fmt.Sprintf("Analyzing games for %q…", s.ForensicsTarget), Style{FG: ColAccentAmber, BG: ColBgPanel})
		return
	}

	var lines []string
	if mode == "fairplay" && fp != nil {
		lines = append(lines,
			fmt.Sprintf("FairPlay Dossier: %s  (%d games across %d active days)", fp.PlayerName, fp.TotalGamesAnalyzed, fp.TotalDaysActive),
			fmt.Sprintf("Overall Issue Score: %.1f / 100  ·  Classification: %s", fp.OverallIssueScore, fp.OverallClassification),
			fmt.Sprintf("Highest-Risk Control: %s (%.1f / 100)", fp.HighestRiskTimeControl, fp.HighestControlIssueScore),
			"",
			"Time-Control Profiles:",
		)
		for _, p := range fp.TimeControlProfiles {
			lines = append(lines, fmt.Sprintf("  • %-9s  %3d gms  IPR %4d (Δ%+d)  ACPL %5.1f  Top1 %4.1f%%  CV %.2f  Clutch %d  Score %.1f (%s)",
				p.Category, p.GamesAnalyzed, p.IntrinsicPerformanceElo, p.EloOverperformanceDelta,
				p.AvgACPL, p.Top1AgreementPct, p.AvgMoveTimeCV, p.ClutchToggleGameCount,
				p.ConsistentIssueScore, p.ConsistencyClassification))
		}
		if len(fp.KeyFindings) > 0 {
			lines = append(lines, "", "Key Forensic Findings:")
			for _, kf := range fp.KeyFindings {
				lines = append(lines, "  - "+kf)
			}
		}
	} else if mode == "scout" && sc != nil {
		lines = append(lines,
			fmt.Sprintf("Opponent Scout Dossier: %s  (%d total games)", sc.PlayerName, sc.TotalGames),
			fmt.Sprintf("As White: %dW / %dD / %dL (%.1f%% win)   ·   As Black: %dW / %dD / %dL (%.1f%% win)",
				sc.WhiteRecord.Wins, sc.WhiteRecord.Draws, sc.WhiteRecord.Losses, sc.WhiteRecord.WinRate,
				sc.BlackRecord.Wins, sc.BlackRecord.Draws, sc.BlackRecord.Losses, sc.BlackRecord.WinRate),
			"",
			"Most Frequent White Openings:",
		)
		for _, ln := range sc.TopOpeningsAsWhite {
			lines = append(lines, fmt.Sprintf("  • %-5s  %3d gms  +%d =%d -%d  (%.1f%%)", ln.ECO, ln.Games, ln.Wins, ln.Draws, ln.Losses, ln.WinRate))
		}
		lines = append(lines, "", "Most Frequent Black Defenses:")
		for _, ln := range sc.TopOpeningsAsBlack {
			lines = append(lines, fmt.Sprintf("  • %-5s  %3d gms  +%d =%d -%d  (%.1f%%)", ln.ECO, ln.Games, ln.Wins, ln.Draws, ln.Losses, ln.WinRate))
		}
		if len(sc.WeakestLines) > 0 {
			lines = append(lines, "", "Primary Prep Targets (Lowest Win Rate Lines):")
			for _, ln := range sc.WeakestLines {
				lines = append(lines, fmt.Sprintf("  • %-5s  %3d gms  +%d =%d -%d  (%.1f%%)", ln.ECO, ln.Games, ln.Wins, ln.Draws, ln.Losses, ln.WinRate))
			}
		}
		if len(sc.RepertoireRecommendations) > 0 {
			lines = append(lines, "", "Recommendations:")
			for _, rec := range sc.RepertoireRecommendations {
				lines = append(lines, "  - "+rec)
			}
		}
	} else {
		lines = []string{
			"Press [F] to run a FairPlay Inspector forensic dossier on White/Black (IPR, ACPL, cadence CV, clutch toggles).",
			"Press [S] to compile an Opponent Preparation Scout dossier (ECO frequencies, weakest targets, prep notes).",
			"Or target any player in the open database via :fairplay <PlayerName> or :scout <PlayerName>.",
		}
	}

	avail := maxInt(1, r.H-2)
	if scroll > maxInt(0, len(lines)-avail) {
		scroll = maxInt(0, len(lines)-avail)
	}
	for i := 0; i < avail && scroll+i < len(lines); i++ {
		scr.WriteText(r.X, r.Y+2+i, r.W, lines[scroll+i], Style{FG: ColTextPrimary, BG: ColBgPanel})
	}
}

func (s *Session) renderBottomBars(scr *Screen) {
	w := scr.Width
	promptY := scr.Height - 2
	legendY := scr.Height - 1

	// Row H-2: Interactive Prompt OR Status Line
	switch s.Mode {
	case ModeMoveInput:
		prompt := fmt.Sprintf(" Play Move (SAN/UCI) > %s█", s.InputBuffer)
		scr.WritePadded(0, promptY, w, prompt, Style{FG: ColBgBase, BG: ColAccentEmerald, Bold: true})
	case ModeSearchFilter:
		prompt := fmt.Sprintf(" Filter Games (player: white: eco: minelo: year: pos) > %s█", s.InputBuffer)
		scr.WritePadded(0, promptY, w, prompt, Style{FG: ColBgBase, BG: ColAccentCyan, Bold: true})
	case ModeCommand:
		prompt := fmt.Sprintf(" :%s█", s.InputBuffer)
		scr.WritePadded(0, promptY, w, prompt, Style{FG: ColBgBase, BG: ColAccentAmber, Bold: true})
	default:
		fg := ColTextPrimary
		if s.StatusIsErr {
			fg = ColAccentRose
		}
		scr.WritePadded(0, promptY, w, " "+s.StatusMsg, Style{FG: fg, BG: ColBgStatus})
	}

	// Row H-1: Keybinding Legend
	legend := " ←/→ Ply  [ ] Var  Click/Drag Move  m Move  e Engine  Space Best  1-6 Tabs  t Theme  p Pieces  f Flip  g Kitty/RGB  / Filter  : Cmd  q Quit"
	scr.WritePadded(0, legendY, w, legend, Style{FG: ColTextMuted, BG: ColBgHeader})
}

func (s *Session) renderHelpModal(scr *Screen) {
	mw := minInt(scr.Width-6, 78)
	mh := minInt(scr.Height-4, 24)
	if mw < 30 || mh < 10 {
		return
	}
	mx := (scr.Width - mw) / 2
	my := (scr.Height - mh) / 2
	box := Rect{X: mx, Y: my, W: mw, H: mh}
	scr.DrawBox(box, "NewScid Terminal Workstation — Help & Commands", "ESC / ? to close", true, ColBgHeader)

	helpLines := []string{
		"BOARD & NAVIGATION",
		"  ← / → or h / l      Step backward / forward one ply (Home / End for start / end)",
		"  [ / ]               Switch between move variations at current branch point",
		"  Mouse Click / Drag  Click piece then target square (or drag & release) on Kitty board",
		"  m                   Enter move by keyboard in SAN (Nf3, O-O, exd5) or UCI (e2e4)",
		"  f / t / p / n       Flip board / Cycle 12 Scid themes / Cycle 6 piece sets / Notation",
		"  g                   Toggle live between Kitty Graphics HD PNG and 24-bit Unicode RGB",
		"",
		"ENGINE, DATABASE & FORENSICS",
		"  e   /  + -          Toggle live Stockfish/Scid UCI engine  /  Adjust Multi-PV (1-5)",
		"  Space               Play top UCI engine move (or top Opening Tree book move)",
		"  1..6 or Tab         Switch tabs: Score · Tree · Games · Bases · Imbalance · Forensics",
		"  N / P               Load Next / Previous game from open .si4 or .pgn database",
		"  /                   Filter database games (player:Carlsen eco:B90 minelo:2650 pos)",
		"  F / S               Run FairPlay forensic dossier [F] or Opponent Scout dossier [S]",
		"",
		"COLON COMMANDS (:)",
		"  :open <db.si4|pgn>  :fen <FEN>   :export <out.pgn>   :save   :new   :game <num>",
		"  :engine <bin>       :fairplay <player>   :scout <player>   :theme <id>   :quit",
	}

	for i := 0; i < mh-2 && i < len(helpLines); i++ {
		st := Style{FG: ColTextPrimary, BG: ColBgHeader}
		if !strings.HasPrefix(helpLines[i], " ") && helpLines[i] != "" {
			st = Style{FG: ColAccentCyan, BG: ColBgHeader, Bold: true}
		}
		scr.WriteText(mx+2, my+1+i, mw-4, helpLines[i], st)
	}
}

// HandleEvent processes a single keyboard or SGR mouse InputEvent.
func (s *Session) HandleEvent(ev InputEvent) {
	if ev.Kind == KeyCtrlC || ev.Kind == KeyCtrlQ {
		s.ShouldQuit = true
		return
	}
	if ev.Kind == KeyCtrlL {
		s.InvalidateKittyCache()
		return
	}

	if s.Mode == ModeHelpModal {
		if ev.Kind == KeyEsc || ev.Kind == KeyEnter || (ev.Kind == KeyRune && (ev.Rune == '?' || ev.Rune == 'q')) {
			s.Mode = ModeNormal
			s.InvalidateKittyCache()
		}
		return
	}

	if s.Mode == ModeMoveInput || s.Mode == ModeSearchFilter || s.Mode == ModeCommand {
		s.handlePromptEvent(ev)
		return
	}

	if ev.Kind == KeyMouse {
		s.handleMouseEvent(ev)
		return
	}

	switch ev.Kind {
	case KeyLeft:
		s.StepPly(-1)
	case KeyRight:
		s.StepPly(1)
	case KeyHome:
		s.JumpToPly(0)
	case KeyEnd:
		s.JumpToPly(len(s.Path))
	case KeyBackspace:
		s.StepPly(-1)
	case KeyTab:
		s.ActiveTab = (s.ActiveTab + 1) % tabCount
	case KeyBackTab:
		s.ActiveTab = (s.ActiveTab + tabCount - 1) % tabCount
	case KeyF1:
		s.ActiveTab = TabScore
	case KeyF2:
		s.ActiveTab = TabTree
	case KeyF3:
		s.ActiveTab = TabGames
	case KeyF4:
		s.ActiveTab = TabBases
	case KeyF5:
		s.ActiveTab = TabImbalance
	case KeyF6:
		s.ActiveTab = TabForensics
	case KeyUp:
		s.scrollActiveTab(-1)
	case KeyDown:
		s.scrollActiveTab(1)
	case KeyPgUp:
		s.scrollActiveTab(-10)
	case KeyPgDn:
		s.scrollActiveTab(10)
	case KeyEnter:
		s.activateSelectedInTab()
	case KeySpace:
		s.playBestSuggestedMove()
	case KeyEsc:
		s.SelectedSq = chess.NoSquare
		s.DragFromSq = chess.NoSquare
	case KeyRune:
		s.handleNormalRune(ev.Rune)
	}
}

func (s *Session) handleNormalRune(r rune) {
	// In Opening Tree tab, pressing 'a'..'i' plays that book move directly
	if s.ActiveTab == TabTree && r >= 'a' && r <= 'i' {
		idx := int(r - 'a')
		s.mu.Lock()
		var san string
		if s.TreeResult != nil && idx < len(s.TreeResult.Moves) {
			san = s.TreeResult.Moves[idx].Move
		}
		s.mu.Unlock()
		if san != "" {
			if err := s.PlayMoveInput(san); err != nil {
				s.SetStatus(err.Error(), true)
			} else {
				s.SetStatus("Played book move "+san, false)
			}
			return
		}
	}

	switch r {
	case 'q':
		s.ShouldQuit = true
	case '?':
		s.Mode = ModeHelpModal
	case '1', '2', '3', '4', '5', '6':
		s.ActiveTab = TabID(r - '1')
	case 'h':
		s.StepPly(-1)
	case 'l':
		s.StepPly(1)
	case 'j':
		s.scrollActiveTab(1)
	case 'k':
		s.scrollActiveTab(-1)
	case '0', '^':
		s.JumpToPly(0)
	case '$', 'G':
		s.JumpToPly(len(s.Path))
	case '[':
		if !s.CycleVariation(-1) {
			s.SetStatus("No variation at current move", false)
		}
	case ']', 'v':
		if !s.CycleVariation(1) {
			s.SetStatus("No variation at current move", false)
		}
	case 'f':
		s.Flipped = !s.Flipped
		s.SetStatus(fmt.Sprintf("Board flipped (Black at bottom: %v)", s.Flipped), false)
	case 't':
		s.CycleTheme()
	case 'p':
		s.CyclePieceSet()
	case 'n':
		s.CycleNotation()
	case 'g':
		s.UseKitty = !s.UseKitty
		s.InvalidateKittyCache()
		if s.UseKitty {
			s.SetStatus("Switched to Kitty Graphics Protocol HD board", false)
		} else {
			s.SetStatus("Switched to 24-bit TrueColor Unicode board", false)
		}
	case 'a':
		s.ShowArrows = !s.ShowArrows
		s.SetStatus(fmt.Sprintf("Board arrows: %v", s.ShowArrows), false)
	case 'e':
		s.ToggleEngine()
	case '=', '+':
		s.AdjustMultiPV(1)
	case '-':
		s.AdjustMultiPV(-1)
	case 'm':
		s.Mode = ModeMoveInput
		s.InputBuffer = ""
	case '/':
		s.Mode = ModeSearchFilter
		s.InputBuffer = s.FilterQuery
	case ':':
		s.Mode = ModeCommand
		s.InputBuffer = ""
	case 'N':
		s.stepDatabaseGame(1)
	case 'P':
		s.stepDatabaseGame(-1)
	case 'x':
		s.ResetNewGame("")
		s.onPositionChanged()
		s.SetStatus("Started fresh analysis board", false)
	case 's':
		if err := s.SaveCurrentGameToDB(); err != nil {
			s.SetStatus(err.Error(), true)
		} else {
			s.SetStatus(fmt.Sprintf("Saved game #%d to %s", s.ActiveGameIndex, s.ActiveDBName), false)
		}
	case 'F':
		s.RunFairPlayAsync("")
	case 'S':
		s.RunScoutAsync("")
	}
}

func (s *Session) handlePromptEvent(ev InputEvent) {
	switch ev.Kind {
	case KeyEsc:
		s.Mode = ModeNormal
		s.InputBuffer = ""
	case KeyCtrlU:
		s.InputBuffer = ""
	case KeyCtrlW:
		trimmed := strings.TrimRight(s.InputBuffer, " ")
		if idx := strings.LastIndexByte(trimmed, ' '); idx != -1 {
			s.InputBuffer = trimmed[:idx+1]
		} else {
			s.InputBuffer = ""
		}
	case KeyBackspace:
		runes := []rune(s.InputBuffer)
		if len(runes) > 0 {
			s.InputBuffer = string(runes[:len(runes)-1])
		}
	case KeyEnter:
		val := strings.TrimSpace(s.InputBuffer)
		mode := s.Mode
		s.Mode = ModeNormal
		s.InputBuffer = ""
		switch mode {
		case ModeMoveInput:
			if val != "" {
				if err := s.PlayMoveInput(val); err != nil {
					s.SetStatus(err.Error(), true)
				} else {
					s.SetStatus("Played "+val, false)
				}
			}
		case ModeSearchFilter:
			s.ActiveTab = TabGames
			s.ApplyFilterQuery(val)
			s.SetStatus(fmt.Sprintf("Filter matched %d games", s.TotalMatched), false)
		case ModeCommand:
			s.ExecuteColonCommand(val)
		}
	case KeySpace:
		s.InputBuffer += " "
	case KeyRune:
		if !ev.Ctrl && !ev.Alt {
			s.InputBuffer += string(ev.Rune)
		}
	}
}

// ExecuteColonCommand executes a `:command` entered in the command bar.
func (s *Session) ExecuteColonCommand(raw string) {
	cmdLine := strings.TrimSpace(raw)
	if cmdLine == "" {
		return
	}
	parts := strings.Fields(cmdLine)
	head := strings.ToLower(parts[0])
	rest := ""
	if len(cmdLine) > len(parts[0]) {
		rest = strings.TrimSpace(cmdLine[len(parts[0]):])
	}

	switch head {
	case "q", "quit", "exit":
		s.ShouldQuit = true
	case "help", "h":
		s.Mode = ModeHelpModal
	case "o", "open", "db", "pgn":
		if rest == "" {
			s.ActiveTab = TabBases
			s.RefreshDiscoveredBases()
			return
		}
		if err := s.OpenPath(rest); err != nil {
			s.SetStatus(fmt.Sprintf("Open failed: %v", err), true)
		}
	case "fen":
		if rest == "" {
			s.SetStatus("FEN: "+s.Board.FEN(), false)
			return
		}
		if err := s.LoadFEN(rest); err != nil {
			s.SetStatus(fmt.Sprintf("Invalid FEN: %v", err), true)
		} else {
			s.SetStatus("Loaded custom FEN position", false)
		}
	case "new", "reset":
		s.ResetNewGame("")
		s.onPositionChanged()
		s.SetStatus("Reset to initial chess position", false)
	case "game", "g":
		n, err := strconv.Atoi(rest)
		if err != nil || n < 1 {
			s.SetStatus("Usage: :game <number>", true)
			return
		}
		if err := s.LoadGameNumber(n); err != nil {
			s.SetStatus(err.Error(), true)
		} else {
			s.SetStatus(fmt.Sprintf("Loaded game #%d (%s vs %s)", n, s.Game.Tag("White"), s.Game.Tag("Black")), false)
		}
	case "filter":
		s.ActiveTab = TabGames
		s.ApplyFilterQuery(rest)
	case "save", "w":
		if rest != "" {
			if err := s.ExportCurrentPGN(rest); err != nil {
				s.SetStatus(fmt.Sprintf("Export error: %v", err), true)
			} else {
				s.SetStatus("Exported PGN to "+rest, false)
			}
			return
		}
		if err := s.SaveCurrentGameToDB(); err != nil {
			s.SetStatus(err.Error(), true)
		} else {
			s.SetStatus(fmt.Sprintf("Saved game #%d to %s", s.ActiveGameIndex, s.ActiveDBName), false)
		}
	case "export":
		if err := s.ExportCurrentPGN(rest); err != nil {
			s.SetStatus(fmt.Sprintf("Export error: %v", err), true)
		} else {
			out := rest
			if out == "" {
				out = "newscid-export.pgn"
			}
			s.SetStatus("Exported current game PGN to "+out, false)
		}
	case "engine":
		if rest != "" {
			s.StopEngine()
			s.EnginePath = rest
		}
		if err := s.StartEngine(); err != nil {
			s.SetStatus(err.Error(), true)
		} else {
			s.SetStatus("Started UCI engine: "+s.EngineName, false)
		}
	case "theme":
		if rest == "" {
			s.CycleTheme()
		} else {
			th := LookupBoardTheme(rest)
			s.ThemeID = th.ID
			s.SetStatus(fmt.Sprintf("Board theme: %s", th.Name), false)
		}
	case "pieces", "set":
		if rest == "" {
			s.CyclePieceSet()
		} else {
			s.PieceSetID = strings.ToLower(rest)
			s.SetStatus("Piece set: "+s.PieceSetID, false)
		}
	case "notation":
		if chess.ValidNotationStyle(rest) {
			s.Notation = chess.NotationStyle(strings.ToLower(rest))
			s.SetStatus("Notation: "+strings.ToUpper(rest), false)
		} else {
			s.CycleNotation()
		}
	case "kitty":
		s.UseKitty = true
		s.InvalidateKittyCache()
		s.SetStatus("Enabled Kitty Graphics Protocol HD board", false)
	case "unicode", "ascii":
		s.UseKitty = false
		s.InvalidateKittyCache()
		s.SetStatus("Enabled 24-bit TrueColor Unicode board", false)
	case "fairplay", "fp":
		s.RunFairPlayAsync(rest)
	case "scout":
		s.RunScoutAsync(rest)
	case "move", "play":
		if err := s.PlayMoveInput(rest); err != nil {
			s.SetStatus(err.Error(), true)
		}
	default:
		// Convenience: allow typing a move directly after ':' like `:e4` or `:Nf3`
		if err := s.PlayMoveInput(cmdLine); err == nil {
			s.SetStatus("Played "+cmdLine, false)
			return
		}
		s.SetStatus(fmt.Sprintf("Unknown command :%s (press ? for help)", head), true)
	}
}

func (s *Session) handleMouseEvent(ev InputEvent) {
	col0 := ev.MouseCol - 1
	row0 := ev.MouseRow - 1

	// 1. Mouse wheel navigation
	if ev.MouseAction == MouseWheelUp || ev.MouseAction == MouseWheelDown {
		dir := 1
		if ev.MouseAction == MouseWheelUp {
			dir = -1
		}
		if s.hitBoardRect.Contains(col0, row0) || s.ActiveTab == TabScore {
			s.StepPly(dir)
		} else {
			s.scrollActiveTab(dir * 2)
		}
		return
	}

	// 2. Board interactions (click-to-move & drag-to-move directly on Kitty / Unicode board!)
	if s.hitBoardRect.Contains(col0, row0) {
		sq := s.BoardSquareAtCell(col0, row0)
		if sq >= 64 {
			return
		}
		if ev.MouseBtn == 2 && ev.MouseAction == MousePress {
			s.SelectedSq = chess.NoSquare
			s.DragFromSq = chess.NoSquare
			return
		}
		if ev.MouseBtn == 0 {
			switch ev.MouseAction {
			case MousePress:
				if s.SelectedSq < 64 && s.SelectedSq != sq {
					if err := s.ApplyLegalMove(chess.Move{From: s.SelectedSq, To: sq}); err == nil {
						return
					}
				}
				p := s.Board.PieceAt(sq)
				if p != chess.Empty && p.Color() == s.Board.Turn {
					s.SelectedSq = sq
					s.DragFromSq = sq
				} else {
					s.SelectedSq = chess.NoSquare
					s.DragFromSq = chess.NoSquare
				}
			case MouseRelease:
				if s.DragFromSq < 64 && s.DragFromSq != sq {
					_ = s.ApplyLegalMove(chess.Move{From: s.DragFromSq, To: sq})
					s.DragFromSq = chess.NoSquare
				}
			}
		}
		return
	}

	if ev.MouseBtn != 0 || ev.MouseAction != MousePress {
		return
	}

	// 3. Tab header clicks
	for i := TabID(0); i < tabCount; i++ {
		if s.hitTabRects[i].Contains(col0, row0) {
			s.ActiveTab = i
			return
		}
	}

	// 4. Scorebook ply clicks
	if s.ActiveTab == TabScore {
		for _, sp := range s.hitScorePlies {
			if row0 == sp.Row && col0 >= sp.X0 && col0 < sp.X1 {
				s.JumpToPly(sp.Ply)
				return
			}
		}
	}

	// 5. List row clicks in Tree / Games / Bases
	if s.hitListRowCount > 0 && row0 >= s.hitListStartRow && row0 < s.hitListStartRow+s.hitListRowCount {
		rel := row0 - s.hitListStartRow
		switch s.ActiveTab {
		case TabTree:
			s.mu.Lock()
			if s.TreeResult != nil && rel < len(s.TreeResult.Moves) {
				s.TreeCursor = rel
			}
			s.mu.Unlock()
			s.activateSelectedInTab()
		case TabGames:
			avail := s.hitListRowCount
			start := 0
			if s.GameListCursor >= avail {
				start = s.GameListCursor - avail + 1
			}
			idx := start + rel
			if idx >= 0 && idx < len(s.GameSummaries) {
				if s.GameListCursor == idx {
					s.activateSelectedInTab()
				} else {
					s.GameListCursor = idx
				}
			}
		case TabBases:
			avail := s.hitListRowCount
			start := 0
			if s.BasesCursor >= avail {
				start = s.BasesCursor - avail + 1
			}
			idx := start + rel
			if idx >= 0 && idx < len(s.DiscoveredBases) {
				if s.BasesCursor == idx {
					s.activateSelectedInTab()
				} else {
					s.BasesCursor = idx
				}
			}
		}
	}
}

// BoardSquareAtCell maps a 0-based terminal character coordinate (col0, row0) inside hitBoardRect to a chess.Square.
func (s *Session) BoardSquareAtCell(col0, row0 int) chess.Square {
	r := s.hitBoardRect
	if !r.Contains(col0, row0) || r.W <= 0 || r.H <= 0 {
		return chess.NoSquare
	}
	bCol := ((col0 - r.X) * 8) / r.W
	bRow := ((row0 - r.Y) * 8) / r.H
	return DisplayCoordsToSquare(bCol, bRow, s.Flipped)
}

func (s *Session) scrollActiveTab(delta int) {
	switch s.ActiveTab {
	case TabScore:
		s.StepPly(delta * 2)
	case TabTree:
		s.mu.Lock()
		if s.TreeResult != nil && len(s.TreeResult.Moves) > 0 {
			s.TreeCursor = clampIdx(s.TreeCursor+delta, len(s.TreeResult.Moves))
		}
		s.mu.Unlock()
	case TabGames:
		if len(s.GameSummaries) > 0 {
			s.GameListCursor = clampIdx(s.GameListCursor+delta, len(s.GameSummaries))
		}
	case TabBases:
		if len(s.DiscoveredBases) > 0 {
			s.BasesCursor = clampIdx(s.BasesCursor+delta, len(s.DiscoveredBases))
		}
	case TabForensics:
		s.ForensicsScroll = maxInt(0, s.ForensicsScroll+delta)
	}
}

func (s *Session) activateSelectedInTab() {
	switch s.ActiveTab {
	case TabScore:
		s.Mode = ModeMoveInput
		s.InputBuffer = ""
	case TabTree:
		s.mu.Lock()
		var san string
		if s.TreeResult != nil && s.TreeCursor >= 0 && s.TreeCursor < len(s.TreeResult.Moves) {
			san = s.TreeResult.Moves[s.TreeCursor].Move
		}
		s.mu.Unlock()
		if san != "" {
			if err := s.PlayMoveInput(san); err != nil {
				s.SetStatus(err.Error(), true)
			} else {
				s.SetStatus("Played book move "+san, false)
			}
		}
	case TabGames:
		if s.GameListCursor >= 0 && s.GameListCursor < len(s.GameSummaries) {
			gs := s.GameSummaries[s.GameListCursor]
			if err := s.LoadGameNumber(gs.ID); err != nil {
				s.SetStatus(err.Error(), true)
			} else {
				s.ActiveTab = TabScore
				s.SetStatus(fmt.Sprintf("Loaded game #%d: %s vs %s (%s)", gs.ID, gs.White, gs.Black, gs.Date), false)
			}
		}
	case TabBases:
		if s.BasesCursor >= 0 && s.BasesCursor < len(s.DiscoveredBases) {
			db := s.DiscoveredBases[s.BasesCursor]
			if err := s.OpenPath(db.BasePath); err != nil {
				s.SetStatus(err.Error(), true)
			} else {
				s.ActiveTab = TabGames
			}
		}
	}
}

func (s *Session) playBestSuggestedMove() {
	s.mu.Lock()
	engOn := s.EngineEnabled
	eval := s.LastEval
	evalFEN := s.EvalPositionFEN
	treeRes := s.TreeResult
	s.mu.Unlock()

	if engOn && evalFEN == s.Board.FEN() {
		uci := eval.BestMove
		if uci == "" && len(eval.Lines) > 0 {
			uci = eval.Lines[0].MoveUCI
		}
		if uci == "" && len(eval.PV) > 0 {
			uci = eval.PV[0]
		}
		if uci != "" {
			if err := s.PlayMoveInput(uci); err == nil {
				s.SetStatus("Played engine move "+uci, false)
				return
			}
		}
	}

	if treeRes != nil && len(treeRes.Moves) > 0 {
		mv := treeRes.Moves[0].Move
		if err := s.PlayMoveInput(mv); err == nil {
			s.SetStatus("Played top book move "+mv, false)
			return
		}
	}

	s.SetStatus("Start engine [e] or open a database for suggested move", false)
}

func (s *Session) stepDatabaseGame(delta int) {
	next := s.ActiveGameIndex + delta
	if next < 1 {
		next = 1
	}
	if err := s.LoadGameNumber(next); err != nil {
		s.SetStatus(err.Error(), true)
		return
	}
	s.SetStatus(fmt.Sprintf("Loaded game #%d: %s vs %s", next, s.Game.Tag("White"), s.Game.Tag("Black")), false)
}

func formatEvalCompact(cp, mateIn int) string {
	if mateIn != 0 {
		if mateIn > 0 {
			return fmt.Sprintf("+M%d", mateIn)
		}
		return fmt.Sprintf("-M%d", -mateIn)
	}
	pawns := float64(cp) / 100.0
	if pawns >= 0 {
		return fmt.Sprintf("+%.2f", pawns)
	}
	return fmt.Sprintf("%.2f", pawns)
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func boolAdv(w, b bool) string {
	if w && !b {
		return "White"
	}
	if b && !w {
		return "Black"
	}
	return "Equal"
}

func deltaLabel(diff int) string {
	if diff > 0 {
		return fmt.Sprintf("White (+%d)", diff)
	}
	if diff < 0 {
		return fmt.Sprintf("Black (+%d)", -diff)
	}
	return "Equal"
}

func clampIdx(v, length int) int {
	if length <= 0 {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v >= length {
		return length - 1
	}
	return v
}
