package gameexport

import (
	"fmt"
	"html"
	"strings"

	"newscid/pkg/chess"
)

// HTMLExportOptions defines customization settings for HTML game export.
type HTMLExportOptions struct {
	Title             string `json:"title,omitempty"`
	Theme             string `json:"theme,omitempty"` // "dark", "light", "classic"
	IncludeDiagram    bool   `json:"includeDiagram"`
	DiagramPly        int    `json:"diagramPly"` // -1 for final position
	IncludeComments   bool   `json:"includeComments"`
	IncludeVariations bool   `json:"includeVariations"`
}

// LaTeXExportOptions defines customization settings for LaTeX game export.
type LaTeXExportOptions struct {
	DocumentClass  string `json:"documentClass,omitempty"` // default "article"
	Package        string `json:"package,omitempty"`       // "chessboard" or "skak"
	IncludeDiagram bool   `json:"includeDiagram"`
	DiagramPly     int    `json:"diagramPly"`
}

// ExportGameHTML exports a chess game into a self-contained, beautifully styled HTML document.
func ExportGameHTML(g *chess.Game, opts HTMLExportOptions) (string, error) {
	if g == nil {
		return "", fmt.Errorf("game cannot be nil")
	}

	return ExportGamesHTML([]*chess.Game{g}, opts)
}

// ExportGamesHTML exports multiple games into a consolidated HTML document.
func ExportGamesHTML(games []*chess.Game, opts HTMLExportOptions) (string, error) {
	if len(games) == 0 {
		return "", fmt.Errorf("no games to export")
	}

	theme := opts.Theme
	if theme == "" {
		theme = "dark"
	}

	title := opts.Title
	if title == "" {
		if len(games) == 1 {
			title = fmt.Sprintf("%s vs %s - %s", games[0].Tag("White"), games[0].Tag("Black"), games[0].Tag("Event"))
		} else {
			title = fmt.Sprintf("Chess Games Export (%d games)", len(games))
		}
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString(getHTMLStyles(theme))
	sb.WriteString("</head>\n<body>\n")
	sb.WriteString("  <div class=\"container\">\n")
	sb.WriteString(fmt.Sprintf("    <header class=\"page-header\"><h1>%s</h1></header>\n", html.EscapeString(title)))

	for idx, g := range games {
		sb.WriteString(fmt.Sprintf("    <article class=\"game-card\" id=\"game-%d\">\n", idx+1))

		// Header block
		white := g.Tag("White")
		black := g.Tag("Black")
		wElo := g.Tag("WhiteElo")
		bElo := g.Tag("BlackElo")
		event := g.Tag("Event")
		site := g.Tag("Site")
		date := g.Tag("Date")
		round := g.Tag("Round")
		eco := g.Tag("ECO")
		result := string(g.Result)
		if result == "" {
			result = g.Tag("Result")
		}

		sb.WriteString("      <div class=\"game-header\">\n")
		sb.WriteString("        <div class=\"players\">\n")
		sb.WriteString(fmt.Sprintf("          <div class=\"player white\"><span class=\"piece-icon\">&#9817;</span> <span class=\"name\">%s</span> %s</div>\n",
			html.EscapeString(white), formatEloBadge(wElo)))
		sb.WriteString(fmt.Sprintf("          <div class=\"player black\"><span class=\"piece-icon\">&#9823;</span> <span class=\"name\">%s</span> %s</div>\n",
			html.EscapeString(black), formatEloBadge(bElo)))
		sb.WriteString("        </div>\n")

		sb.WriteString("        <div class=\"meta-tags\">\n")
		if result != "" {
			sb.WriteString(fmt.Sprintf("          <span class=\"badge result-badge\">%s</span>\n", html.EscapeString(result)))
		}
		if eco != "" {
			sb.WriteString(fmt.Sprintf("          <span class=\"badge eco-badge\">%s</span>\n", html.EscapeString(eco)))
		}
		if event != "" {
			sb.WriteString(fmt.Sprintf("          <span class=\"meta-item\"><strong>Event:</strong> %s</span>\n", html.EscapeString(event)))
		}
		if site != "" {
			sb.WriteString(fmt.Sprintf("          <span class=\"meta-item\"><strong>Site:</strong> %s</span>\n", html.EscapeString(site)))
		}
		if date != "" {
			sb.WriteString(fmt.Sprintf("          <span class=\"meta-item\"><strong>Date:</strong> %s</span>\n", html.EscapeString(date)))
		}
		if round != "" && round != "?" {
			sb.WriteString(fmt.Sprintf("          <span class=\"meta-item\"><strong>Round:</strong> %s</span>\n", html.EscapeString(round)))
		}
		sb.WriteString("        </div>\n")
		sb.WriteString("      </div>\n")

		// Game Content (Moves + Diagram)
		sb.WriteString("      <div class=\"game-body\">\n")
		sb.WriteString("        <div class=\"moves-text\">\n")
		sb.WriteString(formatHTMLMoves(g, opts))
		sb.WriteString("        </div>\n")

		if opts.IncludeDiagram {
			targetFEN := getDiagramFEN(g, opts.DiagramPly)
			sb.WriteString("        <div class=\"diagram-container\">\n")
			sb.WriteString(GenerateSVGChessboard(targetFEN, 280))
			sb.WriteString(fmt.Sprintf("          <div class=\"diagram-caption\">%s</div>\n",
				html.EscapeString(getDiagramCaption(g, opts.DiagramPly))))
			sb.WriteString("        </div>\n")
		}

		sb.WriteString("      </div>\n")
		sb.WriteString("    </article>\n")
	}

	sb.WriteString("    <footer class=\"page-footer\">Exported by NewScid - Professional Chess Database Suite</footer>\n")
	sb.WriteString("  </div>\n</body>\n</html>\n")
	return sb.String(), nil
}

func formatEloBadge(elo string) string {
	if elo == "" || elo == "0" || elo == "?" {
		return ""
	}
	return fmt.Sprintf("<span class=\"elo-badge\">(%s)</span>", html.EscapeString(elo))
}

func formatHTMLMoves(g *chess.Game, opts HTMLExportOptions) string {
	if g.RootMove == nil {
		return fmt.Sprintf("<p class=\"result\">%s</p>", html.EscapeString(string(g.Result)))
	}

	var sb strings.Builder
	sb.WriteString("<p class=\"moves-flow\">")
	writeHTMLNodeTree(&sb, g.RootMove, 1, true, opts)
	sb.WriteString(fmt.Sprintf(" <strong class=\"result-tag\">%s</strong>", html.EscapeString(string(g.Result))))
	sb.WriteString("</p>")
	return sb.String()
}

func writeHTMLNodeTree(sb *strings.Builder, node *chess.MoveNode, moveNum int, isWhite bool, opts HTMLExportOptions) {
	curr := node
	for curr != nil {
		if isWhite {
			sb.WriteString(fmt.Sprintf("<span class=\"move-number\">%d.</span> ", moveNum))
		} else if curr == node {
			sb.WriteString(fmt.Sprintf("<span class=\"move-number\">%d...</span> ", moveNum))
		}

		san := curr.SAN
		if san == "" {
			san = curr.Move.String()
		}
		sb.WriteString(fmt.Sprintf("<span class=\"move-san\">%s</span>", html.EscapeString(san)))

		// Render NAGs if any
		for _, nag := range curr.NAGs {
			symbol := chess.FormatNAG(nag)
			if symbol != "" {
				sb.WriteString(fmt.Sprintf("<span class=\"nag-symbol\">%s</span>", html.EscapeString(symbol)))
			}
		}

		// Render Comment
		if opts.IncludeComments && curr.Comment != "" {
			sb.WriteString(fmt.Sprintf(" <span class=\"move-comment\">{%s}</span>", html.EscapeString(curr.Comment)))
		}

		// Render Variations
		if opts.IncludeVariations && len(curr.Variations) > 0 {
			for _, varNode := range curr.Variations {
				sb.WriteString(" <span class=\"variation\">(")
				writeHTMLNodeTree(sb, varNode, moveNum, isWhite, opts)
				sb.WriteString(")</span>")
			}
		}

		sb.WriteString(" ")
		if isWhite {
			isWhite = false
		} else {
			isWhite = true
			moveNum++
		}
		curr = curr.Next
	}
}

func getDiagramFEN(g *chess.Game, ply int) string {
	if g.RootMove == nil {
		return chess.StartingFEN
	}

	// Default: final position
	if ply < 0 {
		curr := g.RootMove
		for curr.Next != nil {
			curr = curr.Next
		}
		if curr.FEN != "" {
			return curr.FEN
		}
		return chess.StartingFEN
	}

	// Move forward to ply
	currPly := 1
	curr := g.RootMove
	for curr != nil {
		if currPly == ply && curr.FEN != "" {
			return curr.FEN
		}
		if curr.Next == nil {
			if curr.FEN != "" {
				return curr.FEN
			}
			break
		}
		curr = curr.Next
		currPly++
	}
	return chess.StartingFEN
}

func getDiagramCaption(g *chess.Game, ply int) string {
	if ply < 0 {
		return fmt.Sprintf("Final Position (%s)", g.Result)
	}
	return fmt.Sprintf("Position after move %d", (ply+1)/2)
}

// GenerateSVGChessboard renders a clean vector SVG chessboard diagram directly from FEN.
func GenerateSVGChessboard(fen string, size int) string {
	if size <= 0 {
		size = 280
	}
	sqSize := size / 8

	parts := strings.Split(fen, " ")
	placement := parts[0]
	ranks := strings.Split(placement, "/")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" class=\"chessboard-svg\">\n", size, size, size, size))

	// Draw squares
	for r := 0; r < 8; r++ {
		for f := 0; f < 8; f++ {
			x := f * sqSize
			y := r * sqSize
			isLight := (r+f)%2 == 0
			fill := "#f0d9b5"
			if !isLight {
				fill = "#b58863"
			}
			sb.WriteString(fmt.Sprintf("  <rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" fill=\"%s\" />\n", x, y, sqSize, sqSize, fill))
		}
	}

	// Piece unicode symbols map
	pieceUnicode := map[rune]string{
		'K': "&#9812;", 'Q': "&#9813;", 'R': "&#9814;", 'B': "&#9815;", 'N': "&#9816;", 'P': "&#9817;",
		'k': "&#9818;", 'q': "&#9819;", 'r': "&#9820;", 'b': "&#9821;", 'n': "&#9822;", 'p': "&#9823;",
	}

	fontSize := int(float64(sqSize) * 0.8)

	// Draw pieces
	for r := 0; r < 8 && r < len(ranks); r++ {
		f := 0
		for _, ch := range ranks[r] {
			if ch >= '1' && ch <= '8' {
				f += int(ch - '0')
			} else if symbol, ok := pieceUnicode[ch]; ok {
				x := f*sqSize + sqSize/2
				y := r*sqSize + int(float64(sqSize)*0.78)
				color := "#18181b"
				if ch >= 'A' && ch <= 'Z' {
					color = "#ffffff"
				}
				sb.WriteString(fmt.Sprintf("  <text x=\"%d\" y=\"%d\" font-size=\"%d\" text-anchor=\"middle\" fill=\"%s\" stroke=\"#000\" stroke-width=\"0.6\" class=\"chess-piece\">%s</text>\n",
					x, y, fontSize, color, symbol))
				f++
			}
		}
	}

	sb.WriteString("</svg>\n")
	return sb.String()
}

func getHTMLStyles(theme string) string {
	isDark := theme == "dark"
	bg := "#ffffff"
	cardBg := "#f8fafc"
	text := "#0f172a"
	border := "#e2e8f0"
	badgeBg := "#f1f5f9"
	commentColor := "#0284c7"
	varColor := "#64748b"

	if isDark {
		bg = "#09090b"
		cardBg = "#18181b"
		text = "#f4f4f5"
		border = "#27272a"
		badgeBg = "#27272a"
		commentColor = "#38bdf8"
		varColor = "#a1a1aa"
	}

	return fmt.Sprintf(`  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: %s; color: %s; margin: 0; padding: 20px; line-height: 1.6; }
    .container { max-width: 900px; margin: 0 auto; }
    .page-header h1 { font-size: 24px; font-weight: 700; margin-bottom: 24px; border-bottom: 1px solid %s; padding-bottom: 12px; }
    .game-card { background: %s; border: 1px solid %s; border-radius: 12px; padding: 24px; margin-bottom: 24px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1); }
    .game-header { border-bottom: 1px solid %s; padding-bottom: 16px; margin-bottom: 16px; }
    .players { display: flex; flex-direction: column; gap: 6px; font-size: 16px; font-weight: 600; margin-bottom: 10px; }
    .player.white .piece-icon { color: #e2e8f0; text-shadow: 0 0 2px #000; }
    .player.black .piece-icon { color: #000; }
    .elo-badge { font-size: 12px; font-weight: normal; color: %s; margin-left: 6px; }
    .meta-tags { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; font-size: 12px; }
    .badge { display: inline-block; padding: 2px 8px; border-radius: 6px; font-weight: 700; background: %s; border: 1px solid %s; }
    .result-badge { background: #10b98120; border-color: #10b98140; color: #10b981; }
    .eco-badge { background: #06b6d420; border-color: #06b6d440; color: #06b6d4; font-family: monospace; }
    .meta-item { color: %s; }
    .game-body { display: flex; gap: 24px; }
    .moves-text { flex: 1; font-size: 14px; line-height: 1.8; }
    .moves-flow { margin: 0; }
    .move-number { font-weight: 700; color: %s; margin-right: 2px; }
    .move-san { font-weight: 500; cursor: pointer; }
    .nag-symbol { font-weight: 700; color: #f59e0b; margin-left: 1px; }
    .move-comment { color: %s; font-style: italic; }
    .variation { color: %s; font-size: 13px; }
    .result-tag { font-weight: 700; margin-left: 6px; }
    .diagram-container { display: flex; flex-direction: column; align-items: center; shrink: 0; }
    .chessboard-svg { border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.2); }
    .diagram-caption { font-size: 11px; color: %s; margin-top: 8px; text-align: center; }
    .page-footer { text-align: center; font-size: 12px; color: %s; margin-top: 32px; border-top: 1px solid %s; padding-top: 16px; }
    @media (max-width: 768px) {
      .game-body { flex-direction: column-reverse; align-items: center; }
    }
  </style>`,
		bg, text, border, cardBg, border, border, varColor, badgeBg, border, varColor, varColor, commentColor, varColor, varColor, varColor, border)
}

// ExportGameLaTeX exports a chess game into standard LaTeX format using chessboard or skak package.
func ExportGameLaTeX(g *chess.Game, opts LaTeXExportOptions) (string, error) {
	if g == nil {
		return "", fmt.Errorf("game cannot be nil")
	}

	return ExportGamesLaTeX([]*chess.Game{g}, opts)
}

// ExportGamesLaTeX exports multiple games into a single compilable LaTeX document.
func ExportGamesLaTeX(games []*chess.Game, opts LaTeXExportOptions) (string, error) {
	if len(games) == 0 {
		return "", fmt.Errorf("no games to export")
	}

	docClass := opts.DocumentClass
	if docClass == "" {
		docClass = "article"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\\documentclass[11pt,a4paper]{%s}\n", docClass))
	sb.WriteString("\\usepackage[utf8]{inputenc}\n")
	sb.WriteString("\\usepackage[margin=1in]{geometry}\n")
	sb.WriteString("\\usepackage{chessboard}\n")
	sb.WriteString("\\usepackage{xskak}\n")
	sb.WriteString("\\usepackage{hyperref}\n\n")

	title := "Chess Games"
	if len(games) == 1 {
		title = fmt.Sprintf("%s vs. %s", latexEscape(games[0].Tag("White")), latexEscape(games[0].Tag("Black")))
	}
	sb.WriteString(fmt.Sprintf("\\title{%s}\n", title))
	sb.WriteString("\\author{NewScid Export}\n")
	sb.WriteString("\\date{\\today}\n\n")
	sb.WriteString("\\begin{document}\n")
	sb.WriteString("\\maketitle\n\n")

	for idx, g := range games {
		white := latexEscape(g.Tag("White"))
		black := latexEscape(g.Tag("Black"))
		event := latexEscape(g.Tag("Event"))
		date := latexEscape(g.Tag("Date"))
		site := latexEscape(g.Tag("Site"))
		result := latexEscape(string(g.Result))
		if result == "" {
			result = latexEscape(g.Tag("Result"))
		}
		eco := latexEscape(g.Tag("ECO"))

		sb.WriteString(fmt.Sprintf("\\section*{Game %d: %s -- %s}\n", idx+1, white, black))
		sb.WriteString("\\begin{tabular}{ll}\n")
		sb.WriteString(fmt.Sprintf("  \\textbf{Event:} %s & \\textbf{Site:} %s \\\\\n", event, site))
		sb.WriteString(fmt.Sprintf("  \\textbf{Date:} %s & \\textbf{Result:} %s \\\\\n", date, result))
		if eco != "" {
			sb.WriteString(fmt.Sprintf("  \\textbf{ECO:} %s & \\\\\n", eco))
		}
		sb.WriteString("\\end{tabular}\n\n\\vspace{1em}\n\n")

		// Moves in standard LaTeX format
		sb.WriteString("\\noindent\n")
		writeLaTeXMoves(&sb, g)
		sb.WriteString(fmt.Sprintf(" \\textbf{%s}\n\n", result))

		if opts.IncludeDiagram {
			targetFEN := getDiagramFEN(g, opts.DiagramPly)
			sb.WriteString("\\begin{center}\n")
			sb.WriteString(fmt.Sprintf("  \\chessboard[setfen=%s,showmover=false]\n", targetFEN))
			sb.WriteString(fmt.Sprintf("  \\captionof{figure}{%s}\n", latexEscape(getDiagramCaption(g, opts.DiagramPly))))
			sb.WriteString("\\end{center}\n\n")
		}

		if idx < len(games)-1 {
			sb.WriteString("\\newpage\n\n")
		}
	}

	sb.WriteString("\\end{document}\n")
	return sb.String(), nil
}

func writeLaTeXMoves(sb *strings.Builder, g *chess.Game) {
	if g.RootMove == nil {
		return
	}
	curr := g.RootMove
	moveNum := 1
	isWhite := true

	for curr != nil {
		if isWhite {
			sb.WriteString(fmt.Sprintf("\\textbf{%d.} ", moveNum))
		}
		san := curr.SAN
		if san == "" {
			san = curr.Move.String()
		}
		sb.WriteString(latexEscape(san))

		// Append NAGs
		for _, nag := range curr.NAGs {
			sym := chess.FormatNAG(nag)
			if sym != "" {
				sb.WriteString(sym)
			}
		}

		// Comment
		if curr.Comment != "" {
			sb.WriteString(fmt.Sprintf(" \\emph{%s} ", latexEscape(curr.Comment)))
		}

		sb.WriteString(" ")
		if isWhite {
			isWhite = false
		} else {
			isWhite = true
			moveNum++
		}
		curr = curr.Next
	}
}

func latexEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\textbackslash ")
	s = strings.ReplaceAll(s, "&", "\\&")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "$", "\\$")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	return s
}
