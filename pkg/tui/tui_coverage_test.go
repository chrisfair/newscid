package tui

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/engine"
	"newscid/pkg/fairplay"
	"newscid/pkg/scout"
	"newscid/pkg/tree"
)

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met before timeout")
}

func createSamplePGN(t *testing.T, dir string) string {
	t.Helper()
	pgn := `[Event "Spring Open"]
[Site "Oslo"]
[Date "2025.04.10"]
[Round "1"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[WhiteElo "2850"]
[BlackElo "2800"]
[ECO "C65"]
[Result "1-0"]

1. e4! {Best by test} e5 (1... c5 2. Nf3 d6) (1... e6 2. d4 d5) 2. Nf3 Nc6 3. Bb5 Nf6 4. O-O Be7 1-0

[Event "Autumn Cup"]
[Site "London"]
[Date "2024.11.02"]
[Round "2"]
[White "Nakamura, Hikaru"]
[Black "Carlsen, Magnus"]
[WhiteElo "2805"]
[BlackElo "2845"]
[ECO "E60"]
[Result "0-1"]

1. d4 Nf6 2. c4 g6 3. Nf3 Bg7 0-1

[Event "Club Rapid"]
[Site "Berlin"]
[Date "2023.06.15"]
[Round "3"]
[White "Alice"]
[Black "Bob"]
[WhiteElo "1900"]
[BlackElo "1850"]
[ECO "B01"]
[Result "1/2-1/2"]

1. e4 d5 2. exd5 Qxd5 1/2-1/2
`
	p := filepath.Join(dir, "sample.pgn")
	require.NoError(t, os.WriteFile(p, []byte(pgn), 0644))
	return p
}

func createSampleScidDB(t *testing.T, dir, name string) string {
	t.Helper()
	base := filepath.Join(dir, name)
	db := scid.NewDatabase()
	require.NoError(t, db.Create(base, "Test DB"))

	pgns := []string{
		`[Event "Masters"]
[Site "Wijk"]
[Date "2026.01.15"]
[White "Carlsen, Magnus"]
[Black "Fabiano, Caruana"]
[WhiteElo "2860"]
[BlackElo "2810"]
[ECO "C65"]
[Result "1-0"]

1. e4 e5 (1... c5 2. Nf3 d6) (1... e6 2. d4 d5) 2. Nf3 Nc6 3. Bb5 Nf6 1-0`,
		`[Event "Masters"]
[Site "Wijk"]
[Date "2026.01.16"]
[White "Fabiano, Caruana"]
[Black "Carlsen, Magnus"]
[WhiteElo "2810"]
[BlackElo "2860"]
[ECO "D37"]
[Result "1/2-1/2"]

1. d4 d5 2. c4 e6 3. Nc3 Nf6 1/2-1/2`,
	}
	for _, raw := range pgns {
		g, err := chess.ParsePGN(raw)
		require.NoError(t, err)
		_, err = db.SaveGame(g)
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())
	return base + ".si4"
}

func createFakeUCIEngine(t *testing.T, dir string) string {
	t.Helper()
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    uci)
      echo "id name FakeFish 1.0"
      echo "id author Test"
      echo "option name MultiPV type spin default 1 min 1 max 5"
      echo "option name Threads type spin default 1 min 1 max 4"
      echo "option name Hash type spin default 16 min 1 max 128"
      echo "uciok"
      ;;
    isready)
      echo "readyok"
      ;;
    go*)
      echo "info depth 12 multipv 1 score cp 42 nps 120000 pv e2e4 e7e5 g1f3"
      echo "info depth 12 multipv 2 score cp 28 nps 120000 pv d2d4 d7d5"
      echo "bestmove e2e4"
      ;;
    quit)
      exit 0
      ;;
  esac
done
`
	p := filepath.Join(dir, "fake-uci.sh")
	require.NoError(t, os.WriteFile(p, []byte(script), 0755))
	return p
}

func TestSixelCoverage(t *testing.T) {
	assert.Equal(t, "KITTY HD", GraphicsKitty.String())
	assert.Equal(t, "SIXEL HD", GraphicsSixel.String())
	assert.Equal(t, "UNICODE RGB", GraphicsUnicode.String())

	envMap := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	assert.True(t, IsFootTerminal(envMap(map[string]string{"TERM": "foot"})))
	assert.True(t, IsFootTerminal(envMap(map[string]string{"TERM_PROGRAM": "foot"})))
	assert.False(t, IsFootTerminal(envMap(map[string]string{"TERM": "xterm-256color"})))

	assert.True(t, DetectSixelSupportFromEnv(envMap(map[string]string{"NEWSCID_SIXEL": "force"})))
	assert.False(t, DetectSixelSupportFromEnv(envMap(map[string]string{"NEWSCID_SIXEL": "off", "TERM": "foot"})))
	assert.True(t, DetectSixelSupportFromEnv(envMap(map[string]string{"TERM": "mlterm"})))
	assert.True(t, DetectSixelSupportFromEnv(envMap(map[string]string{"TERM_PROGRAM": "contour"})))
	assert.True(t, DetectSixelSupportFromEnv(envMap(map[string]string{"WT_SESSION": "abc"})))
	assert.False(t, DetectSixelSupportFromEnv(envMap(map[string]string{})))

	assert.Equal(t, GraphicsKitty, DetectBestGraphicsModeFromEnv(envMap(map[string]string{"NEWSCID_GRAPHICS": "kitty"})))
	assert.Equal(t, GraphicsSixel, DetectBestGraphicsModeFromEnv(envMap(map[string]string{"NEWSCID_GRAPHICS": "sixel"})))
	assert.Equal(t, GraphicsUnicode, DetectBestGraphicsModeFromEnv(envMap(map[string]string{"NEWSCID_GRAPHICS": "unicode"})))
	assert.Equal(t, GraphicsSixel, DetectBestGraphicsModeFromEnv(envMap(map[string]string{"NEWSCID_KITTY": "0", "TERM": "foot"})))
	assert.Equal(t, GraphicsUnicode, DetectBestGraphicsModeFromEnv(envMap(map[string]string{"NEWSCID_KITTY": "false"})))
	assert.Equal(t, GraphicsUnicode, DetectBestGraphicsModeFromEnv(envMap(map[string]string{})))

	assert.Empty(t, EncodeSixelPlacement(1, 1, nil))
	assert.Empty(t, EncodeSixelImage(image.NewRGBA(image.Rect(0, 0, 0, 0))))

	// Small <=256 color image with placement
	small := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if x < 4 {
				small.SetRGBA(x, y, color.RGBA{R: 200, G: 50, B: 50, A: 255})
			} else {
				small.SetRGBA(x, y, color.RGBA{R: 50, G: 200, B: 50, A: 255})
			}
		}
	}
	placed := EncodeSixelPlacement(3, 5, small)
	assert.True(t, strings.HasPrefix(placed, "\x1b[3;5H\x1bP0;1;0q\"1;1;8;8"))
	assert.True(t, strings.HasSuffix(placed, "\x1b\\"))
	unplaced := EncodeSixelPlacement(0, 0, small)
	assert.True(t, strings.HasPrefix(unplaced, "\x1bP0;1;0q"))

	// Rich image with >256 unique RGB555 buckets to cover palette quantization passes & closest-color LUT
	rich := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			rich.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 8) & 0xff),
				G: uint8((y * 8) & 0xff),
				B: uint8(((x + y) * 4) & 0xff),
				A: 255,
			})
		}
	}
	richSeq := EncodeSixelImage(rich)
	assert.Contains(t, richSeq, "\x1bP0;1;0q\"1;1;32;32")
	assert.True(t, strings.HasSuffix(richSeq, "\x1b\\"))
}

func TestKittyCoverage(t *testing.T) {
	envMap := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	assert.True(t, DetectKittySupportFromEnv(envMap(map[string]string{"NEWSCID_KITTY": "yes"})))
	assert.False(t, DetectKittySupportFromEnv(envMap(map[string]string{"NEWSCID_KITTY": "no"})))
	assert.True(t, DetectKittySupportFromEnv(envMap(map[string]string{"KITTY_PID": "123"})))
	assert.True(t, DetectKittySupportFromEnv(envMap(map[string]string{"GHOSTTY_RESOURCES_DIR": "/tmp"})))
	assert.True(t, DetectKittySupportFromEnv(envMap(map[string]string{"WEZTERM_PANE": "0"})))
	assert.True(t, DetectKittySupportFromEnv(envMap(map[string]string{"TERM_PROGRAM": "konsole"})))
	assert.True(t, DetectKittySupportFromEnv(envMap(map[string]string{"KONSOLE_VERSION": "230800"})))

	assert.Empty(t, EncodeKittyPNGPlacement(1, 1, 0, 0, 0, 0, nil))
	clamped := EncodeKittyPNGPlacement(1, 1, 0, 0, 0, 0, []byte{1, 2, 3})
	assert.True(t, strings.HasPrefix(clamped, "\x1b[1;1H\x1b_Ga=T,f=100,i=1,p=1,c=1,r=1,C=1,q=2,m=0;"))
	assert.Equal(t, "\x1b_Ga=d,d=A,q=2\x1b\\", EncodeKittyDeleteAll())

	// Standalone ESC
	evs, n := ParseInputEvents([]byte{0x1b})
	assert.Equal(t, 1, n)
	require.Len(t, evs, 1)
	assert.Equal(t, KeyEsc, evs[0].Kind)

	// Incomplete APC, CSI, SS3, UTF-8
	evs, n = ParseInputEvents([]byte("\x1b_Gi=31"))
	assert.Equal(t, 0, n)
	assert.Empty(t, evs)
	evs, n = ParseInputEvents([]byte("\x1b["))
	assert.Equal(t, 0, n)
	assert.Empty(t, evs)
	evs, n = ParseInputEvents([]byte("\x1b[12"))
	assert.Equal(t, 0, n)
	assert.Empty(t, evs)
	evs, n = ParseInputEvents([]byte("\x1bO"))
	assert.Equal(t, 0, n)
	assert.Empty(t, evs)
	evs, n = ParseInputEvents([]byte{0xe2, 0x82})
	assert.Equal(t, 0, n)
	assert.Empty(t, evs)

	// Overlong CSI without final byte (>48 bytes) falls back to KeyEsc
	overlong := append([]byte("\x1b["), bytes.Repeat([]byte("1"), 52)...)
	evs, n = ParseInputEvents(overlong)
	assert.Greater(t, n, 0)
	assert.Equal(t, KeyEsc, evs[0].Kind)

	// SS3 sequences + Alt+key + control chars + UTF-8 rune
	seq := []byte(
		"\x1bOA\x1bOB\x1bOC\x1bOD\x1bOH\x1bOF\x1bOP\x1bOQ\x1bOR\x1bOS" +
			"\x1bx" +
			"\n\t\x7f\x08\x03\x11\x0c\x15\x17 \x01♞",
	)
	evs, n = ParseInputEvents(seq)
	assert.Equal(t, len(seq), n)
	kinds := make([]KeyKind, len(evs))
	for i, e := range evs {
		kinds[i] = e.Kind
	}
	assert.Equal(t, []KeyKind{
		KeyUp, KeyDown, KeyRight, KeyLeft, KeyHome, KeyEnd, KeyF1, KeyF2, KeyF3, KeyF4,
		KeyRune,
		KeyEnter, KeyTab, KeyBackspace, KeyBackspace, KeyCtrlC, KeyCtrlQ, KeyCtrlL, KeyCtrlU, KeyCtrlW, KeySpace, KeyRune, KeyRune,
	}, kinds)
	assert.True(t, evs[10].Alt)
	assert.Equal(t, 'x', evs[10].Rune)
	assert.True(t, evs[len(evs)-2].Ctrl)
	assert.Equal(t, 'a', evs[len(evs)-2].Rune)
	assert.Equal(t, '♞', evs[len(evs)-1].Rune)

	// SGR mouse variants & malformed
	mouseSeq := []byte("\x1b[<65;5;6M\x1b[<32;7;8M\x1b[<;1;2M\x1b[<1;2M")
	evs, n = ParseInputEvents(mouseSeq)
	assert.Equal(t, len(mouseSeq), n)
	require.Len(t, evs, 2)
	assert.Equal(t, MouseWheelDown, evs[0].MouseAction)
	assert.Equal(t, MouseDrag, evs[1].MouseAction)

	// Kitty CSI-u variants
	csiu := []byte(
		"\x1b[;u\x1b[13u\x1b[9u\x1b[127u\x1b[32u" +
			"\x1b[113;5u\x1b[108;5u\x1b[117;5u\x1b[119;5u" +
			"\x1b[97;2u\x1b[1u",
	)
	evs, n = ParseInputEvents(csiu)
	assert.Equal(t, len(csiu), n)
	require.Len(t, evs, 9)
	assert.Equal(t, KeyEnter, evs[0].Kind)
	assert.Equal(t, KeyTab, evs[1].Kind)
	assert.Equal(t, KeyBackspace, evs[2].Kind)
	assert.Equal(t, KeySpace, evs[3].Kind)
	assert.Equal(t, KeyCtrlQ, evs[4].Kind)
	assert.Equal(t, KeyCtrlL, evs[5].Kind)
	assert.Equal(t, KeyCtrlU, evs[6].Kind)
	assert.Equal(t, KeyCtrlW, evs[7].Kind)
	assert.Equal(t, KeyRune, evs[8].Kind)
	assert.Equal(t, 'A', evs[8].Rune)

	// Standard CSI navigation & function keys + unknown CSI + DA1 without sixel + bad XTWINOPS
	csiNav := []byte(
		"\x1b[B\x1b[C\x1b[D\x1b[H\x1b[F\x1b[Z" +
			"\x1b[1~\x1b[3~\x1b[4~\x1b[5~\x1b[6~\x1b[11~\x1b[12~\x1b[13~\x1b[14~\x1b[15~\x1b[17~" +
			"\x1b[?62;22c\x1b[6;0;0t\x1b[99X",
	)
	evs, n = ParseInputEvents(csiNav)
	assert.Equal(t, len(csiNav), n)
	require.Len(t, evs, 17)
	assert.Equal(t, KeyDown, evs[0].Kind)
	assert.Equal(t, KeyBackTab, evs[5].Kind)
	assert.Equal(t, KeyF6, evs[16].Kind)
}

func TestScreenAndBoardRenderCoverage(t *testing.T) {
	// Screen helpers
	hc := HexColor("#ff8040")
	assert.Equal(t, RGB{R: 255, G: 128, B: 64, Set: true}, hc)

	scr := NewScreen(5, 5)
	assert.Equal(t, 20, scr.Width)
	assert.Equal(t, 10, scr.Height)
	scr.Resize(5, 5) // clamped to 20x10 -> same size no-op
	scr.Resize(60, 24)
	assert.Equal(t, 60, scr.Width)
	assert.Equal(t, 24, scr.Height)
	scr.Invalidate()
	scr.InvalidateRect(Rect{X: 0, Y: 0, W: 5, H: 5})

	assert.Equal(t, 0, runeCellWidth('\x01'))
	assert.Equal(t, 2, runeCellWidth('中'))
	assert.Equal(t, 1, runeCellWidth('♚'))
	assert.Equal(t, "", TruncateString("hello", 0))
	assert.Equal(t, "…", TruncateString("hello", 1))
	assert.Equal(t, "he…", TruncateString("hello", 3))

	scr.SetCell(-1, 0, 'a', Style{})
	scr.SetCell(0, 0, '\x01', Style{})
	scr.SetCell(scr.Width-1, 0, '中', Style{}) // wide char at last col becomes space
	scr.SetCell(1, 0, '中', Style{Bold: true, Dim: true, Italic: true, Underline: true, FG: ColAccentCyan, BG: ColBgBase})
	assert.True(t, scr.Cells[2].Continuation)

	assert.Equal(t, 0, scr.WriteText(0, -1, 10, "hi", Style{}))
	assert.Equal(t, 0, scr.WriteText(0, 0, 0, "hi", Style{}))
	assert.Greater(t, scr.WriteText(58, 1, 10, "a\x01bcde", Style{}), 0)

	scr.DrawBox(Rect{X: 0, Y: 0, W: 1, H: 1}, "T", "B", false, ColBgPanel)
	scr.DrawBox(Rect{X: 0, Y: 2, W: 30, H: 6}, "Title", "Badge", true, ColBgPanel)

	// Eval bar edge cases
	scr.DrawVerticalEvalBar(0, 0, 2, 0, 0, false) // height < 4 no-op
	scr.DrawVerticalEvalBar(2, 2, 10, 0, 3, false)
	scr.DrawVerticalEvalBar(4, 2, 10, 0, -2, true)
	scr.DrawVerticalEvalBar(6, 2, 10, 5000, 0, false)
	scr.DrawVerticalEvalBar(8, 2, 10, -5000, 0, true)

	// WDL bar
	scr.DrawWDLBar(10, 2, 0, 1, 1, 1, ColBgPanel)
	scr.DrawWDLBar(10, 2, 10, 0, 0, 0, ColBgPanel)
	scr.DrawWDLBar(10, 2, 10, 6, 3, 1, ColBgPanel)
	scr.DrawWDLBar(10, 2, 3, 2, 2, 0, ColBgPanel) // wCols+dCols > width clamp

	// Unicode board edge cases
	checkBoard, err := chess.ParseFEN("4k3/8/8/8/8/8/4Q3/4K3 b - - 0 1")
	require.NoError(t, err)
	scr.DrawUnicodeBoard(Rect{X: 0, Y: 0, W: 14, H: 10}, nil, BoardRenderOptions{ThemeID: "unknown-theme"})
	scr.DrawUnicodeBoard(Rect{X: 0, Y: 0, W: 58, H: 24}, checkBoard, BoardRenderOptions{
		ThemeID:      "slate",
		Flipped:      true,
		SelectedSq:   chess.E8,
		LastFrom:     chess.E1,
		LastTo:       chess.E2,
		LegalTargets: []chess.Square{chess.D8, chess.E2},
		CheckKingSq:  chess.NoSquare,
	})
	glyph, _ := unicodePieceGlyph(chess.Empty)
	assert.Equal(t, ' ', glyph)

	// Flush with Ch==0 and Cutout
	scr.Cells[3*scr.Width+3] = Cell{Ch: 0, Style: Style{FG: ColTextPrimary, BG: ColBgBase}}
	scr.Cutout = Rect{X: 10, Y: 10, W: 4, H: 4}
	var out bytes.Buffer
	require.NoError(t, scr.Flush(&out, true, true, ""))
	assert.Contains(t, out.String(), "\x1b[?2026h")

	// BoardRender edge cases
	opt := BoardRenderOptions{
		ThemeID:      "unknown-theme-xyz",
		PieceSetID:   "",
		SquarePx:     16, // <32 clamped to 72
		ShowCoords:   true,
		CheckKingSq:  chess.NoSquare,
		LegalTargets: []chess.Square{chess.NoSquare, chess.E4},
		Arrows: []BoardArrow{
			{From: chess.E2, To: chess.E2, Kind: ArrowUserHint},       // skipped
			{From: chess.NoSquare, To: chess.E4, Kind: ArrowUserHint}, // skipped
			{From: chess.E2, To: chess.E4, Kind: ArrowUserHint},
		},
	}
	assert.NotZero(t, opt.Hash(nil))
	img, err := RenderBoardImage(nil, opt)
	require.NoError(t, err)
	assert.Equal(t, 72*8, img.Bounds().Dx())

	// SquarePx > 160 clamped to 160 & unknown piece set falls back to cburnett
	optLarge := BoardRenderOptions{PieceSetID: "nonexistent-set", SquarePx: 200, ShowCoords: true}
	imgLarge, err := RenderBoardImage(checkBoard, optLarge)
	require.NoError(t, err)
	assert.Equal(t, 160*8, imgLarge.Bounds().Dx())

	r, g, b := ParseHexRGB("bad")
	assert.Equal(t, uint8(128), r)
	assert.Equal(t, uint8(128), g)
	assert.Equal(t, uint8(128), b)
	r, g, b = ParseHexRGB("#zzzzzz")
	assert.Equal(t, uint8(128), r)
	assert.Equal(t, uint8(128), g)
	assert.Equal(t, uint8(128), b)

	assert.Equal(t, chess.NoSquare, DisplayCoordsToSquare(-1, 0, false))
	assert.Equal(t, chess.NoSquare, DisplayCoordsToSquare(0, 8, true))
	assert.Equal(t, uint8(0), blendByte(0, 0, -1))
	assert.Equal(t, uint8(255), blendByte(255, 255, 2))
	blendPixel(img, -1, -1, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	blendPixel(img, 0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 0})
	drawTinyGlyph(img, 0, 0, '?', 1, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
}

func TestRunCLIAndInteractiveLoop(t *testing.T) {
	tmpDir := t.TempDir()
	pgnPath := createSamplePGN(t, tmpDir)
	si4Path := createSampleScidDB(t, tmpDir, "cli")

	var out, errBuf bytes.Buffer
	assert.Equal(t, 0, RunCLI([]string{"--version"}, strings.NewReader(""), &out, &errBuf))
	assert.NotEmpty(t, out.String())

	out.Reset()
	assert.Equal(t, 0, RunCLI([]string{"--help"}, strings.NewReader(""), &out, &errBuf))
	assert.Contains(t, out.String(), "NewScid TUI")

	errBuf.Reset()
	assert.Equal(t, 2, RunCLI([]string{"--unknown-flag"}, strings.NewReader(""), &out, &errBuf))

	// Positional PGN + --force-kitty + --ply + --cols/--rows clamping + --
	out.Reset()
	errBuf.Reset()
	rc := RunCLI([]string{pgnPath, "--render-once", "--force-kitty", "--ply", "2", "--cols", "4", "--rows", "2", "--", "ignored"}, strings.NewReader(""), &out, &errBuf)
	assert.Equal(t, 0, rc)
	assert.Contains(t, out.String(), "Carlsen, Magnus vs Nakamura, Hikaru")
	assert.Contains(t, out.String(), "\x1b_Ga=T,f=100")

	// Positional SI4 + --sixel + in-check FEN
	out.Reset()
	rc = RunCLI([]string{si4Path, "--render-once", "--sixel", "--fen", "4k3/8/8/8/8/8/4Q3/4K3 b - - 0 1"}, strings.NewReader(""), &out, &errBuf)
	assert.Equal(t, 0, rc)
	assert.Contains(t, out.String(), "\x1bP0;1;0q")

	// Error writer for --render-once in Kitty, Sixel, and Unicode modes
	assert.Equal(t, 1, RunCLI([]string{"--render-once", "--force-kitty"}, strings.NewReader(""), errWriter{}, &errBuf))
	assert.Equal(t, 1, RunCLI([]string{"--render-once", "--sixel"}, strings.NewReader(""), errWriter{}, &errBuf))
	assert.Equal(t, 1, RunCLI([]string{"--render-once", "--unicode"}, strings.NewReader(""), errWriter{}, &errBuf))

	// Interactive loop: send cell pixel report, navigation keys, and 'q' to quit cleanly
	out.Reset()
	errBuf.Reset()
	rc = RunCLI([]string{"--unicode", pgnPath}, strings.NewReader("\x1b[6;20;10t\x1b[C\x1b[Dq"), &out, &errBuf)
	assert.Equal(t, 0, rc)
	assert.Contains(t, out.String(), "\x1b[?1049h")
	assert.Contains(t, out.String(), "\x1b[?1049l")

	// Interactive loop with Ctrl+C and Kitty cleanup sequence
	out.Reset()
	rc = RunCLI([]string{"--force-kitty"}, strings.NewReader("\x03"), &out, &errBuf)
	assert.Equal(t, 0, rc)
	assert.Contains(t, out.String(), EncodeKittyDeleteAll())

	// Interactive loop with EOF draining events and RedrawCh tick
	sess := NewSession(SessionConfig{Graphics: GraphicsUnicode, HasGraphicsMode: true})
	defer sess.Close()
	pr, pw := io.Pipe()
	done := make(chan int, 1)
	out.Reset()
	go func() {
		done <- runInteractiveLoop(sess, pr, &out, &errBuf)
	}()
	sess.notifyRedraw()
	time.Sleep(30 * time.Millisecond)
	_, _ = pw.Write([]byte("f"))
	_ = pw.Close()
	assert.Equal(t, 0, <-done)

	// Interactive loop write failure on initSeq
	sess2 := NewSession(SessionConfig{Graphics: GraphicsUnicode, HasGraphicsMode: true})
	defer sess2.Close()
	assert.Equal(t, 1, runInteractiveLoop(sess2, strings.NewReader("q"), errWriter{}, &errBuf))

	// term_unix helpers on invalid FD
	assert.Equal(t, TermGeometry{Cols: 120, Rows: 36}, GetTermGeometry(-1))
	assert.False(t, IsTerminalFD(-1))
	_, err := EnterRawMode(-1)
	assert.Error(t, err)
	ch, stop := SubscribeResize()
	require.NotNil(t, ch)
	stop()
}

func TestSessionPGNDBFilterAndDiscovery(t *testing.T) {
	tmpDir := t.TempDir()
	pgnPath := createSamplePGN(t, tmpDir)
	si4Path := createSampleScidDB(t, tmpDir, "club")

	// Empty PGN & empty Scid DB
	emptyPGN := filepath.Join(tmpDir, "empty.pgn")
	require.NoError(t, os.WriteFile(emptyPGN, []byte(""), 0644))
	emptyDBBase := filepath.Join(tmpDir, "emptydb")
	edb := scid.NewDatabase()
	require.NoError(t, edb.Create(emptyDBBase, "Empty"))
	require.NoError(t, edb.Close())

	// Run inside tmpDir so RefreshDiscoveredBases finds both .si4 and .pgn
	t.Chdir(tmpDir)

	sess := NewSession(SessionConfig{
		PGNPath:         pgnPath,
		GameNumber:      2,
		Graphics:        GraphicsUnicode,
		HasGraphicsMode: true,
	})
	defer sess.Close()

	assert.True(t, sess.IsPGNSource)
	assert.Equal(t, 3, sess.TotalMatched)
	assert.Equal(t, 2, sess.ActiveGameIndex)
	assert.NotEmpty(t, sess.DiscoveredBases)

	// Wait for PGN Opening Tree worker
	waitUntil(t, func() bool {
		sess.mu.Lock()
		defer sess.mu.Unlock()
		return !sess.TreeBusy
	})

	// PGN filtering across every supported token and matchSummaryCriteria branch
	sess.ApplyFilterQuery("w:Carlsen b:Nakamura e:Spring s:Oslo eco:C65 res:1-0 minelo:2800 maxelo:2900 year:2025")
	assert.Equal(t, 1, sess.TotalMatched)
	sess.ApplyFilterQuery("w:Nobody")
	assert.Equal(t, 0, sess.TotalMatched)
	sess.ApplyFilterQuery("b:Nobody")
	assert.Equal(t, 0, sess.TotalMatched)
	sess.ApplyFilterQuery("p:Nobody")
	assert.Equal(t, 0, sess.TotalMatched)
	sess.ApplyFilterQuery("e:Nobody")
	assert.Equal(t, 0, sess.TotalMatched)
	sess.ApplyFilterQuery("eco:Z99")
	assert.Equal(t, 0, sess.TotalMatched)
	sess.ApplyFilterQuery("res:0-1")
	assert.Equal(t, 1, sess.TotalMatched)
	sess.ApplyFilterQuery("minelo:2999")
	assert.Equal(t, 0, sess.TotalMatched)
	sess.ApplyFilterQuery("unknown:tok")
	assert.Equal(t, 0, sess.TotalMatched)
	sess.ApplyFilterQuery("Carlsen")
	assert.Equal(t, 2, sess.TotalMatched)
	sess.ApplyFilterQuery("pos")
	assert.GreaterOrEqual(t, sess.TotalMatched, 1)
	sess.ApplyFilterQuery("")
	assert.Equal(t, 3, sess.TotalMatched)

	// LoadGameNumber out of range on PGN
	assert.Error(t, sess.LoadGameNumber(99))
	// SaveCurrentGameToDB without ScidDB returns error
	assert.Error(t, sess.SaveCurrentGameToDB())
	// ExportCurrentPGN with empty path uses default filename inside tmpDir
	require.NoError(t, sess.ExportCurrentPGN(""))
	_, err := os.Stat(filepath.Join(tmpDir, "newscid-export.pgn"))
	require.NoError(t, err)

	// OpenPath edge cases
	assert.Error(t, sess.OpenPath(""))
	assert.Error(t, sess.OpenPath("~/definitely-does-not-exist-newscid.si4"))
	assert.Error(t, sess.OpenPath(filepath.Join(tmpDir, "missing.pgn")))

	// Empty PGN and empty Scid DB
	require.NoError(t, sess.OpenPath(emptyPGN))
	assert.Equal(t, 0, sess.TotalMatched)
	require.NoError(t, sess.OpenPath(emptyDBBase+".si4"))
	assert.Equal(t, 0, sess.TotalMatched)

	// Open populated Scid DB & save current game
	require.NoError(t, sess.OpenPath(si4Path))
	assert.Equal(t, 2, sess.TotalMatched)
	require.NoError(t, sess.SaveCurrentGameToDB())
	assert.Equal(t, 3, sess.TotalMatched)

	// No database open -> LoadGameNumber errors
	sessNoDB := NewSession(SessionConfig{
		DBPath:     filepath.Join(tmpDir, "nonexistent.si4"),
		InitialFEN: "not-a-fen",
	})
	defer sessNoDB.Close()
	assert.Error(t, sessNoDB.LoadGameNumber(1))
	assert.True(t, sessNoDB.StatusIsErr)
}

func TestSessionMovesVariationsAndArrows(t *testing.T) {
	sess := NewSession(SessionConfig{Graphics: GraphicsUnicode, HasGraphicsMode: true})
	defer sess.Close()

	// PlayMoveInput: empty, move number prefix, UCI, illegal
	require.NoError(t, sess.PlayMoveInput("  "))
	require.NoError(t, sess.PlayMoveInput("1.e4"))
	require.NoError(t, sess.PlayMoveInput("1...e5"))
	require.NoError(t, sess.PlayMoveInput("g1f3"))
	assert.Error(t, sess.PlayMoveInput("Zz9"))
	assert.Error(t, sess.ApplyLegalMove(chess.Move{From: chess.A1, To: chess.H8}))

	// Rewind and step along existing path via ApplyLegalMove
	sess.JumpToPly(-5)
	assert.Equal(t, 0, sess.CurPly)
	require.NoError(t, sess.ApplyLegalMove(chess.Move{From: chess.E2, To: chess.E4}))
	assert.Equal(t, 1, sess.CurPly)

	// Branch mid-game (at ply 1, play 1...c5 instead of 1...e5)
	require.NoError(t, sess.PlayMoveInput("c5"))
	assert.Equal(t, 2, sess.CurPly)

	// CycleVariation with 1 variation and with 2 variations
	sess.JumpToPly(1)
	require.NoError(t, sess.PlayMoveInput("e6"))
	// Now go back to ply 1 and cycle forward & backward
	sess.JumpToPly(0)
	assert.False(t, sess.CycleVariation(1))

	gWithVars, err := chess.ParsePGN("1. e4 e5 (1... c5 2. Nf3) (1... e6 2. d4) 2. Nf3 Nc6 *")
	require.NoError(t, err)
	sess.SetGame(gWithVars, 1)
	sess.JumpToPly(1)
	assert.True(t, sess.CycleVariation(1))
	assert.True(t, sess.CycleVariation(-1))
	sess.JumpToPly(999)
	assert.Equal(t, len(sess.Path), sess.CurPly)

	// Corrupted BoardFEN fallback in JumpToPly
	sess.Path[0].BoardFEN = "invalid-fen"
	sess.JumpToPly(1)
	assert.Equal(t, chess.WP, sess.Board.PieceAt(chess.E4))

	// Auto-promotion to Queen when Promo == None
	require.NoError(t, sess.LoadFEN("8/4P3/8/8/8/8/8/4K2k w - - 0 1"))
	require.NoError(t, sess.ApplyLegalMove(chess.Move{From: chess.E7, To: chess.E8}))
	assert.Equal(t, chess.WQ, sess.Board.PieceAt(chess.E8))

	// LegalTargetsFrom edge cases (invalid sq, empty, opponent piece, capture)
	require.NoError(t, sess.LoadFEN("4k3/8/8/3p4/4P3/8/8/4K3 w - - 0 1"))
	targs, mask := sess.LegalTargetsFrom(chess.NoSquare)
	assert.Nil(t, targs)
	assert.Zero(t, mask)
	targs, _ = sess.LegalTargetsFrom(chess.A1)
	assert.Nil(t, targs)
	targs, _ = sess.LegalTargetsFrom(chess.D5) // black pawn on white turn
	assert.Nil(t, targs)
	targs, mask = sess.LegalTargetsFrom(chess.E4)
	assert.Contains(t, targs, chess.D5)
	assert.NotZero(t, mask&(uint64(1)<<uint(chess.D5)))

	// SetGame(nil, 0) and FEN tag fallback
	sess.SetGame(nil, 0)
	assert.Equal(t, 0, sess.CurPly)
	gTagFEN := chess.NewGame()
	gTagFEN.SetTag("FEN", "4k3/8/8/8/8/8/4Q3/4K3 b - - 0 1")
	sess.SetGame(gTagFEN, 0)
	assert.Equal(t, chess.Black, sess.Board.Turn)

	// ActiveBoardArrows with ShowArrows=false, EngineEnabled=true (PV1+PV2), and TreeResult (dedup + SAN fallback)
	sess.ResetNewGame("")
	sess.ShowArrows = false
	assert.Nil(t, sess.ActiveBoardArrows())

	sess.ShowArrows = true
	sess.EngineEnabled = true
	sess.EvalPositionFEN = sess.Board.FEN()
	sess.LastEval = engine.EvalInfo{
		Lines: []engine.PVLine{
			{MultiPV: 1, MoveUCI: "e2e4"},
			{MultiPV: 2, MoveUCI: "d2d4"},
			{MultiPV: 3, MoveUCI: "c2c4"},
		},
	}
	sess.ActiveTab = TabTree
	sess.TreeResult = &tree.TreeResult{
		FEN: sess.Board.FEN(),
		Moves: []*tree.MoveStat{
			{Move: "e4", UCI: "e2e4", Count: 10}, // duplicate of PV1
			{Move: "Nf3", UCI: "", Count: 5},     // SAN -> UCI fallback
		},
	}
	arrows := sess.ActiveBoardArrows()
	assert.Len(t, arrows, 3)
	assert.Equal(t, ArrowEnginePV1, arrows[0].Kind)
	assert.Equal(t, ArrowEnginePV2, arrows[1].Kind)
	assert.Equal(t, ArrowOpeningBook, arrows[2].Kind)
}

func TestSessionEngineForensicsScoutAndRendering(t *testing.T) {
	tmpDir := t.TempDir()
	si4Path := createSampleScidDB(t, tmpDir, "forensics")
	pgnPath := createSamplePGN(t, tmpDir)
	fakeEngine := createFakeUCIEngine(t, tmpDir)

	sess := NewSession(SessionConfig{
		DBPath:          si4Path,
		EnginePath:      fakeEngine,
		AutoEngine:      true,
		Graphics:        GraphicsKitty,
		HasGraphicsMode: true,
	})
	defer sess.Close()

	assert.True(t, sess.EngineEnabled)
	assert.Equal(t, "FakeFish 1.0", sess.EngineName)

	// Wait briefly for fake engine eval callback
	waitUntil(t, func() bool {
		sess.mu.Lock()
		defer sess.mu.Unlock()
		return sess.LastEval.Depth >= 12
	})

	// AdjustMultiPV & ToggleEngine & StartEngine already-running path
	sess.AdjustMultiPV(1)
	assert.Equal(t, 3, sess.MultiPV)
	sess.AdjustMultiPV(10)
	assert.Equal(t, 5, sess.MultiPV)
	sess.AdjustMultiPV(1) // no-op at max
	sess.AdjustMultiPV(-10)
	assert.Equal(t, 1, sess.MultiPV)
	sess.ToggleEngine() // pause
	assert.False(t, sess.EngineEnabled)
	sess.ToggleEngine() // resume existing engine
	assert.True(t, sess.EngineEnabled)

	// Cycle theme, piece set, notation
	sess.CycleTheme()
	sess.CyclePieceSet()
	for i := 0; i < 4; i++ {
		sess.CycleNotation()
	}

	// Render with Engine ON, Comments/NAGs/Variations, and Graphics transitions (Kitty -> Sixel -> Unicode -> Kitty)
	scr := NewScreen(132, 38)
	geom := TermGeometry{Cols: 132, Rows: 38, XPixel: 1320, YPixel: 760}
	var out bytes.Buffer

	// Load PGN with comments & NAGs so scorebook renders them
	require.NoError(t, sess.OpenPath(pgnPath))
	sess.JumpToPly(2)

	// Score tab with engine Lines populated
	sess.ActiveTab = TabScore
	require.NoError(t, sess.Render(scr, geom, &out, true))
	assert.Contains(t, out.String(), "FakeFish")

	// Score tab with empty Lines but non-empty PV fallback
	sess.mu.Lock()
	sess.LastEval = engine.EvalInfo{Depth: 10, ScoreCp: -120, PV: []string{"g1f3", "b8c6"}}
	sess.mu.Unlock()
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))

	// Switch Kitty -> Sixel -> Unicode (covers Kitty delete on Sixel switch and Sixel erase on Unicode switch)
	sess.SetGraphicsMode(GraphicsSixel)
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	assert.Contains(t, out.String(), EncodeKittyDeleteImage(KittyBoardImageID))
	assert.Contains(t, out.String(), "\x1bP0;1;0q")

	// Modal over Sixel clears Sixel placement
	sess.Mode = ModeHelpModal
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	assert.False(t, sess.sixelPlaced)
	sess.Mode = ModeNormal

	// Sixel -> Kitty transition
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	assert.True(t, sess.sixelPlaced)
	sess.SetGraphicsMode(GraphicsKitty)
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	assert.True(t, sess.kittyPlaced)
	assert.False(t, sess.sixelPlaced)

	// Prompt modes in bottom bar
	for _, m := range []InputMode{ModeMoveInput, ModeSearchFilter, ModeCommand} {
		sess.Mode = m
		sess.InputBuffer = "e4"
		out.Reset()
		require.NoError(t, sess.Render(scr, geom, &out, false))
	}
	sess.Mode = ModeNormal
	sess.SetStatus("error test", true)
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))

	// Render TabTree with populated moves & WDL bars
	sess.ActiveTab = TabTree
	sess.mu.Lock()
	sess.TreeResult = &tree.TreeResult{
		TotalGames: 10,
		Moves: []*tree.MoveStat{
			{Move: "e4", UCI: "e2e4", Count: 6, WhiteWins: 3, Draws: 2, BlackWins: 1, ScorePct: 66.7},
			{Move: "d4", UCI: "d2d4", Count: 4, WhiteWins: 1, Draws: 1, BlackWins: 2, ScorePct: 37.5},
		},
	}
	sess.TreeBusy = true
	sess.TreeCursor = 1
	sess.mu.Unlock()
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))

	// Render TabImbalance with nil and populated report
	sess.ActiveTab = TabImbalance
	savedImp := sess.ImbalanceRep
	sess.ImbalanceRep = nil
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	sess.ImbalanceRep = savedImp

	// Render checkmate (White and Black) & stalemate & material delta
	for _, fen := range []string{
		"7k/5Q2/6K1/8/8/8/8/8 b - - 0 1",          // Black checkmated
		"6k1/8/8/8/8/8/5q2/6K1 w - - 0 1",         // White in check
		"4k3/4Q3/4K3/8/8/8/8/8 b - - 0 1",         // Stalemate
		"rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3", // White checkmated
	} {
		require.NoError(t, sess.LoadFEN(fen))
		out.Reset()
		require.NoError(t, sess.Render(scr, geom, &out, false))
	}

	// Forensics: RunFairPlayAsync & RunScoutAsync on PGN, Scid DB, and scratch game
	sess.ActiveTab = TabForensics
	sess.ForensicsBusy = true
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	sess.ForensicsBusy = false

	// Render rich synthetic FairPlay dossier (with Radar & KeyFindings) and Scout dossier
	sess.ForensicsMode = "fairplay"
	sess.FairPlayReport = &fairplay.PlayerFairPlayDossier{
		PlayerName:               "Carlsen, Magnus",
		TotalGamesAnalyzed:       10,
		TotalDaysActive:          5,
		OverallIssueScore:        12.5,
		OverallClassification:    "Normal",
		HighestRiskTimeControl:   "Blitz",
		HighestControlIssueScore: 15.0,
		TimeControlProfiles: []fairplay.TimeControlIssueProfile{
			{Category: "Blitz", GamesAnalyzed: 10, IntrinsicPerformanceElo: 2860, AvgACPL: 18.2},
		},
		Radar: &fairplay.RadarRiskAssessment{
			Platform:              "lichess",
			CRSScore:              22.0,
			CheatLikelihoodPct:    5,
			RiskTier:              "low",
			AvgEstimatedAccuracy:  91.2,
			ExpectedBandThreshold: 94.0,
			Signals: []fairplay.RadarSignal{
				{Name: "Accuracy Spike", Score: 10, MaxScore: 25, Flagged: true, Detail: "elevated"},
				{Name: "Time Cadence", Score: 2, MaxScore: 20, Flagged: false, Detail: "natural"},
			},
		},
		KeyFindings: []string{"Consistent performance across blitz sessions"},
	}
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))

	sess.ForensicsMode = "scout"
	sess.ScoutReport = &scout.Report{
		PlayerName: "Carlsen, Magnus",
		TotalGames: 20,
		TopOpeningsAsWhite: []scout.LinePerformance{
			{ECO: "C65", Games: 10, Wins: 6, Draws: 3, Losses: 1, WinRate: 75.0},
		},
		TopOpeningsAsBlack: []scout.LinePerformance{
			{ECO: "B90", Games: 10, Wins: 5, Draws: 3, Losses: 2, WinRate: 65.0},
		},
		WeakestLines: []scout.LinePerformance{
			{ECO: "E60", Games: 4, Wins: 1, Draws: 1, Losses: 2, WinRate: 37.5},
		},
		RepertoireRecommendations: []string{"Target King's Indian sidelines"},
	}
	out.Reset()
	require.NoError(t, sess.Render(scr, geom, &out, false))

	// Real async FairPlay & Scout on PGN, Scid DB, and scratch board
	require.NoError(t, sess.OpenPath(pgnPath))
	sess.RunFairPlayAsync("")
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.ForensicsBusy })
	sess.RunFairPlayAsync("") // switches target from White to Black
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.ForensicsBusy })
	sess.RunScoutAsync("")
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.ForensicsBusy })

	require.NoError(t, sess.OpenPath(si4Path))
	sess.RunFairPlayAsync("Carlsen")
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.ForensicsBusy })
	sess.RunScoutAsync("Carlsen")
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.ForensicsBusy })

	scratch := NewSession(SessionConfig{Graphics: GraphicsUnicode, HasGraphicsMode: true})
	defer scratch.Close()
	scratch.RunFairPlayAsync("")
	waitUntil(t, func() bool { scratch.mu.Lock(); defer scratch.mu.Unlock(); return !scratch.ForensicsBusy })

	// Helper formatting functions
	assert.Equal(t, "+M3", formatEvalCompact(0, 3))
	assert.Equal(t, "-M2", formatEvalCompact(0, -2))
	assert.Equal(t, "+1.25", formatEvalCompact(125, 0))
	assert.Equal(t, "-0.50", formatEvalCompact(-50, 0))
	assert.Equal(t, "Yes", yesNo(true))
	assert.Equal(t, "No", yesNo(false))
	assert.Equal(t, "White", boolAdv(true, false))
	assert.Equal(t, "Black", boolAdv(false, true))
	assert.Equal(t, "Equal", boolAdv(true, true))
	assert.Equal(t, "White (+2)", deltaLabel(2))
	assert.Equal(t, "Black (+3)", deltaLabel(-3))
	assert.Equal(t, "Equal", deltaLabel(0))
	assert.Equal(t, 0, clampIdx(5, 0))
	assert.Equal(t, 0, clampIdx(-1, 5))
	assert.Equal(t, 4, clampIdx(10, 5))
	assert.Equal(t, 2, clampIdx(2, 5))
}

func TestSessionEventsCommandsAndMouse(t *testing.T) {
	tmpDir := t.TempDir()
	si4Path := createSampleScidDB(t, tmpDir, "events")
	pgnPath := createSamplePGN(t, tmpDir)
	fakeEngine := createFakeUCIEngine(t, tmpDir)

	sess := NewSession(SessionConfig{
		DBPath:          si4Path,
		Graphics:        GraphicsUnicode,
		HasGraphicsMode: true,
	})
	defer sess.Close()

	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.TreeBusy })

	// Render once to populate hit rects
	scr := NewScreen(132, 38)
	geom := TermGeometry{Cols: 132, Rows: 38, XPixel: 1320, YPixel: 760}
	var out bytes.Buffer
	require.NoError(t, sess.Render(scr, geom, &out, true))

	// Capability auto-upgrade events
	sess.HandleEvent(InputEvent{Kind: KeyTermCapSixel})
	assert.Equal(t, GraphicsSixel, sess.Graphics)
	sess.HandleEvent(InputEvent{Kind: KeyTermCapKitty})
	assert.Equal(t, GraphicsKitty, sess.Graphics)
	sess.HandleEvent(InputEvent{Kind: KeyCtrlL})

	// Help modal open & dismiss
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: '?'})
	assert.Equal(t, ModeHelpModal, sess.Mode)
	sess.HandleEvent(InputEvent{Kind: KeyEsc})
	assert.Equal(t, ModeNormal, sess.Mode)

	// Navigation & Tab keys
	for _, k := range []KeyKind{
		KeyRight, KeyLeft, KeyEnd, KeyHome, KeyBackspace,
		KeyTab, KeyBackTab, KeyF1, KeyF2, KeyF3, KeyF4, KeyF5, KeyF6,
		KeyUp, KeyDown, KeyPgUp, KeyPgDn, KeyEsc,
	} {
		sess.HandleEvent(InputEvent{Kind: k})
	}

	// TabScore Enter opens ModeMoveInput; exercise prompt editing & submit
	sess.ActiveTab = TabScore
	sess.JumpToPly(0)
	sess.HandleEvent(InputEvent{Kind: KeyEnter})
	assert.Equal(t, ModeMoveInput, sess.Mode)
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: 'e'})
	sess.HandleEvent(InputEvent{Kind: KeySpace})
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: '4'})
	sess.HandleEvent(InputEvent{Kind: KeyCtrlW})
	sess.HandleEvent(InputEvent{Kind: KeyCtrlU})
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: 'e'})
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: '5'})
	sess.HandleEvent(InputEvent{Kind: KeyBackspace})
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: '4'})
	sess.HandleEvent(InputEvent{Kind: KeyEnter})
	assert.Equal(t, ModeNormal, sess.Mode)
	assert.Equal(t, 1, sess.CurPly)

	// ModeSearchFilter via '/'
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: '/'})
	assert.Equal(t, ModeSearchFilter, sess.Mode)
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: 'C'})
	sess.HandleEvent(InputEvent{Kind: KeyEnter})
	assert.Equal(t, TabGames, sess.ActiveTab)
	assert.Equal(t, ModeNormal, sess.Mode)

	// ModeCommand via ':' and Esc cancel
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: ':'})
	assert.Equal(t, ModeCommand, sess.Mode)
	sess.HandleEvent(InputEvent{Kind: KeyEsc})
	assert.Equal(t, ModeNormal, sess.Mode)

	// ModeCommand submit
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: ':'})
	for _, r := range "new" {
		sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: r})
	}
	sess.HandleEvent(InputEvent{Kind: KeyEnter})
	assert.Equal(t, 0, sess.CurPly)

	// Normal runes
	for _, r := range []rune{'1', '2', '3', '4', '5', '6', '1', 'l', 'h', 'j', 'k', '$', '0', '[', ']', 'f', 't', 'p', 'n', 'g', 'g', 'a', '=', '-', 'N', 'P', 's', 'x'} {
		sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: r})
	}

	// Opening Tree hotkey 'a' and Enter activation + Space playBestSuggestedMove
	require.NoError(t, sess.OpenPath(si4Path))
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.TreeBusy })
	sess.ActiveTab = TabTree
	sess.JumpToPly(0)
	sess.mu.Lock()
	sess.TreeResult = &tree.TreeResult{
		FEN:   sess.Board.FEN(),
		Moves: []*tree.MoveStat{{Move: "e4", UCI: "e2e4", Count: 5}},
	}
	sess.TreeCursor = 0
	sess.mu.Unlock()
	sess.HandleEvent(InputEvent{Kind: KeyRune, Rune: 'a'})
	assert.Equal(t, 1, sess.CurPly)

	sess.JumpToPly(0)
	sess.HandleEvent(InputEvent{Kind: KeyEnter})
	assert.Equal(t, 1, sess.CurPly)

	sess.JumpToPly(0)
	sess.HandleEvent(InputEvent{Kind: KeySpace})
	assert.Equal(t, 1, sess.CurPly)

	// playBestSuggestedMove with Engine ON and with neither Engine nor Tree
	sess.JumpToPly(0)
	sess.EngineEnabled = true
	sess.EvalPositionFEN = sess.Board.FEN()
	sess.LastEval = engine.EvalInfo{BestMove: "d2d4"}
	sess.playBestSuggestedMove()
	assert.Equal(t, chess.WP, sess.Board.PieceAt(chess.D4))

	sess.EngineEnabled = false
	sess.TreeResult = nil
	sess.playBestSuggestedMove()
	assert.Contains(t, sess.StatusMsg, "Start engine")

	// TabGames & TabBases Enter activation + scrolling
	sess.ActiveTab = TabGames
	sess.scrollActiveTab(1)
	sess.scrollActiveTab(-1)
	sess.activateSelectedInTab()
	assert.Equal(t, TabScore, sess.ActiveTab)

	sess.DiscoveredBases = []scid.DiscoveredDatabase{{Name: "sample", BasePath: pgnPath, Type: "pgn", Exists: true}}
	sess.ActiveTab = TabBases
	sess.scrollActiveTab(1)
	sess.activateSelectedInTab()
	assert.Equal(t, TabGames, sess.ActiveTab)

	// All ExecuteColonCommand branches
	sess.ExecuteColonCommand("")
	sess.ExecuteColonCommand("help")
	assert.Equal(t, ModeHelpModal, sess.Mode)
	sess.Mode = ModeNormal
	sess.ExecuteColonCommand("open")
	assert.Equal(t, TabBases, sess.ActiveTab)
	sess.ExecuteColonCommand("open " + si4Path)
	sess.ExecuteColonCommand("open /nonexistent/file.si4")
	sess.ExecuteColonCommand("fen")
	sess.ExecuteColonCommand("fen invalid")
	sess.ExecuteColonCommand("fen rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1")
	sess.ExecuteColonCommand("game abc")
	sess.ExecuteColonCommand("game 999")
	sess.ExecuteColonCommand("game 1")
	sess.ExecuteColonCommand("filter Carlsen")
	sess.ExecuteColonCommand("save")
	sess.ExecuteColonCommand("save " + filepath.Join(tmpDir, "saved.pgn"))
	sess.ExecuteColonCommand("export " + filepath.Join(tmpDir, "exp.pgn"))
	sess.ExecuteColonCommand("engine /nonexistent/uci-bin")
	sess.ExecuteColonCommand("engine " + fakeEngine)
	sess.StopEngine()
	sess.ExecuteColonCommand("theme")
	sess.ExecuteColonCommand("theme walnut")
	sess.ExecuteColonCommand("pieces")
	sess.ExecuteColonCommand("pieces merida")
	sess.ExecuteColonCommand("notation fan")
	sess.ExecuteColonCommand("notation invalid")
	sess.ExecuteColonCommand("kitty")
	sess.ExecuteColonCommand("sixel")
	sess.ExecuteColonCommand("unicode")
	sess.ExecuteColonCommand("fairplay Carlsen")
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.ForensicsBusy })
	sess.ExecuteColonCommand("scout Carlsen")
	waitUntil(t, func() bool { sess.mu.Lock(); defer sess.mu.Unlock(); return !sess.ForensicsBusy })
	sess.ExecuteColonCommand("new")
	sess.ExecuteColonCommand("move e4")
	sess.ExecuteColonCommand("move badmove")
	sess.ExecuteColonCommand("e5") // direct move shortcut
	sess.ExecuteColonCommand("boguscmd")
	assert.True(t, sess.StatusIsErr)

	// Mouse interactions: wheel, right-click clear, drag-and-drop, tab click, scorebook click, list clicks
	sess.Flipped = false
	sess.ActiveTab = TabScore
	require.NoError(t, sess.Render(scr, geom, &out, true))
	r := sess.hitBoardRect
	assert.Equal(t, chess.NoSquare, sess.BoardSquareAtCell(-1, -1))

	// Wheel on board steps ply; wheel on non-score tab scrolls tab
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseCol: r.X + 2, MouseRow: r.Y + 2, MouseAction: MouseWheelUp})
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseCol: r.X + 2, MouseRow: r.Y + 2, MouseAction: MouseWheelDown})
	sess.ActiveTab = TabGames
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseCol: 100, MouseRow: 10, MouseAction: MouseWheelDown})
	sess.ActiveTab = TabScore

	// Drag Nf3 (g1 -> f3) and right-click clear
	sess.ResetNewGame("")
	g1Col := r.X + (6*r.W)/8 + 1
	g1Row := r.Y + (7*r.H)/8 + 1
	f3Col := r.X + (5*r.W)/8 + 1
	f3Row := r.Y + (5*r.H)/8 + 1
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: g1Col, MouseRow: g1Row, MouseAction: MousePress})
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 2, MouseCol: g1Col, MouseRow: g1Row, MouseAction: MousePress})
	assert.Equal(t, chess.NoSquare, sess.SelectedSq)
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: g1Col, MouseRow: g1Row, MouseAction: MousePress})
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: f3Col, MouseRow: f3Row, MouseAction: MouseRelease})
	assert.Equal(t, chess.WN, sess.Board.PieceAt(chess.F3))

	// Click empty square clears selection
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: r.X + 1, MouseRow: r.Y + (4*r.H)/8 + 1, MouseAction: MousePress})
	assert.Equal(t, chess.NoSquare, sess.SelectedSq)

	// Click scorebook ply token
	require.NoError(t, sess.Render(scr, geom, &out, false))
	if len(sess.hitScorePlies) > 0 {
		sp := sess.hitScorePlies[0]
		sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: sp.X0 + 1, MouseRow: sp.Row + 1, MouseAction: MousePress})
		assert.Equal(t, sp.Ply, sess.CurPly)
	}

	// Click tab header for Games, render, then click row twice (select + open)
	gamesTab := sess.hitTabRects[TabGames]
	sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: gamesTab.X + 1, MouseRow: gamesTab.Y + 1, MouseAction: MousePress})
	assert.Equal(t, TabGames, sess.ActiveTab)
	require.NoError(t, sess.Render(scr, geom, &out, false))
	if sess.hitListRowCount > 1 {
		rowY := sess.hitListStartRow + 1 + 1 // 1-based row for index 1
		sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: 80, MouseRow: rowY, MouseAction: MousePress})
		assert.Equal(t, 1, sess.GameListCursor)
		sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: 80, MouseRow: rowY, MouseAction: MousePress})
		assert.Equal(t, TabScore, sess.ActiveTab)
	}

	// Click Tree row
	sess.ActiveTab = TabTree
	sess.JumpToPly(0)
	sess.mu.Lock()
	sess.TreeResult = &tree.TreeResult{
		FEN:   sess.Board.FEN(),
		Moves: []*tree.MoveStat{{Move: "d4", UCI: "d2d4", Count: 5}},
	}
	sess.mu.Unlock()
	require.NoError(t, sess.Render(scr, geom, &out, false))
	if sess.hitListRowCount > 0 {
		sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: 80, MouseRow: sess.hitListStartRow + 1, MouseAction: MousePress})
	}

	// Click Bases row twice
	sess.ActiveTab = TabBases
	sess.BasesCursor = 0
	sess.DiscoveredBases = []scid.DiscoveredDatabase{
		{Name: "a", BasePath: si4Path, Type: "si4", Exists: true},
		{Name: "b", BasePath: pgnPath, Type: "pgn", Exists: true},
	}
	require.NoError(t, sess.Render(scr, geom, &out, false))
	if sess.hitListRowCount > 1 {
		rowY := sess.hitListStartRow + 2
		sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: 80, MouseRow: rowY, MouseAction: MousePress})
		assert.Equal(t, 1, sess.BasesCursor)
		sess.HandleEvent(InputEvent{Kind: KeyMouse, MouseBtn: 0, MouseCol: 80, MouseRow: rowY, MouseAction: MousePress})
	}

	// Quit via :quit and Ctrl+Q
	sess.ExecuteColonCommand("quit")
	assert.True(t, sess.ShouldQuit)
	sess.ShouldQuit = false
	sess.HandleEvent(InputEvent{Kind: KeyCtrlQ})
	assert.True(t, sess.ShouldQuit)
}
