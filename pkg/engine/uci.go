package engine

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// EvalInfo represents real-time analysis details from an engine.
type EvalInfo struct {
	Depth    int      `json:"depth"`
	ScoreCp  int      `json:"scoreCp"`  // centipawns
	MateIn   int      `json:"mateIn"`   // moves to mate (0 if not mate)
	NPS      int64    `json:"nps"`      // nodes per second
	Nodes    int64    `json:"nodes"`
	PV       []string `json:"pv"`       // principal variation moves
	BestMove string   `json:"bestMove"`
}

// Engine represents a running UCI engine process.
type Engine struct {
	Path        string
	Name        string
	Author      string
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      *bufio.Scanner
	mu          sync.Mutex
	isAnalyzing bool
	stopChan    chan struct{}
	lastEval    EvalInfo
	onEval      func(EvalInfo)
}

func NewEngine(path string) *Engine {
	return &Engine{
		Path:     path,
		stopChan: make(chan struct{}),
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

	// Send isready
	fmt.Fprintln(e.stdin, "isready")
	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if line == "readyok" {
			break
		}
	}

	// Start background listener
	go e.listenOutput()

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

// Quit stops the engine process.
func (e *Engine) Quit() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stdin != nil {
		fmt.Fprintln(e.stdin, "quit")
		e.stdin.Close()
	}
	if e.cmd != nil {
		_ = e.cmd.Wait()
	}
	e.cmd = nil
	e.stdin = nil
	return nil
}

func (e *Engine) listenOutput() {
	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "info ") {
			eval := parseInfoLine(line)
			e.mu.Lock()
			e.lastEval = eval
			cb := e.onEval
			e.mu.Unlock()

			if cb != nil {
				cb(eval)
			}
		} else if strings.HasPrefix(line, "bestmove ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				e.mu.Lock()
				e.lastEval.BestMove = parts[1]
				e.isAnalyzing = false
				cb := e.onEval
				currEval := e.lastEval
				e.mu.Unlock()

				if cb != nil {
					cb(currEval)
				}
			}
		}
	}
}

func parseInfoLine(line string) EvalInfo {
	var eval EvalInfo
	tokens := strings.Fields(line)

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t {
		case "depth":
			if i+1 < len(tokens) {
				eval.Depth, _ = strconv.Atoi(tokens[i+1])
			}
		case "score":
			if i+2 < len(tokens) {
				typ := tokens[i+1]
				val, _ := strconv.Atoi(tokens[i+2])
				if typ == "cp" {
					eval.ScoreCp = val
				} else if typ == "mate" {
					eval.MateIn = val
				}
			}
		case "nps":
			if i+1 < len(tokens) {
				eval.NPS, _ = strconv.ParseInt(tokens[i+1], 10, 64)
			}
		case "nodes":
			if i+1 < len(tokens) {
				eval.Nodes, _ = strconv.ParseInt(tokens[i+1], 10, 64)
			}
		case "pv":
			eval.PV = tokens[i+1:]
			return eval
		}
	}

	return eval
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
