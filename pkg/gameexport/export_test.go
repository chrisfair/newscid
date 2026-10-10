package gameexport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func createSampleGame() *chess.Game {
	g := chess.NewGame()
	g.SetTag("White", "Carlsen, Magnus")
	g.SetTag("Black", "Nakamura, Hikaru")
	g.SetTag("WhiteElo", "2850")
	g.SetTag("BlackElo", "2800")
	g.SetTag("Event", "World Blitz Championship & Trophy $100% win")
	g.SetTag("Site", "Almaty, KAZ")
	g.SetTag("Date", "2022.12.30")
	g.SetTag("Round", "12")
	g.SetTag("ECO", "B01")
	g.Result = chess.ResultWhiteWins

	// Build move tree: 1. e4 d5 (1... e5) 2. exd5 Qxd5 3. Nc3
	b := chess.NewBoard()

	m1, _ := b.ParseUCI("e2e4")
	b.MakeMove(m1)
	node1 := &chess.MoveNode{
		Move: m1,
		UCI:  "e2e4",
		SAN:  "e4",
		FEN:  b.FEN(),
		NAGs: []int{1}, // !
	}
	g.RootMove = node1

	m2, _ := b.ParseUCI("d7d5")
	b.MakeMove(m2)
	node2 := &chess.MoveNode{
		Move:    m2,
		UCI:     "d7d5",
		SAN:     "d5",
		FEN:     b.FEN(),
		Comment: "Scandinavian Defense",
	}
	node1.Next = node2

	// Variation: 1... e5
	bVar := chess.NewBoard()
	mVar1, _ := bVar.ParseUCI("e2e4")
	bVar.MakeMove(mVar1)
	mVar2, _ := bVar.ParseUCI("e7e5")
	bVar.MakeMove(mVar2)
	varNode := &chess.MoveNode{
		Move:    mVar2,
		UCI:     "e7e5",
		SAN:     "e5",
		FEN:     bVar.FEN(),
		Comment: "Classical e5",
	}
	node1.Variations = []*chess.MoveNode{varNode}

	m3, _ := b.ParseUCI("e4d5")
	b.MakeMove(m3)
	node3 := &chess.MoveNode{
		Move: m3,
		UCI:  "e4d5",
		SAN:  "exd5",
		FEN:  b.FEN(),
	}
	node2.Next = node3

	m4, _ := b.ParseUCI("d8d5")
	b.MakeMove(m4)
	node4 := &chess.MoveNode{
		Move: m4,
		UCI:  "d8d5",
		SAN:  "Qxd5",
		FEN:  b.FEN(),
	}
	node3.Next = node4

	m5, _ := b.ParseUCI("b1c3")
	b.MakeMove(m5)
	node5 := &chess.MoveNode{
		Move: m5,
		UCI:  "b1c3",
		SAN:  "Nc3",
		FEN:  b.FEN(),
	}
	node4.Next = node5

	return g
}

func TestExportGameHTML_Errors(t *testing.T) {
	_, err := ExportGameHTML(nil, HTMLExportOptions{})
	assert.Error(t, err)

	_, err = ExportGamesHTML([]*chess.Game{}, HTMLExportOptions{})
	assert.Error(t, err)
}

func TestExportGameHTML_Success(t *testing.T) {
	g := createSampleGame()

	opts := HTMLExportOptions{
		Theme:             "light",
		IncludeDiagram:    true,
		DiagramPly:        2,
		IncludeComments:   true,
		IncludeVariations: true,
	}

	htmlContent, err := ExportGameHTML(g, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, htmlContent)

	assert.Contains(t, htmlContent, "<!DOCTYPE html>")
	assert.Contains(t, htmlContent, "Carlsen, Magnus")
	assert.Contains(t, htmlContent, "Nakamura, Hikaru")
	assert.Contains(t, htmlContent, "2850")
	assert.Contains(t, htmlContent, "B01")
	assert.Contains(t, htmlContent, "1-0")
	assert.Contains(t, htmlContent, "e4")
	assert.Contains(t, htmlContent, "d5")
	assert.Contains(t, htmlContent, "Scandinavian Defense")
	assert.Contains(t, htmlContent, "Classical e5")
	assert.Contains(t, htmlContent, "<svg")
	assert.Contains(t, htmlContent, "chessboard-svg")
}

func TestExportGameHTML_DarkThemeAndFinalPosition(t *testing.T) {
	g := createSampleGame()

	opts := HTMLExportOptions{
		Title:          "Championship Clash",
		Theme:          "dark",
		IncludeDiagram: true,
		DiagramPly:     -1,
	}

	htmlContent, err := ExportGameHTML(g, opts)
	require.NoError(t, err)

	assert.Contains(t, htmlContent, "<title>Championship Clash</title>")
	assert.Contains(t, htmlContent, "Final Position (1-0)")
	assert.Contains(t, htmlContent, "#09090b")
}

func TestExportGameHTML_NoRootMove(t *testing.T) {
	g := chess.NewGame()
	g.SetTag("White", "Player 1")
	g.SetTag("Black", "Player 2")
	g.Result = chess.ResultDraw

	htmlContent, err := ExportGameHTML(g, HTMLExportOptions{})
	require.NoError(t, err)
	assert.Contains(t, htmlContent, "1/2-1/2")
}

func TestExportGamesHTML_Multiple(t *testing.T) {
	g1 := createSampleGame()
	g2 := createSampleGame()
	g2.SetTag("White", "Kasparov, Garry")
	g2.SetTag("Black", "Karpov, Anatoly")

	htmlContent, err := ExportGamesHTML([]*chess.Game{g1, g2}, HTMLExportOptions{})
	require.NoError(t, err)

	assert.Contains(t, htmlContent, "Chess Games Export (2 games)")
	assert.Contains(t, htmlContent, "game-1")
	assert.Contains(t, htmlContent, "game-2")
	assert.Contains(t, htmlContent, "Kasparov, Garry")
}

func TestExportGameLaTeX_Errors(t *testing.T) {
	_, err := ExportGameLaTeX(nil, LaTeXExportOptions{})
	assert.Error(t, err)

	_, err = ExportGamesLaTeX([]*chess.Game{}, LaTeXExportOptions{})
	assert.Error(t, err)
}

func TestExportGameLaTeX_Success(t *testing.T) {
	g := createSampleGame()

	opts := LaTeXExportOptions{
		DocumentClass:  "article",
		IncludeDiagram: true,
		DiagramPly:     3,
	}

	latexContent, err := ExportGameLaTeX(g, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, latexContent)

	assert.Contains(t, latexContent, "\\documentclass[11pt,a4paper]{article}")
	assert.Contains(t, latexContent, "\\usepackage{chessboard}")
	assert.Contains(t, latexContent, "Carlsen, Magnus")
	assert.Contains(t, latexContent, "Nakamura, Hikaru")
	// Special chars escaping check
	assert.Contains(t, latexContent, "\\$100\\% win")
	assert.Contains(t, latexContent, "\\textbf{1.} e4!")
	assert.Contains(t, latexContent, "\\chessboard[setfen=")
	assert.Contains(t, latexContent, "\\end{document}")
}

func TestExportGamesLaTeX_Multiple(t *testing.T) {
	g1 := createSampleGame()
	g2 := createSampleGame()
	g2.SetTag("White", "Spassky, Boris")
	g2.SetTag("Black", "Fischer, Robert J.")
	g2.Result = chess.ResultBlackWins

	opts := LaTeXExportOptions{
		IncludeDiagram: true,
		DiagramPly:     -1,
	}

	latexContent, err := ExportGamesLaTeX([]*chess.Game{g1, g2}, opts)
	require.NoError(t, err)

	assert.Contains(t, latexContent, "\\section*{Game 1:")
	assert.Contains(t, latexContent, "\\section*{Game 2:")
	assert.Contains(t, latexContent, "Spassky, Boris")
	assert.Contains(t, latexContent, "Fischer, Robert J.")
	assert.Contains(t, latexContent, "\\newpage")
}

func TestGenerateSVGChessboard(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	svg := GenerateSVGChessboard(fen, 320)
	assert.Contains(t, svg, "<svg")
	assert.Contains(t, svg, "width=\"320\" height=\"320\"")
	assert.Contains(t, svg, "&#9817;") // White pawn
	assert.Contains(t, svg, "&#9820;") // Black rook

	// Test default size <= 0
	svgDefault := GenerateSVGChessboard(fen, 0)
	assert.Contains(t, svgDefault, "width=\"280\"")
}

func TestGetDiagramFEN_EdgeCases(t *testing.T) {
	// Game with no moves
	gEmpty := chess.NewGame()
	fen := getDiagramFEN(gEmpty, 5)
	assert.Equal(t, chess.StartingFEN, fen)

	// Game with root move but ply exceeding length
	g := createSampleGame()
	fenBeyond := getDiagramFEN(g, 999)
	assert.NotEmpty(t, fenBeyond)
	assert.NotEqual(t, "", fenBeyond)
}
