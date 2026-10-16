package scid

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"newscid/pkg/chess"
)

var (
	scidClkRegex = regexp.MustCompile(`\[%clk\s+([^\]]+)\]`)
	scidEmtRegex = regexp.MustCompile(`\[%emt\s+([^\]]+)\]`)
)

// Special tokens in SCID move stream
const (
	EncodeNAG          = 11
	EncodeComment      = 12
	EncodeStartMarker  = 13
	EncodeEndMarker    = 14
	EncodeEndGame      = 15
)

var commonTagNames = map[byte]string{
	241: "WhiteCountry",
	242: "BlackCountry",
	243: "Annotator",
	244: "PlyCount",
	245: "EventDate",
	246: "Opening",
	247: "Variation",
	248: "Setup",
	249: "Source",
	250: "SetUp",
}

var commonTagIDs = map[string]byte{
	"WhiteCountry": 241,
	"BlackCountry": 242,
	"Annotator":    243,
	"PlyCount":     244,
	"EventDate":    245,
	"Opening":      246,
	"Variation":    247,
	"Setup":        248,
	"Source":       249,
	"SetUp":        250,
}

// DecodeGameData decodes a game record from .sg4 bytes given the index entry and namebase.
func DecodeGameData(raw []byte, entry *IndexEntry, nb *NameBase) (*chess.Game, error) {
	game := chess.NewGame()

	// Fill tags from index and namebase
	if nb != nil {
		game.SetTag("White", nb.GetName(NamePlayer, entry.WhiteID))
		game.SetTag("Black", nb.GetName(NamePlayer, entry.BlackID))
		game.SetTag("Event", nb.GetName(NameEvent, entry.EventID))
		game.SetTag("Site", nb.GetName(NameSite, entry.SiteID))
		game.SetTag("Round", nb.GetName(NameRound, entry.RoundID))
	}
	game.SetTag("Date", entry.Date)
	game.SetTag("Result", string(entry.Result))
	if entry.WhiteElo > 0 {
		game.SetTag("WhiteElo", fmt.Sprintf("%d", entry.WhiteElo))
	}
	if entry.BlackElo > 0 {
		game.SetTag("BlackElo", fmt.Sprintf("%d", entry.BlackElo))
	}

	r := bytes.NewReader(raw)

	// 1. Decode non-standard / extra tags
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0 {
			// End of tags
			break
		}

		var tagName string
		if b > 240 {
			// Common tag
			tagName = commonTagNames[b]
			if tagName == "" {
				tagName = fmt.Sprintf("Tag_%d", b)
			}
		} else {
			// Custom tag: length = b
			tagLen := int(b)
			tagBytes := make([]byte, tagLen)
			if _, err := io.ReadFull(r, tagBytes); err != nil {
				return nil, err
			}
			tagName = string(tagBytes)
		}

		valLenByte, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		valBytes := make([]byte, int(valLenByte))
		if _, err := io.ReadFull(r, valBytes); err != nil {
			return nil, err
		}
		game.SetTag(tagName, string(valBytes))
	}

	// 2. Decode Game Flags
	gflags, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	nonStandardStart := (gflags & 1) != 0

	// 3. Start position
	var startBoard *chess.Board
	if nonStandardStart {
		var fenBytes []byte
		for {
			fb, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if fb == 0 {
				break
			}
			fenBytes = append(fenBytes, fb)
		}
		fenStr := string(fenBytes)
		game.StartFEN = fenStr
		b, err := chess.ParseFEN(fenStr)
		if err != nil {
			return nil, fmt.Errorf("decoding start FEN %q: %w", fenStr, err)
		}
		startBoard = b
	} else {
		startBoard = chess.NewBoard()
		game.StartFEN = chess.StartingFEN
	}

	// 4. Decode moves
	var commentNodes []*chess.MoveNode
	rootMove, plyCount, err := decodeVariation(r, startBoard.Clone(), 0, &commentNodes)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// 5. Decode comments (null-terminated UTF-8 strings following moves)
	for _, node := range commentNodes {
		var commentBytes []byte
		for {
			cb, err := r.ReadByte()
			if err != nil {
				break
			}
			if cb == 0 {
				break
			}
			commentBytes = append(commentBytes, cb)
		}
		if len(commentBytes) > 0 {
			cStr := string(commentBytes)
			if node.Comment != "" {
				node.Comment += " " + cStr
			} else {
				node.Comment = cStr
			}
			if node.Clock == "" {
				if m := scidClkRegex.FindStringSubmatch(cStr); len(m) > 1 {
					node.Clock = strings.TrimSpace(m[1])
				}
			}
			if node.EMT == "" {
				if m := scidEmtRegex.FindStringSubmatch(cStr); len(m) > 1 {
					node.EMT = strings.TrimSpace(m[1])
				}
			}
		}
	}

	game.RootMove = rootMove
	game.PlyCount = plyCount
	return game, nil
}

func decodeVariation(r *bytes.Reader, b *chess.Board, level int, commentNodes *[]*chess.MoveNode) (*chess.MoveNode, int, error) {
	var root *chess.MoveNode
	var curr *chess.MoveNode
	var prevBoard *chess.Board
	plyCount := 0

	for {
		bVal, err := r.ReadByte()
		if err != nil {
			return root, plyCount, err
		}

		if bVal == EncodeEndGame || bVal == EncodeEndMarker {
			break
		}

		switch bVal {
		case EncodeStartMarker:
			// Variation starts: child of previous position before curr move
			if curr != nil && prevBoard != nil {
				varBoard := prevBoard.Clone()
				varNode, _, err := decodeVariation(r, varBoard, level+1, commentNodes)
				if err != nil && err != io.EOF {
					return root, plyCount, err
				}
				if varNode != nil {
					curr.Variations = append(curr.Variations, varNode)
				}
			}
		case EncodeNAG:
			nagByte, err := r.ReadByte()
			if err == nil && curr != nil {
				curr.NAGs = append(curr.NAGs, int(nagByte))
			}
		case EncodeComment:
			if curr != nil {
				*commentNodes = append(*commentNodes, curr)
			}
		default:
			prevBoard = b.Clone()
			move, err := decodePieceMove(r, b, bVal)
			if err != nil {
				return root, plyCount, err
			}

			san := b.MoveToSAN(move)
			b.MakeMove(move)
			plyCount++

			node := &chess.MoveNode{
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
	}

	return root, plyCount, nil
}

func decodePieceMove(r *bytes.Reader, b *chess.Board, val byte) (chess.Move, error) {
	pieceNum := int(val >> 4)
	code := val & 15

	if pieceNum >= b.PieceCount[b.Turn] {
		return chess.Move{}, fmt.Errorf("invalid pieceNum %d (pieces count %d)", pieceNum, b.PieceCount[b.Turn])
	}

	from := b.PieceList[b.Turn][pieceNum]
	movingPiece := b.Squares[from]
	if movingPiece == chess.Empty {
		return chess.Move{}, fmt.Errorf("empty square at from %s", from)
	}

	pt := movingPiece.Type()
	var to chess.Square
	promo := chess.None

	switch pt {
	case chess.King:
		sqdiffs := []int{0, -9, -8, -7, -1, 1, 7, 8, 9, -2, 2}
		if int(code) >= len(sqdiffs) {
			return chess.Move{}, fmt.Errorf("invalid king move code %d", code)
		}
		diff := sqdiffs[code]
		to = chess.Square(int(from) + diff)

	case chess.Knight:
		sqdiffs := []int{0, -17, -15, -10, -6, 6, 10, 15, 17}
		if int(code) >= len(sqdiffs) || code == 0 {
			return chess.Move{}, fmt.Errorf("invalid knight move code %d", code)
		}
		to = chess.Square(int(from) + sqdiffs[code])

	case chess.Rook:
		if code >= 8 {
			to = chess.MakeSquare(from.File(), int(code-8))
		} else {
			to = chess.MakeSquare(int(code), from.Rank())
		}

	case chess.Bishop:
		fyle := int(code & 7)
		fylediff := fyle - from.File()
		if code >= 8 {
			to = chess.Square(int(from) - 7*fylediff)
		} else {
			to = chess.Square(int(from) + 9*fylediff)
		}

	case chess.Queen:
		if code >= 8 {
			// Rook vertical
			to = chess.MakeSquare(from.File(), int(code-8))
		} else if int(code) != from.File() {
			// Rook horizontal
			to = chess.MakeSquare(int(code), from.Rank())
		} else {
			// Diagonal move (2-byte encoding)
			toByte, err := r.ReadByte()
			if err != nil {
				return chess.Move{}, err
			}
			to = chess.Square(toByte - 64)
		}

	case chess.Pawn:
		toSquareDiff := []int{7, 8, 9, 7, 8, 9, 7, 8, 9, 7, 8, 9, 7, 8, 9, 16}
		promoPieces := []chess.PieceType{
			chess.None, chess.None, chess.None,
			chess.Queen, chess.Queen, chess.Queen,
			chess.Rook, chess.Rook, chess.Rook,
			chess.Bishop, chess.Bishop, chess.Bishop,
			chess.Knight, chess.Knight, chess.Knight,
			chess.None,
		}
		diff := toSquareDiff[code]
		if b.Turn == chess.White {
			to = chess.Square(int(from) + diff)
		} else {
			to = chess.Square(int(from) - diff)
		}
		promo = promoPieces[code]

	default:
		return chess.Move{}, fmt.Errorf("unknown piece type %v", pt)
	}

	return chess.Move{From: from, To: to, Promo: promo}, nil
}

// EncodeGameData encodes a Game into .sg4 binary bytes.
func EncodeGameData(game *chess.Game) ([]byte, error) {
	var buf bytes.Buffer

	// 1. Tags
	for k, v := range game.Tags {
		// Skip index-stored tags
		if k == "White" || k == "Black" || k == "Event" || k == "Site" || k == "Round" || k == "Date" || k == "Result" || k == "WhiteElo" || k == "BlackElo" {
			continue
		}
		if code, ok := commonTagIDs[k]; ok {
			buf.WriteByte(code)
		} else {
			buf.WriteByte(byte(len(k)))
			buf.WriteString(k)
		}
		buf.WriteByte(byte(len(v)))
		buf.WriteString(v)
	}
	buf.WriteByte(0) // End of tags

	// 2. Flags & Start FEN
	isCustomFEN := game.StartFEN != "" && game.StartFEN != chess.StartingFEN
	var flags byte = 0
	if isCustomFEN {
		flags |= 1
	}
	buf.WriteByte(flags)

	var b *chess.Board
	if isCustomFEN {
		buf.WriteString(game.StartFEN)
		buf.WriteByte(0)
		parsed, err := chess.ParseFEN(game.StartFEN)
		if err != nil {
			return nil, err
		}
		b = parsed
	} else {
		b = chess.NewBoard()
	}

	// 3. Encode Moves
	var comments []string
	if game.RootMove != nil {
		encodeNodeTree(&buf, b, game.RootMove, &comments)
	}
	buf.WriteByte(EncodeEndGame)

	// 4. Encode Comments
	for _, c := range comments {
		buf.WriteString(c)
		buf.WriteByte(0)
	}

	return buf.Bytes(), nil
}

func encodeNodeTree(buf *bytes.Buffer, b *chess.Board, node *chess.MoveNode, comments *[]string) {
	curr := node
	for curr != nil {
		beforeBoard := b.Clone()

		// Encode Move
		encodeSingleMove(buf, b, curr.Move)
		b.MakeMove(curr.Move)

		// NAGs
		for _, nag := range curr.NAGs {
			buf.WriteByte(EncodeNAG)
			buf.WriteByte(byte(nag))
		}

		// Comment & Clock metadata
		fullComment := curr.Comment
		if curr.Clock != "" && !strings.Contains(fullComment, "%clk") {
			if fullComment != "" {
				fullComment += " "
			}
			fullComment += fmt.Sprintf("[%%clk %s]", curr.Clock)
		}
		if curr.EMT != "" && !strings.Contains(fullComment, "%emt") {
			if fullComment != "" {
				fullComment += " "
			}
			fullComment += fmt.Sprintf("[%%emt %s]", curr.EMT)
		}
		if fullComment != "" {
			buf.WriteByte(EncodeComment)
			*comments = append(*comments, fullComment)
		}

		// Variations
		for _, v := range curr.Variations {
			buf.WriteByte(EncodeStartMarker)
			varBoard := beforeBoard.Clone()
			encodeNodeTree(buf, varBoard, v, comments)
			buf.WriteByte(EncodeEndMarker)
		}

		curr = curr.Next
	}
}

func encodeSingleMove(buf *bytes.Buffer, b *chess.Board, m chess.Move) {
	pieceNum := b.PiecePos[m.From]
	pt := b.Squares[m.From].Type()
	var code byte = 0

	switch pt {
	case chess.King:
		diff := int(m.To) - int(m.From)
		sqdiffs := []int{0, -9, -8, -7, -1, 1, 7, 8, 9, -2, 2}
		for i, d := range sqdiffs {
			if d == diff {
				code = byte(i)
				break
			}
		}
		buf.WriteByte(byte((pieceNum << 4) | int(code)))

	case chess.Knight:
		diff := int(m.To) - int(m.From)
		sqdiffs := []int{0, -17, -15, -10, -6, 6, 10, 15, 17}
		for i, d := range sqdiffs {
			if d == diff {
				code = byte(i)
				break
			}
		}
		buf.WriteByte(byte((pieceNum << 4) | int(code)))

	case chess.Rook:
		if m.From.Rank() == m.To.Rank() {
			code = byte(m.To.File())
		} else {
			code = byte(8 + m.To.Rank())
		}
		buf.WriteByte(byte((pieceNum << 4) | int(code)))

	case chess.Bishop:
		fylediff := m.To.File() - m.From.File()
		rankdiff := m.To.Rank() - m.From.Rank()
		code = byte(m.To.File())
		if rankdiff*fylediff < 0 {
			code += 8
		}
		buf.WriteByte(byte((pieceNum << 4) | int(code)))

	case chess.Queen:
		if m.From.Rank() == m.To.Rank() {
			code = byte(m.To.File())
			buf.WriteByte(byte((pieceNum << 4) | int(code)))
		} else if m.From.File() == m.To.File() {
			code = byte(8 + m.To.Rank())
			buf.WriteByte(byte((pieceNum << 4) | int(code)))
		} else {
			// Diagonal: 2 bytes
			code = byte(m.From.File())
			buf.WriteByte(byte((pieceNum << 4) | int(code)))
			buf.WriteByte(byte(m.To + 64))
		}

	case chess.Pawn:
		diff := int(m.To) - int(m.From)
		if diff < 0 {
			diff = -diff
		}
		if diff == 16 {
			code = 15
		} else {
			if diff == 7 {
				code = 0
			} else if diff == 8 {
				code = 1
			} else {
				code = 2
			}
			if m.Promo != chess.None {
				promoOffset := 3 * int(m.Promo-chess.Queen+1)
				code += byte(promoOffset)
			}
		}
		buf.WriteByte(byte((pieceNum << 4) | int(code)))
	}
}

// ReadGameBytes reads game data bytes from an open .sg4 file.
func ReadGameBytes(filename string, offset, length uint32) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, err
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
