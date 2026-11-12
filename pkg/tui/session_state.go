package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/eco"
	"newscid/pkg/engine"
	"newscid/pkg/fairplay"
	"newscid/pkg/fileformat"
	"newscid/pkg/imbalance"
	"newscid/pkg/scout"
	"newscid/pkg/sets"
	"newscid/pkg/tree"
)

// TabID identifies the active right-hand workstation inspector tab.
type TabID int

const (
	TabScore TabID = iota
	TabTree
	TabGames
	TabBases
	TabImbalance
	TabForensics
	tabCount = 6
)

var tabTitles = [tabCount]string{
	"1:Score",
	"2:Tree",
	"3:Games",
	"4:Bases",
	"5:Imbalance",
	"6:Forensics",
}

// InputMode represents whether keystrokes control the board/UI or edit the prompt line.
type InputMode uint8

const (
	ModeNormal InputMode = iota
	ModeMoveInput
	ModeSearchFilter
	ModeCommand
	ModeHelpModal
)

// PlyFrame records one step along the active game path (supporting mainline and variations).
type PlyFrame struct {
	Node      *chess.MoveNode
	ParentNext *chess.MoveNode
	BoardFEN  string
	Move      chess.Move
	SAN       string
	Ply       int
}

// SessionConfig configures startup options for a NewScid TUI workstation session.
type SessionConfig struct {
	DBPath      string
	PGNPath     string
	InitialFEN  string
	GameNumber  int
	EnginePath  string
	AutoEngine  bool
	UseKitty    bool
	ThemeID     string
	PieceSetID  string
	Flipped     bool
}

// Session manages all chess state, database queries, async UCI engine, and opening tree workers.
type Session struct {
	mu sync.Mutex

	// Visual & protocol preferences
	UseKitty      bool
	KittyDetected bool
	ThemeID       string
	PieceSetID    string
	Notation      chess.NotationStyle
	Flipped       bool
	ShowArrows    bool
	ActiveTab     TabID
	Mode          InputMode
	InputBuffer   string
	StatusMsg     string
	StatusIsErr   bool
	ShouldQuit    bool

	// Chess game & board state
	Game       *chess.Game
	Board      *chess.Board
	Path       []PlyFrame // Path[0..CurPly-1] are moves played from Game.StartFEN
	CurPly     int        // 0 = at StartFEN, len(Path) = at end of current line
	SelectedSq chess.Square
	DragFromSq chess.Square
	ECOClassifier *eco.Classifier
	CurrentECO    *eco.Entry
	ImbalanceRep  *imbalance.Report

	// Database / PGN state
	ScidDB          *scid.Database
	ActiveDBPath    string
	ActiveDBName    string
	IsPGNSource     bool
	PGNGames        []*chess.Game
	PGNSummaries    []*scid.GameSummary
	ActiveGameIndex int // 1-based game number in DB or PGN

	FilterQuery     string
	FilterCriteria  scid.SearchCriteria
	GameSummaries   []*scid.GameSummary
	TotalMatched    int
	GameListCursor  int
	GamePageSize    int
	DiscoveredBases []scid.DiscoveredDatabase
	BasesCursor     int

	// Opening Tree state
	TreeResult *tree.TreeResult
	TreeCursor int
	TreeBusy   bool
	treeSeq    uint64

	// UCI Engine state
	Engine         *engine.Engine
	EnginePath     string
	EngineName     string
	EngineEnabled  bool
	MultiPV        int
	LastEval       engine.EvalInfo
	EvalPositionFEN string

	// Forensics & Scout state
	ForensicsMode   string // "fairplay" or "scout"
	ForensicsTarget string
	FairPlayReport  *fairplay.PlayerFairPlayDossier
	ScoutReport     *scout.Report
	ForensicsScroll int
	ForensicsBusy   bool

	// Async wakeup channel for UI redraws when background engine/tree/forensics complete
	RedrawCh chan struct{}

	// Cached Kitty PNG payload
	lastBoardHash uint64
	lastBoardPNG  []byte
	lastBoardRect Rect
	kittyPlaced   bool

	// Hit-test rectangles updated during layout render
	hitBoardRect     Rect
	hitBoardIsKitty  bool
	hitTabRects      [tabCount]Rect
	hitScorePlies    []scoreHitSpan
	hitListStartRow  int
	hitListRowCount  int
}

type scoreHitSpan struct {
	Row int
	X0  int
	X1  int
	Ply int
}

// NewSession initializes a NewScid TUI session from cfg.
func NewSession(cfg SessionConfig) *Session {
	themeID := cfg.ThemeID
	if themeID == "" {
		themeID = "tournament-green"
	}
	pieceID := cfg.PieceSetID
	if pieceID == "" {
		pieceID = "cburnett"
	}

	ecoCls := eco.NewClassifier()
	if _, err := os.Stat(fileformat.SystemScidECOPath); err == nil {
		_ = ecoCls.LoadFromFile(fileformat.SystemScidECOPath)
	}

	s := &Session{
		UseKitty:       cfg.UseKitty,
		KittyDetected:  DetectKittySupport(),
		ThemeID:        themeID,
		PieceSetID:     pieceID,
		Notation:       chess.NotationSAN,
		Flipped:        cfg.Flipped,
		ShowArrows:     true,
		ActiveTab:      TabScore,
		Mode:           ModeNormal,
		SelectedSq:     chess.NoSquare,
		DragFromSq:     chess.NoSquare,
		ECOClassifier:  ecoCls,
		GamePageSize:   200,
		MultiPV:        2,
		EnginePath:     cfg.EnginePath,
		RedrawCh:       make(chan struct{}, 4),
	}

	s.ResetNewGame("")
	s.RefreshDiscoveredBases()

	if cfg.DBPath != "" {
		if err := s.OpenPath(cfg.DBPath); err != nil {
			s.SetStatus(fmt.Sprintf("Failed to open %s: %v", cfg.DBPath, err), true)
		}
	} else if cfg.PGNPath != "" {
		if err := s.OpenPath(cfg.PGNPath); err != nil {
			s.SetStatus(fmt.Sprintf("Failed to open %s: %v", cfg.PGNPath, err), true)
		}
	}

	if cfg.GameNumber > 0 {
		_ = s.LoadGameNumber(cfg.GameNumber)
	}

	if cfg.InitialFEN != "" {
		if err := s.LoadFEN(cfg.InitialFEN); err != nil {
			s.SetStatus(fmt.Sprintf("Invalid FEN: %v", err), true)
		}
	}

	if cfg.AutoEngine {
		_ = s.StartEngine()
	}

	s.onPositionChanged()
	if s.StatusMsg == "" {
		modeTag := "Unicode 24-bit RGB"
		if s.UseKitty {
			modeTag = "Kitty Graphics HD"
		}
		s.SetStatus(fmt.Sprintf("Ready (%s) — click/drag squares, press m to enter move, ? for help", modeTag), false)
	}

	return s
}

// Close shuts down any running UCI engine and open Scid database cleanly.
func (s *Session) Close() {
	s.StopEngine()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ScidDB != nil && s.ScidDB.IsOpen {
		_ = s.ScidDB.Close()
		s.ScidDB = nil
	}
}

func (s *Session) notifyRedraw() {
	select {
	case s.RedrawCh <- struct{}{}:
	default:
	}
}

// SetStatus updates the bottom status line message.
func (s *Session) SetStatus(msg string, isErr bool) {
	s.StatusMsg = msg
	s.StatusIsErr = isErr
}

// InvalidateKittyCache forces the next render pass to re-transmit the Kitty PNG placement.
func (s *Session) InvalidateKittyCache() {
	s.lastBoardHash = 0
}

// ResetNewGame initializes a clean game starting from startFEN (or standard initial position if empty).
func (s *Session) ResetNewGame(startFEN string) {
	g := chess.NewGame()
	g.SetTag("White", "White")
	g.SetTag("Black", "Black")
	g.SetTag("Event", "NewScid CLI Analysis")
	g.SetTag("Date", time.Now().Format("2006.01.02"))

	var b *chess.Board
	if strings.TrimSpace(startFEN) != "" {
		parsed, err := chess.ParseFEN(startFEN)
		if err == nil {
			b = parsed
			g.StartFEN = parsed.FEN()
			g.SetTag("FEN", g.StartFEN)
			g.SetTag("SetUp", "1")
		}
	}
	if b == nil {
		b = chess.NewBoard()
		g.StartFEN = chess.StartingFEN
	}

	s.Game = g
	s.Board = b
	s.Path = nil
	s.CurPly = 0
	s.SelectedSq = chess.NoSquare
	s.DragFromSq = chess.NoSquare
	s.ActiveGameIndex = 0
}

// LoadFEN resets the board to a specific FEN string.
func (s *Session) LoadFEN(fen string) error {
	b, err := chess.ParseFEN(strings.TrimSpace(fen))
	if err != nil {
		return err
	}
	s.ResetNewGame(b.FEN())
	s.onPositionChanged()
	return nil
}

// SetGame loads a parsed chess.Game onto the workstation board and rebuilds the mainline PlyFrame slice.
func (s *Session) SetGame(g *chess.Game, index int) {
	if g == nil {
		s.ResetNewGame("")
		s.onPositionChanged()
		return
	}
	s.Game = g
	s.ActiveGameIndex = index
	s.SelectedSq = chess.NoSquare
	s.DragFromSq = chess.NoSquare

	s.rebuildMainlineFromRoot()
	s.JumpToPly(0)
}

func (s *Session) rebuildMainlineFromRoot() {
	b := s.startBoardForGame()
	var frames []PlyFrame
	curr := s.Game.RootMove
	ply := 1
	for curr != nil {
		san := curr.SAN
		if san == "" {
			san = b.MoveToSAN(curr.Move)
			curr.SAN = san
		}
		if curr.UCI == "" {
			curr.UCI = curr.Move.String()
		}
		b.MakeMove(curr.Move)
		fen := b.FEN()
		curr.FEN = fen
		frames = append(frames, PlyFrame{
			Node:     curr,
			BoardFEN: fen,
			Move:     curr.Move,
			SAN:      san,
			Ply:      ply,
		})
		curr = curr.Next
		ply++
	}
	s.Path = frames
}

func (s *Session) startBoardForGame() *chess.Board {
	if s.Game != nil && s.Game.StartFEN != "" && s.Game.StartFEN != chess.StartingFEN {
		if b, err := chess.ParseFEN(s.Game.StartFEN); err == nil {
			return b
		}
	}
	if s.Game != nil {
		if fenTag := s.Game.Tag("FEN"); fenTag != "" && fenTag != "?" {
			if b, err := chess.ParseFEN(fenTag); err == nil {
				return b
			}
		}
	}
	return chess.NewBoard()
}

// JumpToPly moves the active board state to ply (0 .. len(s.Path)).
func (s *Session) JumpToPly(ply int) {
	if ply < 0 {
		ply = 0
	}
	if ply > len(s.Path) {
		ply = len(s.Path)
	}
	s.CurPly = ply
	s.SelectedSq = chess.NoSquare
	s.DragFromSq = chess.NoSquare

	if ply == 0 {
		s.Board = s.startBoardForGame()
	} else {
		if b, err := chess.ParseFEN(s.Path[ply-1].BoardFEN); err == nil {
			s.Board = b
		} else {
			b := s.startBoardForGame()
			for i := 0; i < ply; i++ {
				b.MakeMove(s.Path[i].Move)
			}
			s.Board = b
		}
	}
	s.onPositionChanged()
}

// StepPly advances or rewinds by delta plies.
func (s *Session) StepPly(delta int) {
	s.JumpToPly(s.CurPly + delta)
}

// CycleVariation switches among sibling variations at the next move (or current move) if available.
func (s *Session) CycleVariation(dir int) bool {
	// Check variations on the move about to be played (at s.CurPly), or the move just played (s.CurPly-1)
	targetIdx := s.CurPly
	if targetIdx >= len(s.Path) || len(s.Path[targetIdx].Node.Variations) == 0 {
		if s.CurPly > 0 && len(s.Path[s.CurPly-1].Node.Variations) > 0 {
			targetIdx = s.CurPly - 1
		} else {
			return false
		}
	}

	anchor := s.Path[targetIdx].Node
	if len(anchor.Variations) == 0 {
		return false
	}

	// Rotate: swap chosen variation branch into active Path from targetIdx onward
	var chosen *chess.MoveNode
	if dir >= 0 {
		chosen = anchor.Variations[0]
	} else {
		chosen = anchor.Variations[len(anchor.Variations)-1]
	}
	if chosen == nil {
		return false
	}

	// Build board right before targetIdx
	var b *chess.Board
	if targetIdx == 0 {
		b = s.startBoardForGame()
	} else {
		var err error
		b, err = chess.ParseFEN(s.Path[targetIdx-1].BoardFEN)
		if err != nil {
			b = s.startBoardForGame()
			for i := 0; i < targetIdx; i++ {
				b.MakeMove(s.Path[i].Move)
			}
		}
	}

	// Rebuild path tail following chosen variation
	newTail := make([]PlyFrame, 0, 16)
	curr := chosen
	ply := targetIdx + 1
	for curr != nil {
		san := curr.SAN
		if san == "" {
			san = b.MoveToSAN(curr.Move)
			curr.SAN = san
		}
		b.MakeMove(curr.Move)
		fen := b.FEN()
		curr.FEN = fen
		newTail = append(newTail, PlyFrame{
			Node:     curr,
			BoardFEN: fen,
			Move:     curr.Move,
			SAN:      san,
			Ply:      ply,
		})
		curr = curr.Next
		ply++
	}

	// Rotate variation list on anchor so pressing ']' again cycles through remaining variations
	oldClone := *anchor
	oldClone.Variations = nil
	if len(anchor.Variations) > 1 {
		rest := append([]*chess.MoveNode{}, anchor.Variations[1:]...)
		chosen.Variations = append(rest, &oldClone)
	} else {
		chosen.Variations = []*chess.MoveNode{&oldClone}
	}
	if len(newTail) > 0 {
		newTail[0].Node = chosen
	}

	s.Path = append(s.Path[:targetIdx], newTail...)
	s.JumpToPly(targetIdx + 1)
	s.SetStatus(fmt.Sprintf("Switched to variation %s", chosen.SAN), false)
	return true
}

// PlayMoveInput parses a SAN or UCI move string and plays it on the active board.
func (s *Session) PlayMoveInput(raw string) error {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	// Strip optional leading move number like "1.e4" or "12...Nf6"
	if dotIdx := strings.LastIndex(text, "."); dotIdx != -1 && dotIdx+1 < len(text) {
		text = strings.TrimSpace(text[dotIdx+1:])
	}

	m, err := s.Board.ParseSAN(text)
	if err != nil {
		m, err = s.Board.ParseUCI(text)
		if err != nil {
			return fmt.Errorf("illegal or unrecognized move %q", raw)
		}
	}
	return s.ApplyLegalMove(m)
}

// ApplyLegalMove executes a legal move on the board, advancing if it matches the next move on Path
// or appending/branching the move tree cleanly.
func (s *Session) ApplyLegalMove(m chess.Move) error {
	legal := s.Board.GenerateLegalMoves()
	matched := false
	for _, lm := range legal {
		if lm.From == m.From && lm.To == m.To {
			if m.Promo == chess.None && lm.Promo != chess.None {
				if lm.Promo == chess.Queen {
					m = lm
					matched = true
					break
				}
				continue
			}
			if lm.Promo == m.Promo {
				m = lm
				matched = true
				break
			}
		}
	}
	if !matched {
		return fmt.Errorf("illegal move %s", m.String())
	}

	// If next move on current path matches m, simply step forward
	if s.CurPly < len(s.Path) && s.Path[s.CurPly].Move.Equals(m) {
		s.StepPly(1)
		return nil
	}

	san := s.Board.MoveToSAN(m)
	s.Board.MakeMove(m)
	newFEN := s.Board.FEN()

	newNode := &chess.MoveNode{
		Move: m,
		SAN:  san,
		UCI:  m.String(),
		FEN:  newFEN,
	}

	if s.CurPly == len(s.Path) {
		// Append to end of current line
		if s.CurPly == 0 {
			if s.Game.RootMove == nil {
				s.Game.RootMove = newNode
			} else {
				s.Game.RootMove.Variations = append(s.Game.RootMove.Variations, newNode)
			}
		} else {
			prevNode := s.Path[s.CurPly-1].Node
			if prevNode.Next == nil {
				prevNode.Next = newNode
			} else {
				prevNode.Next.Variations = append(prevNode.Next.Variations, newNode)
			}
		}
		s.Path = append(s.Path, PlyFrame{
			Node:     newNode,
			BoardFEN: newFEN,
			Move:     m,
			SAN:      san,
			Ply:      s.CurPly + 1,
		})
		s.CurPly++
	} else {
		// Branching mid-game: attach as variation on existing next move and follow new branch
		existing := s.Path[s.CurPly].Node
		existing.Variations = append(existing.Variations, newNode)
		s.Path = append(s.Path[:s.CurPly], PlyFrame{
			Node:     newNode,
			BoardFEN: newFEN,
			Move:     m,
			SAN:      san,
			Ply:      s.CurPly + 1,
		})
		s.CurPly = len(s.Path)
	}

	s.Game.PlyCount = len(s.Path)
	s.SelectedSq = chess.NoSquare
	s.DragFromSq = chess.NoSquare
	s.onPositionChanged()
	return nil
}

// LastMoveSquares returns the From and To squares of the most recently played ply (or NoSquare).
func (s *Session) LastMoveSquares() (chess.Square, chess.Square) {
	if s.CurPly > 0 && s.CurPly <= len(s.Path) {
		mv := s.Path[s.CurPly-1].Move
		return mv.From, mv.To
	}
	return chess.NoSquare, chess.NoSquare
}

// LegalTargetsFrom returns all legal destination squares and capture bitmask for pieces on fromSq.
func (s *Session) LegalTargetsFrom(fromSq chess.Square) ([]chess.Square, uint64) {
	if fromSq >= 64 || s.Board == nil {
		return nil, 0
	}
	p := s.Board.PieceAt(fromSq)
	if p == chess.Empty || p.Color() != s.Board.Turn {
		return nil, 0
	}
	var targets []chess.Square
	var capMask uint64
	seen := uint64(0)
	for _, mv := range s.Board.GenerateLegalMoves() {
		if mv.From == fromSq {
			bit := uint64(1) << uint(mv.To)
			if (seen & bit) == 0 {
				seen |= bit
				targets = append(targets, mv.To)
			}
			if mv.Capture != chess.None || s.Board.PieceAt(mv.To) != chess.Empty {
				capMask |= bit
			}
		}
	}
	return targets, capMask
}

// ActiveBoardArrows computes directional arrows for UCI Engine PV lines and Opening Tree book moves.
func (s *Session) ActiveBoardArrows() []BoardArrow {
	if !s.ShowArrows || s.Board == nil {
		return nil
	}
	s.mu.Lock()
	eval := s.LastEval
	evalFEN := s.EvalPositionFEN
	treeRes := s.TreeResult
	engOn := s.EngineEnabled
	tab := s.ActiveTab
	s.mu.Unlock()

	curFEN := s.Board.FEN()
	var arrows []BoardArrow

	if engOn && evalFEN == curFEN {
		for idx, line := range eval.Lines {
			if idx >= 2 {
				break
			}
			if len(line.MoveUCI) >= 4 {
				from := chess.SquareFromString(line.MoveUCI[0:2])
				to := chess.SquareFromString(line.MoveUCI[2:4])
				if from < 64 && to < 64 {
					kind := ArrowEnginePV1
					if idx == 1 {
						kind = ArrowEnginePV2
					}
					arrows = append(arrows, BoardArrow{From: from, To: to, Kind: kind})
				}
			}
		}
	}

	if (tab == TabTree || !engOn) && treeRes != nil && len(treeRes.Moves) > 0 {
		maxBook := 2
		if tab == TabTree {
			maxBook = 3
		}
		for i := 0; i < len(treeRes.Moves) && i < maxBook; i++ {
			mv := treeRes.Moves[i]
			uci := mv.UCI
			if len(uci) < 4 {
				if parsed, err := s.Board.ParseSAN(mv.Move); err == nil {
					uci = parsed.String()
				}
			}
			if len(uci) >= 4 {
				from := chess.SquareFromString(uci[0:2])
				to := chess.SquareFromString(uci[2:4])
				if from < 64 && to < 64 {
					dup := false
					for _, ex := range arrows {
						if ex.From == from && ex.To == to {
							dup = true
							break
						}
					}
					if !dup {
						arrows = append(arrows, BoardArrow{From: from, To: to, Kind: ArrowOpeningBook})
					}
				}
			}
		}
	}

	return arrows
}

func (s *Session) onPositionChanged() {
	// Update ECO classification
	sans := make([]string, 0, s.CurPly)
	for i := 0; i < s.CurPly && i < len(s.Path); i++ {
		sans = append(sans, s.Path[i].SAN)
	}
	s.CurrentECO = s.ECOClassifier.Classify(sans)
	s.ImbalanceRep = imbalance.AnalyzeBoard(s.Board)

	fen := s.Board.FEN()

	// Trigger live UCI engine search
	if s.EngineEnabled && s.Engine != nil {
		s.mu.Lock()
		s.LastEval = engine.EvalInfo{MultiPV: s.MultiPV}
		s.EvalPositionFEN = fen
		s.mu.Unlock()
		_ = s.Engine.Analyze(fen, nil, 0)
	}

	// Trigger async Opening Tree lookup
	s.refreshOpeningTreeAsync(fen)
}

func (s *Session) refreshOpeningTreeAsync(fen string) {
	s.mu.Lock()
	s.treeSeq++
	seq := s.treeSeq
	db := s.ScidDB
	isPGN := s.IsPGNSource
	pgnGames := s.PGNGames
	s.TreeBusy = (db != nil && db.IsOpen) || len(pgnGames) > 0
	s.mu.Unlock()

	if (db == nil || !db.IsOpen) && len(pgnGames) == 0 {
		s.mu.Lock()
		s.TreeResult = &tree.TreeResult{FEN: fen, Moves: []*tree.MoveStat{}}
		s.TreeBusy = false
		s.mu.Unlock()
		return
	}

	go func() {
		var res *tree.TreeResult
		if isPGN {
			res = tree.BuildTreeFromGames(fen, pgnGames)
		} else if db != nil && db.IsOpen {
			res, _ = tree.BuildTreeFromDatabase(db, fen, 3000)
		}
		s.mu.Lock()
		if s.treeSeq == seq {
			s.TreeResult = res
			s.TreeCursor = 0
			s.TreeBusy = false
			s.notifyRedraw()
		}
		s.mu.Unlock()
	}()
}

// RefreshDiscoveredBases scans Scid/NewScid standard folders plus local working directory for .si4 and .pgn files.
func (s *Session) RefreshDiscoveredBases() {
	bases := scid.DiscoverScidDatabases()
	seen := make(map[string]bool, len(bases))
	var filtered []scid.DiscoveredDatabase
	for _, b := range bases {
		if !b.Exists {
			continue
		}
		seen[b.BasePath] = true
		filtered = append(filtered, b)
	}

	// Also scan current directory for .si4 and .pgn files for smooth CLI usage
	if entries, err := os.ReadDir("."); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			abs, err := filepath.Abs(name)
			if err != nil {
				abs = name
			}
			if fileformat.IsScidIndexPath(name) {
				base := fileformat.TrimScidExtensions(abs)
				if !seen[base] {
					seen[base] = true
					info := scid.InspectDatabase(base, "Working Directory")
					if info.Exists {
						filtered = append([]scid.DiscoveredDatabase{info}, filtered...)
					}
				}
			} else if fileformat.IsPGNPath(name) {
				if !seen[abs] {
					seen[abs] = true
					fi, _ := e.Info()
					sz := ""
					if fi != nil {
						sz = scid.FormatFileSize(fi.Size())
					}
					filtered = append([]scid.DiscoveredDatabase{{
						Name:     strings.TrimSuffix(name, filepath.Ext(name)),
						BasePath: abs,
						Type:     "pgn",
						Category: "Working Directory",
						SizeStr:  sz,
						Exists:   true,
					}}, filtered...)
				}
			}
		}
	}

	s.DiscoveredBases = filtered
	if s.BasesCursor >= len(s.DiscoveredBases) {
		s.BasesCursor = 0
	}
}

// OpenPath opens either a Scid v4 database (.si4 or base path) or a .pgn file.
func (s *Session) OpenPath(rawPath string) error {
	clean := strings.TrimSpace(rawPath)
	if clean == "" {
		return fmt.Errorf("empty path")
	}
	if strings.HasPrefix(clean, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			clean = filepath.Join(home, clean[2:])
		}
	}

	if fileformat.IsPGNPath(clean) {
		return s.openPGNFile(clean)
	}
	// Check if rawPath + ".pgn" exists when exact .si4 does not
	si4Candidate := fileformat.ScidIndexPath(clean)
	if _, err := os.Stat(si4Candidate); err != nil {
		if fi, err2 := os.Stat(clean); err2 == nil && !fi.IsDir() && strings.HasSuffix(strings.ToLower(clean), ".pgn") {
			return s.openPGNFile(clean)
		}
	}
	return s.openScidDB(clean)
}

func (s *Session) openScidDB(path string) error {
	db := scid.NewDatabase()
	if err := db.Open(path); err != nil {
		return err
	}

	s.mu.Lock()
	if s.ScidDB != nil && s.ScidDB.IsOpen {
		_ = s.ScidDB.Close()
	}
	s.ScidDB = db
	s.IsPGNSource = false
	s.PGNGames = nil
	s.PGNSummaries = nil
	s.ActiveDBPath = db.BasePath
	s.ActiveDBName = filepath.Base(db.BasePath)
	s.FilterQuery = ""
	s.FilterCriteria = scid.SearchCriteria{IgnoreDeleted: true}
	s.GameListCursor = 0
	s.mu.Unlock()

	s.RefreshGameList()
	if db.NumGames() > 0 {
		_ = s.LoadGameNumber(1)
	} else {
		s.ResetNewGame("")
		s.onPositionChanged()
	}
	s.SetStatus(fmt.Sprintf("Opened Scid database %s (%d games)", s.ActiveDBName, db.NumGames()), false)
	return nil
}

func (s *Session) openPGNFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	games, err := chess.ParsePGNDatabase(f)
	if err != nil {
		return err
	}

	summaries := make([]*scid.GameSummary, 0, len(games))
	for i, g := range games {
		g.ID = i + 1
		wElo, _ := strconv.Atoi(g.Tag("WhiteElo"))
		bElo, _ := strconv.Atoi(g.Tag("BlackElo"))
		ecoTag := g.Tag("ECO")
		if ecoTag == "?" {
			ecoTag = ""
		}
		summaries = append(summaries, &scid.GameSummary{
			ID:       i + 1,
			White:    g.Tag("White"),
			Black:    g.Tag("Black"),
			Result:   g.Result,
			Date:     g.Tag("Date"),
			Event:    g.Tag("Event"),
			Site:     g.Tag("Site"),
			Round:    g.Tag("Round"),
			WhiteElo: wElo,
			BlackElo: bElo,
			ECO:      ecoTag,
			Moves:    (g.PlyCount + 1) / 2,
		})
	}

	s.mu.Lock()
	if s.ScidDB != nil && s.ScidDB.IsOpen {
		_ = s.ScidDB.Close()
		s.ScidDB = nil
	}
	s.IsPGNSource = true
	s.PGNGames = games
	s.PGNSummaries = summaries
	s.ActiveDBPath = path
	s.ActiveDBName = filepath.Base(path)
	s.FilterQuery = ""
	s.FilterCriteria = scid.SearchCriteria{}
	s.GameListCursor = 0
	s.mu.Unlock()

	s.RefreshGameList()
	if len(games) > 0 {
		_ = s.LoadGameNumber(1)
	} else {
		s.ResetNewGame("")
		s.onPositionChanged()
	}
	s.SetStatus(fmt.Sprintf("Opened PGN %s (%d games)", s.ActiveDBName, len(games)), false)
	return nil
}

// LoadGameNumber loads 1-based gameNum from the active Scid database or PGN file.
func (s *Session) LoadGameNumber(gameNum int) error {
	if s.IsPGNSource {
		if gameNum < 1 || gameNum > len(s.PGNGames) {
			return fmt.Errorf("game #%d out of range (1..%d)", gameNum, len(s.PGNGames))
		}
		s.SetGame(s.PGNGames[gameNum-1], gameNum)
		return nil
	}
	if s.ScidDB != nil && s.ScidDB.IsOpen {
		g, err := s.ScidDB.GetGame(gameNum)
		if err != nil {
			return err
		}
		s.SetGame(g, gameNum)
		return nil
	}
	return fmt.Errorf("no database open")
}

// ApplyFilterQuery parses a CLI search filter string and updates GameSummaries.
// Supports tokens: player:<x>, white:<x>, black:<x>, event:<x>, site:<x>, eco:<x>, minelo:<n>, maxelo:<n>, year:<n>, res:<1-0|0-1|1/2>, pos (current FEN), or free-text.
func (s *Session) ApplyFilterQuery(q string) {
	s.FilterQuery = strings.TrimSpace(q)
	crit := scid.SearchCriteria{IgnoreDeleted: true}
	var freeWords []string

	for _, tok := range strings.Fields(s.FilterQuery) {
		lower := strings.ToLower(tok)
		if lower == "pos" || lower == "position" || lower == "fen" {
			crit.PositionFEN = s.Board.FEN()
			continue
		}
		if kv := strings.SplitN(tok, ":", 2); len(kv) == 2 {
			k := strings.ToLower(kv[0])
			v := strings.ReplaceAll(kv[1], "_", " ")
			switch k {
			case "w", "white":
				crit.White = v
			case "b", "black":
				crit.Black = v
			case "p", "player":
				crit.Player = v
			case "e", "event":
				crit.Event = v
			case "s", "site":
				crit.Site = v
			case "eco":
				crit.ECO = strings.ToUpper(v)
			case "res", "result":
				crit.Result = v
			case "minelo", "elo":
				crit.MinElo, _ = strconv.Atoi(v)
			case "maxelo":
				crit.MaxElo, _ = strconv.Atoi(v)
			case "year":
				yr, _ := strconv.Atoi(v)
				crit.YearFrom = yr
				crit.YearTo = yr
			default:
				freeWords = append(freeWords, tok)
			}
		} else {
			freeWords = append(freeWords, tok)
		}
	}

	if len(freeWords) > 0 && crit.Player == "" {
		crit.Player = strings.Join(freeWords, " ")
	}

	s.FilterCriteria = crit
	s.GameListCursor = 0
	s.RefreshGameList()
}

// RefreshGameList reloads the filtered GameSummaries list from ScidDB or PGNSummaries.
func (s *Session) RefreshGameList() {
	if s.IsPGNSource {
		var matched []*scid.GameSummary
		var posSet map[int]bool
		if s.FilterCriteria.PositionFEN != "" {
			idxs := tree.SearchPositionInGames(s.FilterCriteria.PositionFEN, s.PGNGames)
			posSet = make(map[int]bool, len(idxs))
			for _, idx := range idxs {
				posSet[idx] = true
			}
		}
		for i, gs := range s.PGNSummaries {
			if posSet != nil && !posSet[i] {
				continue
			}
			if !matchSummaryCriteria(gs, s.FilterCriteria) {
				continue
			}
			matched = append(matched, gs)
		}
		s.TotalMatched = len(matched)
		if len(matched) > s.GamePageSize {
			matched = matched[:s.GamePageSize]
		}
		s.GameSummaries = matched
		return
	}

	if s.ScidDB != nil && s.ScidDB.IsOpen {
		list, total, err := s.ScidDB.GetGameSummaries(0, s.GamePageSize, s.FilterCriteria)
		if err == nil {
			s.GameSummaries = list
			s.TotalMatched = total
			return
		}
	}

	s.GameSummaries = nil
	s.TotalMatched = 0
}

func matchSummaryCriteria(gs *scid.GameSummary, c scid.SearchCriteria) bool {
	if c.White != "" && !strings.Contains(strings.ToLower(gs.White), strings.ToLower(c.White)) {
		return false
	}
	if c.Black != "" && !strings.Contains(strings.ToLower(gs.Black), strings.ToLower(c.Black)) {
		return false
	}
	if c.Player != "" {
		q := strings.ToLower(c.Player)
		if !strings.Contains(strings.ToLower(gs.White), q) &&
			!strings.Contains(strings.ToLower(gs.Black), q) &&
			!strings.Contains(strings.ToLower(gs.Event), q) {
			return false
		}
	}
	if c.Event != "" && !strings.Contains(strings.ToLower(gs.Event), strings.ToLower(c.Event)) {
		return false
	}
	if c.ECO != "" && !strings.HasPrefix(strings.ToUpper(gs.ECO), strings.ToUpper(c.ECO)) {
		return false
	}
	if c.Result != "" && !strings.Contains(string(gs.Result), c.Result) {
		return false
	}
	if c.MinElo > 0 && gs.WhiteElo < c.MinElo && gs.BlackElo < c.MinElo {
		return false
	}
	return true
}

// StartEngine discovers and starts a UCI chess engine (Stockfish / Scid configured engine).
func (s *Session) StartEngine() error {
	if s.Engine != nil {
		s.EngineEnabled = true
		s.onPositionChanged()
		return nil
	}

	path := strings.TrimSpace(s.EnginePath)
	if path == "" {
		for _, cfg := range engine.DiscoverScidEngines() {
			if cfg.Available && cfg.UCI && cfg.ResolvedCmd != "" {
				path = cfg.ResolvedCmd
				break
			}
		}
	}
	if path == "" {
		for _, bin := range []string{"stockfish", "fairy-stockfish", "critter", "fruit"} {
			if resolved, err := exec.LookPath(bin); err == nil {
				path = resolved
				break
			}
		}
	}
	if path == "" {
		s.EngineEnabled = false
		return fmt.Errorf("no UCI engine found in PATH or Scid config (use :engine /path/to/stockfish)")
	}

	eng := engine.NewEngine(path)
	if err := eng.Start(); err != nil {
		s.EngineEnabled = false
		return err
	}
	_ = eng.SetThreads(2)
	_ = eng.SetHash(64)
	_ = eng.SetMultiPV(s.MultiPV)

	eng.SetOnEval(func(info engine.EvalInfo) {
		s.mu.Lock()
		s.LastEval = info
		s.mu.Unlock()
		s.notifyRedraw()
	})

	s.Engine = eng
	s.EnginePath = path
	s.EngineName = eng.Name
	if s.EngineName == "" {
		s.EngineName = filepath.Base(path)
	}
	s.EngineEnabled = true
	s.onPositionChanged()
	return nil
}

// StopEngine halts and terminates the running UCI engine subprocess.
func (s *Session) StopEngine() {
	if s.Engine != nil {
		_ = s.Engine.Quit()
		s.Engine = nil
	}
	s.EngineEnabled = false
}

// ToggleEngine toggles live UCI analysis on/off.
func (s *Session) ToggleEngine() {
	if s.EngineEnabled {
		if s.Engine != nil {
			_ = s.Engine.Stop()
		}
		s.EngineEnabled = false
		s.SetStatus("UCI engine paused", false)
		return
	}
	if err := s.StartEngine(); err != nil {
		s.SetStatus(err.Error(), true)
		return
	}
	s.SetStatus(fmt.Sprintf("UCI engine active: %s (MultiPV=%d)", s.EngineName, s.MultiPV), false)
}

// AdjustMultiPV changes Multi-PV line count between 1 and 5.
func (s *Session) AdjustMultiPV(delta int) {
	next := s.MultiPV + delta
	if next < 1 {
		next = 1
	}
	if next > 5 {
		next = 5
	}
	if next == s.MultiPV {
		return
	}
	s.MultiPV = next
	if s.Engine != nil {
		_ = s.Engine.SetMultiPV(next)
		if s.EngineEnabled {
			_ = s.Engine.Analyze(s.Board.FEN(), nil, 0)
		}
	}
	s.SetStatus(fmt.Sprintf("Engine Multi-PV set to %d", s.MultiPV), false)
}

// CycleTheme advances to the next Scid/NewScid board color theme.
func (s *Session) CycleTheme() {
	themes := sets.GetAvailableBoardThemes()
	idx := 0
	for i, t := range themes {
		if t.ID == s.ThemeID {
			idx = (i + 1) % len(themes)
			break
		}
	}
	s.ThemeID = themes[idx].ID
	s.SetStatus(fmt.Sprintf("Board theme: %s (%s)", themes[idx].Name, themes[idx].ID), false)
}

// CyclePieceSet advances to the next embedded high-DPI tournament piece set.
func (s *Session) CyclePieceSet() {
	idx := 0
	for i, id := range AvailableTUIPieceSets {
		if id == s.PieceSetID {
			idx = (i + 1) % len(AvailableTUIPieceSets)
			break
		}
	}
	s.PieceSetID = AvailableTUIPieceSets[idx]
	s.SetStatus(fmt.Sprintf("Piece set: %s", s.PieceSetID), false)
}

// CycleNotation cycles SAN -> FAN -> LAN -> UCI notation styles in the scorebook.
func (s *Session) CycleNotation() {
	order := []chess.NotationStyle{
		chess.NotationSAN,
		chess.NotationFAN,
		chess.NotationLAN,
		chess.NotationUCI,
	}
	idx := 0
	for i, st := range order {
		if st == s.Notation {
			idx = (i + 1) % len(order)
			break
		}
	}
	s.Notation = order[idx]
	s.SetStatus(fmt.Sprintf("Move notation: %s", strings.ToUpper(string(s.Notation))), false)
}

// RunFairPlayAsync computes a longitudinal FairPlay Inspector dossier in the background.
func (s *Session) RunFairPlayAsync(playerName string) {
	target := strings.TrimSpace(playerName)
	if target == "" {
		w := s.Game.Tag("White")
		b := s.Game.Tag("Black")
		if s.ForensicsTarget == w && b != "" && b != "?" {
			target = b
		} else if w != "" && w != "?" && w != "White" {
			target = w
		} else if len(s.GameSummaries) > 0 {
			target = s.GameSummaries[0].White
		} else {
			target = "White"
		}
	}

	s.ActiveTab = TabForensics
	s.ForensicsMode = "fairplay"
	s.ForensicsTarget = target
	s.ForensicsScroll = 0
	s.ForensicsBusy = true
	s.SetStatus(fmt.Sprintf("Running FairPlay forensic analysis for %q...", target), false)

	db := s.ScidDB
	isPGN := s.IsPGNSource
	pgnGames := s.PGNGames
	curGame := s.Game

	go func() {
		var rep *fairplay.PlayerFairPlayDossier
		if !isPGN && db != nil && db.IsOpen {
			rep, _ = fairplay.AnalyzePlayerFairPlay(db, target, 250)
		} else if isPGN && len(pgnGames) > 0 {
			ids := make([]int, len(pgnGames))
			for i := range pgnGames {
				ids[i] = i + 1
			}
			rep = fairplay.BuildPlayerFairPlayDossier(target, pgnGames, ids)
		} else if curGame != nil {
			rep = fairplay.BuildPlayerFairPlayDossier(target, []*chess.Game{curGame}, []int{1})
		}

		s.mu.Lock()
		s.FairPlayReport = rep
		s.ForensicsBusy = false
		if rep != nil {
			s.SetStatus(fmt.Sprintf("FairPlay dossier ready for %s (%d games, issue=%.1f/100)", rep.PlayerName, rep.TotalGamesAnalyzed, rep.OverallIssueScore), false)
		} else {
			s.SetStatus(fmt.Sprintf("No games matched %q for FairPlay dossier", target), true)
		}
		s.mu.Unlock()
		s.notifyRedraw()
	}()
}

// RunScoutAsync compiles an Opponent Preparation Scout dossier in the background.
func (s *Session) RunScoutAsync(playerName string) {
	target := strings.TrimSpace(playerName)
	if target == "" {
		w := s.Game.Tag("White")
		b := s.Game.Tag("Black")
		if s.ForensicsTarget == w && b != "" && b != "?" {
			target = b
		} else if w != "" && w != "?" && w != "White" {
			target = w
		} else if len(s.GameSummaries) > 0 {
			target = s.GameSummaries[0].White
		} else {
			target = "White"
		}
	}

	s.ActiveTab = TabForensics
	s.ForensicsMode = "scout"
	s.ForensicsTarget = target
	s.ForensicsScroll = 0
	s.ForensicsBusy = true
	s.SetStatus(fmt.Sprintf("Compiling opponent scout dossier for %q...", target), false)

	db := s.ScidDB
	isPGN := s.IsPGNSource
	pgnSummaries := s.PGNSummaries

	go func() {
		var summaries []*scid.GameSummary
		if !isPGN && db != nil && db.IsOpen {
			summaries, _, _ = db.GetGameSummaries(0, 5000, scid.SearchCriteria{Player: target, IgnoreDeleted: true})
		} else if isPGN {
			summaries = pgnSummaries
		}
		rep := scout.GenerateReport(target, summaries)

		s.mu.Lock()
		s.ScoutReport = rep
		s.ForensicsBusy = false
		s.SetStatus(fmt.Sprintf("Scout dossier ready for %s (%d games)", rep.PlayerName, rep.TotalGames), false)
		s.mu.Unlock()
		s.notifyRedraw()
	}()
}

// SaveCurrentGameToDB saves the active game to the open Scid v4 database or exports PGN.
func (s *Session) SaveCurrentGameToDB() error {
	if s.ScidDB != nil && s.ScidDB.IsOpen {
		id, err := s.ScidDB.SaveGame(s.Game)
		if err != nil {
			return err
		}
		s.ActiveGameIndex = id
		s.RefreshGameList()
		return nil
	}
	return fmt.Errorf("open a Scid .si4 database first (or use :export <file.pgn>)")
}

// ExportCurrentPGN writes the current game's PGN to filePath.
func (s *Session) ExportCurrentPGN(filePath string) error {
	p := strings.TrimSpace(filePath)
	if p == "" {
		p = "newscid-export.pgn"
	}
	return os.WriteFile(p, []byte(s.Game.ExportPGN()), 0644)
}
