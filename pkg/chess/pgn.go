package chess

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// PGNGame represents a game parsed from or exported to PGN.
func (g *Game) ExportPGN() string {
	var sb strings.Builder

	// Write standard tags in standard order
	standardTags := []string{"Event", "Site", "Date", "Round", "White", "Black", "Result"}
	for _, tag := range standardTags {
		sb.WriteString(fmt.Sprintf("[%s \"%s\"]\n", tag, g.Tag(tag)))
	}

	// Write extra tags
	for k, v := range g.Tags {
		isStd := false
		for _, st := range standardTags {
			if k == st {
				isStd = true
				break
			}
		}
		if !isStd {
			sb.WriteString(fmt.Sprintf("[%s \"%s\"]\n", k, v))
		}
	}
	sb.WriteByte('\n')

	// Write moves
	if g.RootMove != nil {
		writeMoveNodeTree(&sb, g.RootMove, 1, true)
	}

	sb.WriteByte(' ')
	sb.WriteString(string(g.Result))
	sb.WriteByte('\n')

	return sb.String()
}

func writeMoveNodeTree(sb *strings.Builder, node *MoveNode, moveNum int, isWhite bool) {
	curr := node
	first := true

	for curr != nil {
		if !first {
			sb.WriteByte(' ')
		}
		first = false

		if isWhite {
			sb.WriteString(fmt.Sprintf("%d. ", moveNum))
		} else if first || curr == node {
			sb.WriteString(fmt.Sprintf("%d... ", moveNum))
		}

		sb.WriteString(curr.SAN)

		// NAGs
		for _, nag := range curr.NAGs {
			sb.WriteString(fmt.Sprintf(" %s", FormatNAG(nag)))
		}

		// Comments
		if curr.Comment != "" {
			sb.WriteString(fmt.Sprintf(" {%s}", curr.Comment))
		}

		// Variations
		for _, v := range curr.Variations {
			sb.WriteString(" (")
			writeMoveNodeTree(sb, v, moveNum, isWhite)
			sb.WriteString(")")
		}

		if !isWhite {
			moveNum++
		}
		isWhite = !isWhite
		curr = curr.Next
	}
}

// ParsePGN parses a single game from a PGN string.
func ParsePGN(pgn string) (*Game, error) {
	games, err := ParsePGNDatabase(strings.NewReader(pgn))
	if err != nil {
		return nil, err
	}
	if len(games) == 0 {
		return nil, fmt.Errorf("no games found in PGN")
	}
	return games[0], nil
}

// ParsePGNDatabase parses all games from a PGN reader stream.
func ParsePGNDatabase(r io.Reader) ([]*Game, error) {
	scanner := bufio.NewScanner(r)
	var games []*Game
	var currentGame *Game
	var moveText strings.Builder
	inHeader := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if inHeader && currentGame != nil && len(currentGame.Tags) > 0 {
				inHeader = false
			}
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Tag line
			if !inHeader && currentGame != nil {
				// Finish previous game
				parseMoveText(currentGame, moveText.String())
				games = append(games, currentGame)
				currentGame = nil
				moveText.Reset()
				inHeader = true
			}

			if currentGame == nil {
				currentGame = NewGame()
			}

			content := line[1 : len(line)-1]
			firstSpace := strings.IndexByte(content, ' ')
			if firstSpace != -1 {
				key := strings.TrimSpace(content[:firstSpace])
				val := strings.Trim(strings.TrimSpace(content[firstSpace:]), "\"")
				currentGame.SetTag(key, val)
			}
		} else {
			// Move text line
			inHeader = false
			if currentGame == nil {
				currentGame = NewGame()
			}
			moveText.WriteString(line)
			moveText.WriteByte(' ')
		}
	}

	if currentGame != nil {
		parseMoveText(currentGame, moveText.String())
		games = append(games, currentGame)
	}

	return games, nil
}

func parseMoveText(game *Game, text string) {
	var board *Board
	if fen, ok := game.Tags["FEN"]; ok && fen != "" {
		b, err := ParseFEN(fen)
		if err == nil {
			board = b
			game.StartFEN = fen
		}
	}
	if board == nil {
		board = NewBoard()
		game.StartFEN = StartingFEN
	}

	tokens := tokenizePGN(text)
	root, ply := parseTokensToMoveTree(tokens, board)
	game.RootMove = root
	game.PlyCount = ply

	// Check if game has Result token in text if not set in header
	for i := len(tokens) - 1; i >= 0; i-- {
		t := tokens[i]
		if t == "1-0" || t == "0-1" || t == "1/2-1/2" || t == "*" {
			game.Result = Result(t)
			game.SetTag("Result", t)
			break
		}
	}
}

func tokenizePGN(text string) []string {
	var tokens []string
	var curr strings.Builder
	inComment := false

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if inComment {
			if ch == '}' {
				inComment = false
				tokens = append(tokens, "{"+curr.String()+"}")
				curr.Reset()
			} else {
				curr.WriteRune(ch)
			}
			continue
		}

		if ch == '{' {
			if curr.Len() > 0 {
				tokens = append(tokens, curr.String())
				curr.Reset()
			}
			inComment = true
			continue
		}

		if ch == '(' || ch == ')' {
			if curr.Len() > 0 {
				tokens = append(tokens, curr.String())
				curr.Reset()
			}
			tokens = append(tokens, string(ch))
			continue
		}

		if unicode.IsSpace(ch) {
			if curr.Len() > 0 {
				tokens = append(tokens, curr.String())
				curr.Reset()
			}
			continue
		}

		curr.WriteRune(ch)
	}

	if curr.Len() > 0 {
		tokens = append(tokens, curr.String())
	}

	return tokens
}

func parseTokensToMoveTree(tokens []string, initialBoard *Board) (*MoveNode, int) {
	b := initialBoard.Clone()
	var root *MoveNode
	var curr *MoveNode
	plyCount := 0

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		// Skip comments and attach to previous move
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			comment := strings.Trim(t, "{}")
			if curr != nil {
				curr.Comment = comment
			}
			continue
		}

		// Handle NAGs
		if strings.HasPrefix(t, "$") {
			nag, err := strconv.Atoi(t[1:])
			if err == nil && curr != nil {
				curr.NAGs = append(curr.NAGs, nag)
			}
			continue
		}

		// Skip move numbers like "1." or "1..."
		if strings.Contains(t, ".") {
			parts := strings.Split(t, ".")
			t = parts[len(parts)-1]
			if t == "" {
				continue
			}
		}

		// Skip game terminations
		if t == "1-0" || t == "0-1" || t == "1/2-1/2" || t == "*" {
			break
		}

		// Try to parse SAN move
		move, err := b.ParseSAN(t)
		if err != nil {
			continue
		}

		san := b.MoveToSAN(move)
		b.MakeMove(move)
		plyCount++

		node := &MoveNode{
			Move: move,
			SAN:  san,
			UCI:  move.String(),
			FEN:  b.FEN(),
		}

		if root == nil {
			root = node
			curr = node
		} else {
			curr.Next = node
			curr = node
		}
	}

	return root, plyCount
}
