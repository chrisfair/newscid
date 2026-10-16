package engine

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"newscid/pkg/chess"
)

// PVLine represents one candidate principal variation line in Multi-PV mode.
type PVLine struct {
	MultiPV     int      `json:"multipv"`     // 1-indexed rank of this line (1 = best)
	Depth       int      `json:"depth"`
	ScoreCp     int      `json:"scoreCp"`     // centipawns
	MateIn      int      `json:"mateIn"`      // moves to mate (0 if not mate)
	NPS         int64    `json:"nps"`         // nodes per second
	Nodes       int64    `json:"nodes"`
	PV          []string `json:"pv"`          // UCI move sequence
	MoveUCI     string   `json:"moveUci"`     // first move of PV
	SAN         string   `json:"san"`         // SAN representation of candidate move
	PVFormatted string   `json:"pvFormatted"` // formatted move string e.g. "1. e4 e5 2. Nf3"
}

// EvalInfo represents real-time analysis details from an engine.
type EvalInfo struct {
	Depth    int      `json:"depth"`
	ScoreCp  int      `json:"scoreCp"`  // centipawns (from line 1)
	MateIn   int      `json:"mateIn"`   // moves to mate (0 if not mate)
	NPS      int64    `json:"nps"`      // nodes per second
	Nodes    int64    `json:"nodes"`
	PV       []string `json:"pv"`       // principal variation moves (line 1)
	BestMove string   `json:"bestMove"`
	MultiPV  int      `json:"multiPv"`  // current configured MultiPV count (1-5)
	Lines    []PVLine `json:"lines"`    // all active PV lines sorted by MultiPV rank
}

// Engine represents a running UCI engine process.
type Engine struct {
	Path             string
	Name             string
	Author           string
	cmd              *exec.Cmd
	stdin            io.WriteCloser
	stdout           *bufio.Scanner
	mu               sync.Mutex
	isAnalyzing      bool
	stopChan         chan struct{}
	bestMoveListener chan string
	readyListener    chan struct{}
	lastEval         EvalInfo
	onEval           func(EvalInfo)
	multiPV          int
	threads          int
	hashMB           int
	currentFEN       string
	pvLines          map[int]PVLine
}

func NewEngine(path string) *Engine {
	return &Engine{
		Path:     path,
		stopChan: make(chan struct{}),
		multiPV:  1,
		pvLines:  make(map[int]PVLine),
	}
}

// Start launches the engine process and initializes UCI protocol.
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	cmd := exec.Command(e.Path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	e.cmd = cmd
	e.stdin = stdin
	e.stdout = bufio.NewScanner(stdoutPipe)

	// Send uci
	fmt.Fprintln(e.stdin, "uci")

	// Read until uciok
	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if strings.HasPrefix(line, "id name ") {
			e.Name = strings.TrimPrefix(line, "id name ")
		} else if strings.HasPrefix(line, "id author ") {
			e.Author = strings.TrimPrefix(line, "id author ")
		} else if line == "uciok" {
			break
		}
	}

	// Apply configured options
	if e.multiPV > 1 {
		fmt.Fprintf(e.stdin, "setoption name MultiPV value %d\n", e.multiPV)
	}
	if e.threads > 0 {
		fmt.Fprintf(e.stdin, "setoption name Threads value %d\n", e.threads)
	}
	if e.hashMB > 0 {
		fmt.Fprintf(e.stdin, "setoption name Hash value %d\n", e.hashMB)
	}
	if e.multiPV > 1 || e.threads > 0 || e.hashMB > 0 {
		fmt.Fprintln(e.stdin, "isready")
		for e.stdout.Scan() {
			if strings.TrimSpace(e.stdout.Text()) == "readyok" {
				break
			}
		}
	}

	// Start background listener
	go e.listenOutput()

	return nil
}

// SetMultiPV configures how many candidate principal variations to analyze (1 to 10).
func (e *Engine) SetMultiPV(n int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if n < 1 {
		n = 1
	} else if n > 10 {
		n = 10
	}
	e.multiPV = n

	if e.stdin != nil {
		fmt.Fprintf(e.stdin, "setoption name MultiPV value %d\n", n)
		fmt.Fprintln(e.stdin, "isready")
	}

	e.pvLines = make(map[int]PVLine)
	e.lastEval.MultiPV = n
	return nil
}

// GetMultiPV returns current MultiPV setting.
func (e *Engine) GetMultiPV() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.multiPV <= 0 {
		return 1
	}
	return e.multiPV
}

// SetThreads configures engine search threads.
func (e *Engine) SetThreads(n int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if n < 1 {
		n = 1
	}
	e.threads = n

	if e.stdin != nil {
		fmt.Fprintf(e.stdin, "setoption name Threads value %d\n", n)
	}
	return nil
}

// SetHash configures engine hash table memory in megabytes.
func (e *Engine) SetHash(mb int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if mb < 1 {
		mb = 16
	}
	e.hashMB = mb

	if e.stdin != nil {
		fmt.Fprintf(e.stdin, "setoption name Hash value %d\n", mb)
	}
	return nil
}

// SetOption sends a generic UCI setoption command.
func (e *Engine) SetOption(name, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin != nil {
		fmt.Fprintf(e.stdin, "setoption name %s value %s\n", name, value)
		fmt.Fprintln(e.stdin, "isready")
	}
	return nil
}

// SetOnEval registers a callback for live analysis updates.
func (e *Engine) SetOnEval(fn func(EvalInfo)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onEval = fn
}

// GetLastEval returns the most recent evaluation.
func (e *Engine) GetLastEval() EvalInfo {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastEval
}

// Analyze sends a position and begins analysis.
func (e *Engine) Analyze(fen string, moves []string, maxDepth int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin == nil {
		return fmt.Errorf("engine not started")
	}

	// Stop previous search
	fmt.Fprintln(e.stdin, "stop")

	// Ensure multiPV is applied
	if e.multiPV > 1 {
		fmt.Fprintf(e.stdin, "setoption name MultiPV value %d\n", e.multiPV)
	}

	e.currentFEN = fen
	e.pvLines = make(map[int]PVLine)
	e.lastEval = EvalInfo{
		MultiPV: e.multiPV,
		Lines:   make([]PVLine, 0),
	}

	posCmd := "position fen " + fen
	if len(moves) > 0 {
		posCmd += " moves " + strings.Join(moves, " ")
	}
	fmt.Fprintln(e.stdin, posCmd)

	goCmd := "go infinite"
	if maxDepth > 0 {
		goCmd = fmt.Sprintf("go depth %d", maxDepth)
	}
	fmt.Fprintln(e.stdin, goCmd)
	e.isAnalyzing = true

	return nil
}

// StopAndSync halts any ongoing search and synchronizes with the engine via isready / readyok.
func (e *Engine) StopAndSync() error {
	e.mu.Lock()
	if e.stdin == nil {
		e.mu.Unlock()
		return fmt.Errorf("engine not started")
	}

	readyCh := make(chan struct{}, 1)
	e.readyListener = readyCh

	if e.isAnalyzing {
		fmt.Fprintln(e.stdin, "stop")
		e.isAnalyzing = false
	}
	fmt.Fprintln(e.stdin, "isready")
	e.mu.Unlock()

	select {
	case <-readyCh:
		e.mu.Lock()
		e.readyListener = nil
		e.mu.Unlock()
		return nil
	case <-time.After(1500 * time.Millisecond):
		e.mu.Lock()
		e.readyListener = nil
		e.mu.Unlock()
		return nil
	}
}

// SearchBestMove searches for the best move in a given position with time limit in milliseconds.
func (e *Engine) SearchBestMove(fen string, moves []string, movetimeMs int) (string, EvalInfo, error) {
	// First ensure any previous search is stopped and engine is ready
	_ = e.StopAndSync()

	e.mu.Lock()
	if e.stdin == nil {
		e.mu.Unlock()
		return "", EvalInfo{}, fmt.Errorf("engine not started")
	}

	bestMoveCh := make(chan string, 1)
	e.bestMoveListener = bestMoveCh
	e.currentFEN = fen

	posCmd := "position fen " + fen
	if len(moves) > 0 {
		posCmd += " moves " + strings.Join(moves, " ")
	}
	fmt.Fprintln(e.stdin, posCmd)

	if movetimeMs <= 0 {
		movetimeMs = 500
	}
	fmt.Fprintf(e.stdin, "go movetime %d\n", movetimeMs)
	e.isAnalyzing = true
	e.mu.Unlock()

	select {
	case best := <-bestMoveCh:
		e.mu.Lock()
		eval := e.lastEval
		e.bestMoveListener = nil
		e.isAnalyzing = false
		e.mu.Unlock()
		return best, eval, nil
	case <-time.After(time.Duration(movetimeMs+3000) * time.Millisecond):
		e.mu.Lock()
		eval := e.lastEval
		best := eval.BestMove
		e.bestMoveListener = nil
		e.isAnalyzing = false
		e.mu.Unlock()
		if best != "" {
			return best, eval, nil
		}
		return "", eval, fmt.Errorf("engine search timed out")
	}
}

// Stop halts current analysis.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin != nil {
		fmt.Fprintln(e.stdin, "stop")
	}
	e.isAnalyzing = false
	return nil
}

// Quit stops the engine process gracefully, or forces kill if it does not exit within 1 second.
func (e *Engine) Quit() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin != nil {
		fmt.Fprintln(e.stdin, "quit")
		_ = e.stdin.Close()
	}
	if e.cmd != nil && e.cmd.Process != nil {
		done := make(chan struct{})
		go func() {
			_ = e.cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			_ = e.cmd.Process.Kill()
		}
	}
	e.cmd = nil
	e.stdin = nil
	e.isAnalyzing = false
	return nil
}

func (e *Engine) listenOutput() {
	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "info ") {
			e.mu.Lock()
			pvLine, hasPV := ParseInfoLine(line, e.currentFEN)
			if hasPV {
				e.pvLines[pvLine.MultiPV] = pvLine

				// Rebuild sorted lines
				lines := make([]PVLine, 0, len(e.pvLines))
				for rank := 1; rank <= 10; rank++ {
					if l, ok := e.pvLines[rank]; ok {
						lines = append(lines, l)
					}
				}
				e.lastEval.Lines = lines
				e.lastEval.MultiPV = e.multiPV

				// Line 1 updates root evaluation
				if pvLine.MultiPV == 1 {
					e.lastEval.Depth = pvLine.Depth
					e.lastEval.ScoreCp = pvLine.ScoreCp
					e.lastEval.MateIn = pvLine.MateIn
					e.lastEval.NPS = pvLine.NPS
					e.lastEval.Nodes = pvLine.Nodes
					e.lastEval.PV = pvLine.PV
					if pvLine.MoveUCI != "" {
						e.lastEval.BestMove = pvLine.MoveUCI
					}
				} else if e.lastEval.Depth == 0 {
					e.lastEval.Depth = pvLine.Depth
				}
			}
			cb := e.onEval
			currEval := e.lastEval
			e.mu.Unlock()

			if cb != nil && hasPV {
				cb(currEval)
			}
		} else if strings.HasPrefix(line, "bestmove ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				e.mu.Lock()
				e.lastEval.BestMove = parts[1]
				e.isAnalyzing = false
				cb := e.onEval
				currEval := e.lastEval
				listener := e.bestMoveListener
				e.mu.Unlock()

				if listener != nil {
					select {
					case listener <- parts[1]:
					default:
					}
				}

				if cb != nil {
					cb(currEval)
				}
			}
		} else if line == "readyok" {
			e.mu.Lock()
			listener := e.readyListener
			e.mu.Unlock()

			if listener != nil {
				select {
				case listener <- struct{}{}:
				default:
				}
			}
		}
	}
}

// FormatPV converts a sequence of UCI moves starting from fen into readable SAN notation and returns candidate move SAN and formatted PV.
func FormatPV(fen string, pvMoves []string) (candidateSAN string, formattedPV string) {
	if len(pvMoves) == 0 {
		return "", ""
	}

	board, err := chess.ParseFEN(fen)
	if err != nil {
		board = chess.NewBoard()
	}

	var formattedParts []string
	for i, uci := range pvMoves {
		if i >= 12 { // Limit PV preview length for clean display
			break
		}
		m, err := board.ParseUCI(uci)
		if err != nil {
			break
		}
		san := board.MoveToSAN(m)
		if i == 0 {
			candidateSAN = san
		}

		if board.Turn == chess.White {
			formattedParts = append(formattedParts, fmt.Sprintf("%d. %s", board.FullMove, san))
		} else {
			if i == 0 {
				formattedParts = append(formattedParts, fmt.Sprintf("%d... %s", board.FullMove, san))
			} else {
				formattedParts = append(formattedParts, san)
			}
		}
		board.MakeMove(m)
	}

	if candidateSAN == "" && len(pvMoves) > 0 {
		candidateSAN = pvMoves[0]
	}
	if len(formattedParts) == 0 && len(pvMoves) > 0 {
		formattedPV = strings.Join(pvMoves, " ")
	} else {
		formattedPV = strings.Join(formattedParts, " ")
	}

	return candidateSAN, formattedPV
}

// ParseInfoLine parses a UCI 'info' line and returns the PVLine and whether it contained a PV.
func ParseInfoLine(line string, currentFEN string) (PVLine, bool) {
	var pvLine PVLine
	pvLine.MultiPV = 1
	tokens := strings.Fields(line)
	hasPV := false

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t {
		case "depth":
			if i+1 < len(tokens) {
				pvLine.Depth, _ = strconv.Atoi(tokens[i+1])
			}
		case "multipv":
			if i+1 < len(tokens) {
				mpv, err := strconv.Atoi(tokens[i+1])
				if err == nil && mpv > 0 {
					pvLine.MultiPV = mpv
				}
			}
		case "score":
			if i+2 < len(tokens) {
				typ := tokens[i+1]
				val, _ := strconv.Atoi(tokens[i+2])
				if typ == "cp" {
					pvLine.ScoreCp = val
				} else if typ == "mate" {
					pvLine.MateIn = val
				}
			}
		case "nps":
			if i+1 < len(tokens) {
				pvLine.NPS, _ = strconv.ParseInt(tokens[i+1], 10, 64)
			}
		case "nodes":
			if i+1 < len(tokens) {
				pvLine.Nodes, _ = strconv.ParseInt(tokens[i+1], 10, 64)
			}
		case "pv":
			pvLine.PV = tokens[i+1:]
			if len(pvLine.PV) > 0 {
				pvLine.MoveUCI = pvLine.PV[0]
				candidateSAN, formattedPV := FormatPV(currentFEN, pvLine.PV)
				pvLine.SAN = candidateSAN
				pvLine.PVFormatted = formattedPV
			}
			return pvLine, true
		}
	}

	return pvLine, hasPV
}

// FindAvailableEngines scans standard paths for installed chess engines.
func FindAvailableEngines() []string {
	paths := []string{
		"/usr/bin/stockfish",
		"/usr/bin/fruit",
		"/usr/bin/phalanx",
		"/usr/games/stockfish",
		"/usr/local/bin/stockfish",
	}

	var found []string
	for _, p := range paths {
		if path, err := exec.LookPath(p); err == nil {
			found = append(found, path)
		}
	}
	return found
}
