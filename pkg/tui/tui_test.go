package tui

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestRenderBoardPNGAllPieceSetsAndThemes(t *testing.T) {
	b, err := chess.ParseFEN("r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4")
	require.NoError(t, err)

	for _, pset := range AvailableTUIPieceSets {
		opt := BoardRenderOptions{
			ThemeID:      "tournament-green",
			PieceSetID:   pset,
			SquarePx:     64,
			Flipped:      pset == "merida",
			SelectedSq:   chess.F3,
			LastFrom:     chess.G8,
			LastTo:       chess.F6,
			LegalTargets: []chess.Square{chess.G5, chess.E5},
			CaptureMask:  uint64(1) << uint(chess.E5),
			Arrows: []BoardArrow{
				{From: chess.D2, To: chess.D3, Kind: ArrowEnginePV1},
				{From: chess.B1, To: chess.C3, Kind: ArrowEnginePV2},
				{From: chess.C2, To: chess.C3, Kind: ArrowOpeningBook},
			},
			ShowCoords: true,
		}

		pngBytes, err := RenderBoardPNG(b, opt)
		require.NoError(t, err, "piece set %s", pset)
		require.NotEmpty(t, pngBytes)

		decoded, err := png.Decode(bytes.NewReader(pngBytes))
		require.NoError(t, err)
		assert.Equal(t, 512, decoded.Bounds().Dx())
		assert.Equal(t, 512, decoded.Bounds().Dy())
	}

	// Verify check glow renders without error on an in-check position
	checkBoard, err := chess.ParseFEN("4k3/8/8/8/8/8/4Q3/4K3 b - - 0 1")
	require.NoError(t, err)
	assert.True(t, checkBoard.InCheck(chess.Black))
	checkPNG, err := RenderBoardPNG(checkBoard, BoardRenderOptions{
		ThemeID:    "walnut",
		PieceSetID: "cburnett",
		SquarePx:   64,
		ShowCoords: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, checkPNG)
}

func TestKittyDetectionAndChunkEncoding(t *testing.T) {
	assert.True(t, DetectKittySupportFromEnv(func(k string) string {
		if k == "KITTY_WINDOW_ID" {
			return "1"
		}
		return ""
	}))
	assert.True(t, DetectKittySupportFromEnv(func(k string) string {
		if k == "TERM" {
			return "xterm-ghostty"
		}
		return ""
	}))
	assert.False(t, DetectKittySupportFromEnv(func(k string) string {
		if k == "KITTY_WINDOW_ID" {
			return "1"
		}
		if k == "NEWSCID_KITTY" {
			return "0"
		}
		return ""
	}))
	assert.False(t, DetectKittySupportFromEnv(func(string) string { return "" }))

	// Create a synthetic 8KB payload so multiple 4096-byte Base64 chunks are emitted
	fakePNG := bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47, 0xaa, 0x55}, 1400)
	seq := EncodeKittyPNGPlacement(42, 1, 4, 6, 32, 16, fakePNG)

	assert.True(t, strings.HasPrefix(seq, "\x1b[4;6H\x1b_Ga=T,f=100,i=42,p=1,c=32,r=16,C=1,q=2,m=1;"))
	assert.Contains(t, seq, "\x1b_Gm=1;")
	assert.Contains(t, seq, "\x1b_Gm=0;")
	assert.True(t, strings.HasSuffix(seq, "\x1b\\"))
	assert.Equal(t, "\x1b_Ga=d,d=I,i=42,q=2\x1b\\", EncodeKittyDeleteImage(42))
}

func TestParseInputEventsMouseCSIuAndANSI(t *testing.T) {
	raw := []byte(
		"\x1b_Gi=42;OK\x1b\\" + // stray Kitty response ignored cleanly
			"\x1b[<0;12;8M" + // SGR left press at (12,8)
			"\x1b[<0;12;6m" + // SGR left release at (12,6)
			"\x1b[<64;10;10M" + // SGR wheel up
			"\x1b[27u" + // Kitty CSI-u Escape
			"\x1b[9;2u" + // Kitty CSI-u Shift+Tab
			"\x1b[99;5u" + // Kitty CSI-u Ctrl+C
			"\x1b[A" + // Up arrow
			"e4\r",
	)

	evs, consumed := ParseInputEvents(raw)
	assert.Equal(t, len(raw), consumed)
	require.Len(t, evs, 10)

	assert.Equal(t, KeyMouse, evs[0].Kind)
	assert.Equal(t, MousePress, evs[0].MouseAction)
	assert.Equal(t, 12, evs[0].MouseCol)
	assert.Equal(t, 8, evs[0].MouseRow)

	assert.Equal(t, KeyMouse, evs[1].Kind)
	assert.Equal(t, MouseRelease, evs[1].MouseAction)
	assert.Equal(t, 12, evs[1].MouseCol)
	assert.Equal(t, 6, evs[1].MouseRow)

	assert.Equal(t, KeyMouse, evs[2].Kind)
	assert.Equal(t, MouseWheelUp, evs[2].MouseAction)

	assert.Equal(t, KeyEsc, evs[3].Kind)
	assert.Equal(t, KeyBackTab, evs[4].Kind)
	assert.Equal(t, KeyCtrlC, evs[5].Kind)
	assert.Equal(t, KeyUp, evs[6].Kind)
	assert.Equal(t, rune('e'), evs[7].Rune)
	assert.Equal(t, rune('4'), evs[8].Rune)
	assert.Equal(t, KeyEnter, evs[9].Kind)
}

func TestSessionDatabaseMouseMovesTabsAndKittyRender(t *testing.T) {
	tmpDir := t.TempDir()
	dbBase := filepath.Join(tmpDir, "club")

	db := scid.NewDatabase()
	require.NoError(t, db.Create(dbBase, "Test Club DB"))

	g1, err := chess.ParsePGN(`[Event "Blitz Match"]
[Site "Lichess"]
[Date "2026.05.10"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[WhiteElo "2882"]
[BlackElo "2875"]
[Result "1-0"]

1. e4 e5 (1... c5 2. Nf3 d6) 2. Nf3 Nc6 3. Bb5 a6 1-0`)
	require.NoError(t, err)
	_, err = db.SaveGame(g1)
	require.NoError(t, err)

	g2, err := chess.ParsePGN(`[Event "Rapid Open"]
[Site "Oslo"]
[Date "2026.05.11"]
[White "Nakamura, Hikaru"]
[Black "Carlsen, Magnus"]
[WhiteElo "2875"]
[BlackElo "2882"]
[Result "1/2-1/2"]

1. d4 Nf6 2. c4 g6 1/2-1/2`)
	require.NoError(t, err)
	_, err = db.SaveGame(g2)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	sess := NewSession(SessionConfig{
		DBPath:     dbBase + ".si4",
		UseKitty:   true,
		ThemeID:    "tournament-green",
		PieceSetID: "cburnett",
	})
	defer sess.Close()

	assert.Equal(t, 2, sess.TotalMatched)
	assert.Equal(t, "Carlsen, Magnus", sess.Game.Tag("White"))

	// Wait briefly for async Opening Tree worker
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		sess.mu.Lock()
		busy := sess.TreeBusy
		sess.mu.Unlock()
		if !busy {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	scr := NewScreen(128, 38)
	geom := TermGeometry{Cols: 128, Rows: 38, XPixel: 1280, YPixel: 760}
	var out bytes.Buffer

	require.NoError(t, sess.Render(scr, geom, &out, true))
	rendered := out.String()
	// Must contain DEC 2026 sync frame and Kitty PNG placement
	assert.Contains(t, rendered, "\x1b[?2026h")
	assert.Contains(t, rendered, "\x1b_Ga=T,f=100,i=42,p=1")
	assert.Contains(t, rendered, "\x1b[?2026l")

	// Second render with identical board state should NOT re-transmit the Kitty PNG
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	assert.NotContains(t, out.String(), "\x1b_Ga=T,f=100")

	// Play 1. e4 by clicking e2 then e4 directly on the Kitty graphical board!
	r := sess.hitBoardRect
	require.Greater(t, r.W, 8)
	e2Col := r.X + (4*r.W)/8 + 1 // 0-based col for file 'e' (4)
	e2Row := r.Y + (6*r.H)/8     // 0-based row for rank '2' (7-1 = 6)
	e4Col := e2Col
	e4Row := r.Y + (4*r.H)/8 // 0-based row for rank '4' (7-3 = 4)

	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: e2Col + 1, MouseRow: e2Row + 1, MouseAction: MousePress})
	assert.Equal(t, chess.E2, sess.SelectedSq)
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: e4Col + 1, MouseRow: e4Row + 1, MouseAction: MousePress})
	assert.Equal(t, 1, sess.CurPly)
	assert.Equal(t, chess.WP, sess.Board.PieceAt(chess.E4))

	// Cycle variation at ply 1 (1... e5 vs 1... c5)
	assert.True(t, sess.CycleVariation(1))
	assert.Equal(t, chess.BP, sess.Board.PieceAt(chess.C5))

	// Render every tab cleanly
	for tab := TabScore; tab < tabCount; tab++ {
		sess.ActiveTab = tab
		out.Reset()
		require.NoError(t, sess.Render(scr, geom, &out, false))
	}

	// Toggle 'g' to switch to 24-bit TrueColor Unicode fallback — must emit Kitty delete sequence!
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: 'g'})
	assert.False(t, sess.UseKitty)
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	assert.Contains(t, out.String(), "\x1b_Ga=d,d=I,i=42,q=2\x1b\\")

	// Filter games via ':filter eco:E' or player query
	sess.ExecuteColonCommand("filter Nakamura")
	assert.Equal(t, 2, sess.TotalMatched)

	// Export PGN via ':export'
	exportPath := filepath.Join(tmpDir, "out.pgn")
	sess.ExecuteColonCommand("export " + exportPath)
	exportedBytes, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	assert.Contains(t, string(exportedBytes), "Carlsen, Magnus")
}

func TestRunCLIRenderOnceKittyAndUnicode(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := RunCLI([]string{
		"--render-once",
		"--kitty",
		"--theme", "walnut",
		"--piece-set", "merida",
		"--fen", "rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 2",
	}, strings.NewReader(""), &out, &errBuf)
	assert.Equal(t, 0, code)
	assert.Empty(t, errBuf.String())
	assert.Contains(t, out.String(), "\x1b_Ga=T,f=100,i=42,p=1")
	assert.Contains(t, out.String(), "FEN: rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq")

	out.Reset()
	code = RunCLI([]string{
		"--render-once",
		"--unicode",
		"--theme", "slate",
	}, strings.NewReader(""), &out, &errBuf)
	assert.Equal(t, 0, code)
	assert.Contains(t, out.String(), "♚")
	assert.Contains(t, out.String(), "FEN:")
}

func TestFootSixelBoardRenderingAndAutoUpgrade(t *testing.T) {
	assert.Equal(t, GraphicsSixel, DetectBestGraphicsModeFromEnv(func(k string) string {
		if k == "TERM" {
			return "foot"
		}
		return ""
	}))
	assert.Equal(t, GraphicsSixel, DetectBestGraphicsModeFromEnv(func(k string) string {
		if k == "TERM" {
			return "foot-extra"
		}
		return ""
	}))
	assert.Equal(t, GraphicsKitty, DetectBestGraphicsModeFromEnv(func(k string) string {
		if k == "TERM" {
			return "xterm-ghostty"
		}
		return ""
	}))

	// Verify terminal capability probe parsing (Kitty i=31 OK, DA1 Sixel ;4, XTWINOPS cell size)
	probeEvs, consumed := ParseInputEvents([]byte("\x1b_Gi=31;OK\x1b\\\x1b[?62;4;22;28c\x1b[6;29;14t"))
	assert.Equal(t, len("\x1b_Gi=31;OK\x1b\\\x1b[?62;4;22;28c\x1b[6;29;14t"), consumed)
	require.Len(t, probeEvs, 3)
	assert.Equal(t, KeyTermCapKitty, probeEvs[0].Kind)
	assert.Equal(t, KeyTermCapSixel, probeEvs[1].Kind)
	assert.Equal(t, KeyTermCellPixels, probeEvs[2].Kind)
	assert.Equal(t, 14, probeEvs[2].MouseCol)
	assert.Equal(t, 29, probeEvs[2].MouseRow)

	// Verify Sixel session rendering & caching at foot HiDPI geometry (14x29 cell pixels)
	sess := NewSession(SessionConfig{
		Graphics:        GraphicsSixel,
		HasGraphicsMode: true,
		ThemeID:         "tournament-green",
		PieceSetID:      "cburnett",
	})
	defer sess.Close()

	scr := NewScreen(140, 40)
	geom := TermGeometry{Cols: 140, Rows: 40, XPixel: 1960, YPixel: 1160}
	var out bytes.Buffer

	require.NoError(t, sess.Render(scr, geom, &out, true))
	frame1 := out.String()
	assert.Contains(t, frame1, "SIXEL HD")
	assert.Contains(t, frame1, "\x1bP0;1;0q\"1;1;")
	assert.NotContains(t, frame1, "\x1b_Ga=T")

	// Second render with unchanged board must NOT re-emit Sixel payload
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	assert.NotContains(t, out.String(), "\x1bP0;1;0q\"1;1;")

	// Running --kitty inside TERM=foot must automatically render Sixel instead of an invisible Kitty APC box
	t.Setenv("TERM", "foot")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")
	t.Setenv("WEZTERM_EXECUTABLE", "")

	var cliOut, cliErr bytes.Buffer
	rc := RunCLI([]string{"--render-once", "--kitty"}, strings.NewReader(""), &cliOut, &cliErr)
	assert.Equal(t, 0, rc)
	assert.Contains(t, cliOut.String(), "\x1bP0;1;0q\"1;1;512;512")
}
