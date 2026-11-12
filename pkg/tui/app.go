package tui

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/fileformat"
	"newscid/pkg/version"
)

// RunCLI parses command-line flags and launches the NewScid Kitty Protocol / TrueColor TUI workstation
// (or renders a single position inline when --render-once is passed).
func RunCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	for _, a := range args {
		if a == "-v" || a == "--version" || a == "version" {
			fmt.Fprintln(stdout, version.String())
			return 0
		}
		if a == "-h" || a == "--help" || a == "help" {
			printUsage(stdout)
			return 0
		}
	}

	fs := flag.NewFlagSet("newscid-tui", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dbFlag := fs.String("db", "", "Open a Scid v4 database (.si4 or base path)")
	pgnFlag := fs.String("pgn", "", "Open a PGN file (.pgn)")
	fenFlag := fs.String("fen", "", "Load initial position from FEN")
	gameFlag := fs.Int("game", 0, "1-based game number to load from database or PGN")
	plyFlag := fs.Int("ply", -1, "Jump to specific half-move ply on startup")
	engineFlag := fs.String("engine", "", "Path to UCI engine binary (e.g. stockfish)")
	autoEngFlag := fs.Bool("auto-engine", false, "Start live UCI engine analysis immediately on launch")
	noEngFlag := fs.Bool("no-engine", false, "Disable automatic UCI engine startup")
	kittyFlag := fs.Bool("kitty", false, "Use high-DPI pixel board (Kitty Graphics Protocol, or Sixel on foot)")
	forceKittyFlag := fs.Bool("force-kitty", false, "Force raw Kitty APC sequences even on non-Kitty terminals")
	sixelFlag := fs.Bool("sixel", false, "Force DEC Sixel high-DPI pixel board rendering (foot, xterm, contour)")
	unicodeFlag := fs.Bool("unicode", false, "Use 24-bit TrueColor Unicode board instead of pixel graphics")
	themeFlag := fs.String("theme", "tournament-green", "Board theme ID (tournament-green, classic-wood, walnut, slate, blue-white, ...)")
	pieceFlag := fs.String("piece-set", "cburnett", "Piece set ID (cburnett, merida, staunty, alpha, leipzig, cardinal)")
	flipFlag := fs.Bool("flip", false, "Flip board orientation (Black at bottom)")
	renderOnceFlag := fs.Bool("render-once", false, "Render the board position once directly to stdout and exit")
	colsFlag := fs.Int("cols", 28, "Board width in terminal columns for --render-once Kitty output")
	rowsFlag := fs.Int("rows", 14, "Board height in terminal rows for --render-once Kitty output")

	// Allow positional database/PGN path before or after flags
	flagArgs, posArgs := splitPositionalArgs(args)
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}
	posArgs = append(posArgs, fs.Args()...)

	dbPath := strings.TrimSpace(*dbFlag)
	pgnPath := strings.TrimSpace(*pgnFlag)
	if dbPath == "" && pgnPath == "" && len(posArgs) > 0 {
		p := posArgs[0]
		if fileformat.IsPGNPath(p) {
			pgnPath = p
		} else {
			dbPath = p
		}
	}

	gMode := DetectBestGraphicsModeFromEnv(os.Getenv)
	explicit := false
	switch {
	case *forceKittyFlag:
		gMode = GraphicsKitty
		explicit = true
	case *sixelFlag:
		gMode = GraphicsSixel
		explicit = true
	case *unicodeFlag:
		gMode = GraphicsUnicode
		explicit = true
	case *kittyFlag:
		if IsFootTerminal(os.Getenv) {
			gMode = GraphicsSixel
		} else {
			gMode = GraphicsKitty
		}
		explicit = true
	}

	autoEngine := (*autoEngFlag || *engineFlag != "") && !*noEngFlag

	sess := NewSession(SessionConfig{
		DBPath:          dbPath,
		PGNPath:         pgnPath,
		InitialFEN:      *fenFlag,
		GameNumber:      *gameFlag,
		EnginePath:      *engineFlag,
		AutoEngine:      autoEngine,
		Graphics:        gMode,
		HasGraphicsMode: true,
		ExplicitMode:    explicit,
		UseKitty:        gMode == GraphicsKitty,
		ThemeID:         *themeFlag,
		PieceSetID:      *pieceFlag,
		Flipped:         *flipFlag,
	})
	defer sess.Close()

	if *plyFlag >= 0 {
		sess.JumpToPly(*plyFlag)
	}

	if *renderOnceFlag {
		if err := renderInlineOnce(sess, stdout, *colsFlag, *rowsFlag); err != nil {
			fmt.Fprintf(stderr, "newscid-tui render error: %v\n", err)
			return 1
		}
		return 0
	}

	return runInteractiveLoop(sess, stdin, stdout, stderr)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "NewScid TUI — Kitty Protocol & TrueColor Terminal Chess Workstation (%s)\n\n", version.Short())
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  newscid-tui [options] [database.si4 | games.pgn]")
	fmt.Fprintln(w, "  newscid tui [options] [database.si4 | games.pgn]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --db <path>         Open a Scid v4 database (.si4 or base path)")
	fmt.Fprintln(w, "  --pgn <path>        Open a PGN file (.pgn)")
	fmt.Fprintln(w, "  --game <N>          Load 1-based game number from database or PGN")
	fmt.Fprintln(w, "  --ply <N>           Jump to half-move ply N on startup")
	fmt.Fprintln(w, "  --fen <FEN>         Start from a custom FEN position")
	fmt.Fprintln(w, "  --kitty             Enable high-DPI pixel board (Kitty Graphics, or Sixel on foot)")
	fmt.Fprintln(w, "  --sixel             Force DEC Sixel high-DPI pixel board (foot, xterm, contour)")
	fmt.Fprintln(w, "  --unicode           Use 24-bit TrueColor Unicode fallback board")
	fmt.Fprintln(w, "  --theme <id>        Board theme (tournament-green, classic-wood, walnut, blue-white, slate, ...)")
	fmt.Fprintln(w, "  --piece-set <id>    Piece set (cburnett, merida, staunty, alpha, leipzig, cardinal)")
	fmt.Fprintln(w, "  --flip              Flip board orientation (Black at bottom)")
	fmt.Fprintln(w, "  --engine <path>     UCI engine executable path (auto-discovers Stockfish / Scid engines)")
	fmt.Fprintln(w, "  --auto-engine       Start live UCI engine analysis immediately")
	fmt.Fprintln(w, "  --render-once       Render position directly to terminal stdout and exit")
	fmt.Fprintln(w, "  -v, --version       Print version metadata")
	fmt.Fprintln(w, "  -h, --help          Show this help text")
}

// splitPositionalArgs allows `newscid-tui games.si4 --kitty --game 5` in any order,
// unlike standard library flag which stops parsing at the first positional argument.
func splitPositionalArgs(args []string) (flags []string, positional []string) {
	takesValue := map[string]bool{
		"-db": true, "--db": true,
		"-pgn": true, "--pgn": true,
		"-fen": true, "--fen": true,
		"-game": true, "--game": true,
		"-ply": true, "--ply": true,
		"-engine": true, "--engine": true,
		"-theme": true, "--theme": true,
		"-piece-set": true, "--piece-set": true,
		"-cols": true, "--cols": true,
		"-rows": true, "--rows": true,
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && takesValue[a] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, a)
		}
	}
	return flags, positional
}

func renderInlineOnce(s *Session, stdout io.Writer, cols, rows int) error {
	if cols < 8 {
		cols = 28
	}
	if rows < 4 {
		rows = 14
	}

	w := s.Game.Tag("White")
	b := s.Game.Tag("Black")
	if w != "?" && w != "White" || b != "?" && b != "Black" {
		fmt.Fprintf(stdout, "%s vs %s (%s) — %s\n", w, b, s.Game.Result, s.Game.Tag("Event"))
	}

	lastFrom, lastTo := s.LastMoveSquares()
	checkSq := chess.NoSquare
	if s.Board.InCheck(s.Board.Turn) {
		checkSq = s.Board.KingSquare(s.Board.Turn)
	}
	opt := BoardRenderOptions{
		ThemeID:     s.ThemeID,
		PieceSetID:  s.PieceSetID,
		SquarePx:    80,
		Flipped:     s.Flipped,
		SelectedSq:  chess.NoSquare,
		LastFrom:    lastFrom,
		LastTo:      lastTo,
		Arrows:      s.ActiveBoardArrows(),
		ShowCoords:  true,
		CheckKingSq: checkSq,
	}

	switch s.Graphics {
	case GraphicsKitty:
		pngBytes, err := RenderBoardPNG(s.Board, opt)
		if err != nil {
			return err
		}
		seq := encodeInlineKittyPNG(cols, rows, pngBytes)
		if _, err := io.WriteString(stdout, seq+"\n"); err != nil {
			return err
		}
	case GraphicsSixel:
		opt.SquarePx = 64
		img, err := RenderBoardImage(s.Board, opt)
		if err != nil {
			return err
		}
		seq := EncodeSixelImage(img)
		if _, err := io.WriteString(stdout, seq+"\n"); err != nil {
			return err
		}
	default:
		scrW := 36
		scrH := 18
		scr := NewScreen(scrW, scrH)
		scr.DrawUnicodeBoard(Rect{X: 0, Y: 0, W: scrW, H: scrH}, s.Board, opt)
		if err := renderUnicodeInline(scr, stdout); err != nil {
			return err
		}
	}

	fmt.Fprintf(stdout, "FEN: %s\n", s.Board.FEN())
	return nil
}

func encodeInlineKittyPNG(cols, rows int, pngBytes []byte) string {
	// Place inline at current cursor and allow cursor to advance below image (do not pass C=1)
	raw := EncodeKittyPNGPlacement(KittyBoardImageID, KittyBoardPlacementID, 1, 1, cols, rows, pngBytes)
	// Strip the leading absolute cursor move "\x1b[1;1H" and C=1 so terminal advances naturally below the image
	raw = strings.TrimPrefix(raw, "\x1b[1;1H")
	raw = strings.Replace(raw, ",C=1,", ",", 1)
	return raw
}

func renderUnicodeInline(scr *Screen, w io.Writer) error {
	var sb strings.Builder
	for y := 0; y < 9; y++ {
		var cur Style
		have := false
		for x := 0; x < 28 && x < scr.Width; x++ {
			c := scr.Cells[y*scr.Width+x]
			if !have || cur != c.Style {
				writeSGR(&sb, c.Style)
				cur = c.Style
				have = true
			}
			sb.WriteRune(c.Ch)
		}
		sb.WriteString("\x1b[0m\n")
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

func runInteractiveLoop(sess *Session, stdin io.Reader, stdout, stderr io.Writer) int {
	inFile, inIsFile := stdin.(*os.File)
	outFile, outIsFile := stdout.(*os.File)

	var restoreRaw func()
	ttyFD := -1
	if inIsFile && IsTerminalFD(int(inFile.Fd())) {
		fd := int(inFile.Fd())
		rfn, err := EnterRawMode(fd)
		if err != nil {
			fmt.Fprintf(stderr, "newscid-tui: failed to enter raw terminal mode: %v\n", err)
			return 1
		}
		restoreRaw = rfn
		ttyFD = fd
	} else if outIsFile && IsTerminalFD(int(outFile.Fd())) {
		ttyFD = int(outFile.Fd())
	}

	defer func() {
		// Delete Kitty placements if used, pop Kitty keyboard mode, disable Sixel cursor mode & mouse tracking, leave alt screen
		cleanupSeq := ""
		if sess.Graphics == GraphicsKitty || sess.kittyPlaced {
			cleanupSeq += EncodeKittyDeleteAll()
		}
		cleanupSeq += "\x1b[?8452l\x1b[<u\x1b[?1002l\x1b[?1006l\x1b[?25h\x1b[?1049l\x1b[0m"
		_, _ = io.WriteString(stdout, cleanupSeq)
		if restoreRaw != nil {
			restoreRaw()
		}
	}()

	geom := TermGeometry{Cols: 120, Rows: 36}
	if ttyFD >= 0 {
		geom = GetTermGeometry(ttyFD)
	}

	// Enter alternate screen, hide cursor, enable SGR mouse, Sixel private cursor mode (?8452h), and Kitty CSI-u disambiguation
	initSeq := "\x1b[?1049h\x1b[2J\x1b[H\x1b[?25l\x1b[?1002h\x1b[?1006h\x1b[?8452h\x1b[>1u"
	if ttyFD >= 0 {
		if geom.XPixel == 0 || geom.YPixel == 0 {
			initSeq += "\x1b[16t"
		}
		if !sess.ExplicitMode && sess.Graphics == GraphicsUnicode {
			if !IsFootTerminal(os.Getenv) {
				initSeq += "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
			}
			initSeq += "\x1b[c"
		}
	}
	if _, err := io.WriteString(stdout, initSeq); err != nil {
		return 1
	}

	scr := NewScreen(geom.Cols, geom.Rows)

	resizeCh, stopResize := SubscribeResize()
	defer stopResize()

	inputCh := make(chan InputEvent, 64)
	eofCh := make(chan struct{})

	go func() {
		defer close(eofCh)
		var pending []byte
		buf := make([]byte, 1024)
		for {
			n, err := stdin.Read(buf)
			if n > 0 {
				pending = append(pending, buf[:n]...)
				evs, consumed := ParseInputEvents(pending)
				if consumed > 0 {
					pending = append(pending[:0], pending[consumed:]...)
				}
				for _, ev := range evs {
					inputCh <- ev
				}
			}
			if err != nil {
				return
			}
		}
	}()

	if err := sess.Render(scr, geom, stdout, true); err != nil {
		return 1
	}

	// Coalesce high-frequency UCI engine updates to ~60Hz max
	var redrawTimer *time.Timer
	var redrawTick <-chan time.Time

	for !sess.ShouldQuit {
		select {
		case ev := <-inputCh:
			fullRedraw := false
			applyEv := func(e InputEvent) {
				if e.Kind == KeyTermCellPixels && e.MouseCol > 0 && e.MouseRow > 0 {
					nx := geom.Cols * e.MouseCol
					ny := geom.Rows * e.MouseRow
					if nx != geom.XPixel || ny != geom.YPixel {
						geom.XPixel = nx
						geom.YPixel = ny
						sess.InvalidateKittyCache()
						fullRedraw = true
					}
					return
				}
				sess.HandleEvent(e)
			}
			applyEv(ev)
			// Drain any immediately queued input events before rendering
		drainInput:
			for !sess.ShouldQuit {
				select {
				case nextEv := <-inputCh:
					applyEv(nextEv)
				default:
					break drainInput
				}
			}
			if sess.ShouldQuit {
				break
			}
			_ = sess.Render(scr, geom, stdout, fullRedraw)

		case <-resizeCh:
			if ttyFD >= 0 {
				geom = GetTermGeometry(ttyFD)
			}
			scr.Invalidate()
			sess.InvalidateKittyCache()
			_, _ = io.WriteString(stdout, "\x1b[2J")
			_ = sess.Render(scr, geom, stdout, true)

		case <-sess.RedrawCh:
			if redrawTimer == nil {
				redrawTimer = time.NewTimer(16 * time.Millisecond)
				redrawTick = redrawTimer.C
			}

		case <-redrawTick:
			redrawTimer = nil
			redrawTick = nil
			_ = sess.Render(scr, geom, stdout, false)

		case <-eofCh:
			// Drain remaining queued events (great for scripted stdin testing!) then exit
			for {
				select {
				case ev := <-inputCh:
					sess.HandleEvent(ev)
				default:
					sess.ShouldQuit = true
				}
				if sess.ShouldQuit {
					break
				}
			}
		}
	}

	return 0
}
