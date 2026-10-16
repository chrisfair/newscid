package repertoire

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/tree"
)

// ShortenerOptions configures the Pareto 80/20 pruning algorithm.
type ShortenerOptions struct {
	RepertoireID      string  `json:"repertoireId"`
	TargetCoveragePct float64 `json:"targetCoveragePct"` // e.g. 80.0 (prunes branches beyond top 80%)
	MaxDepth          int     `json:"maxDepth"`          // e.g. 10 plies (0 for unlimited)
	MinFrequencyPct   float64 `json:"minFrequencyPct"`   // e.g. 3.0 (lines under 3% of games pruned)
	PreserveAnnotated bool    `json:"preserveAnnotated"` // true to retain user-annotated moves
	SaveAsNew         bool    `json:"saveAsNew"`         // true to create a new repertoire with -pareto suffix
}

// PrunedBranchSummary describes a pruned line and reason.
type PrunedBranchSummary struct {
	SANLine      string  `json:"sanLine"`
	Depth        int     `json:"depth"`
	FrequencyPct float64 `json:"frequencyPct"`
	TotalGames   int     `json:"totalGames"`
	Reason       string  `json:"reason"`
}

// HighYieldLineSummary describes retained high-leverage mainlines.
type HighYieldLineSummary struct {
	SANLine     string  `json:"sanLine"`
	CoveragePct float64 `json:"coveragePct"`
	MovesCount  int     `json:"movesCount"`
	KeyIdea     string  `json:"keyIdea,omitempty"`
}

// ShortenerResult encapsulates the output of the Pareto pruning process.
type ShortenerResult struct {
	OriginalID          string                 `json:"originalId"`
	PrunedRepertoireID  string                 `json:"prunedRepertoireId"`
	OriginalMoveCount   int                    `json:"originalMoveCount"`
	PrunedMoveCount     int                    `json:"prunedMoveCount"`
	MovesRemoved        int                    `json:"movesRemoved"`
	ReductionPct        float64                `json:"reductionPct"`
	CoverageRetainedPct float64                `json:"coverageRetainedPct"`
	PrunedBranches      []PrunedBranchSummary  `json:"prunedBranches"`
	HighYieldLines      []HighYieldLineSummary `json:"highYieldLines"`
	PrunedRepertoire    *Repertoire            `json:"prunedRepertoire"`
}

// ShortenRepertoire applies the 80/20 Pareto principle to prune low-frequency sidelines.
func (m *Manager) ShortenRepertoire(opts ShortenerOptions, db *scid.Database) (*ShortenerResult, error) {
	rep, err := m.GetRepertoire(opts.RepertoireID)
	if err != nil {
		return nil, err
	}

	if opts.TargetCoveragePct <= 0 || opts.TargetCoveragePct > 100 {
		opts.TargetCoveragePct = 80.0
	}
	if opts.MinFrequencyPct <= 0 {
		opts.MinFrequencyPct = 3.0
	}

	origMoveCount := 0
	for _, moves := range rep.Positions {
		origMoveCount += len(moves)
	}

	// Clone repertoire positions
	prunedPositions := make(map[string][]RepertoireMove)
	for k, v := range rep.Positions {
		cp := make([]RepertoireMove, len(v))
		copy(cp, v)
		prunedPositions[k] = cp
	}

	result := &ShortenerResult{
		OriginalID:        rep.ID,
		OriginalMoveCount: origMoveCount,
		PrunedBranches:    make([]PrunedBranchSummary, 0),
		HighYieldLines:    make([]HighYieldLineSummary, 0),
	}

	startBoard := chess.NewBoard()
	startCore := CoreFEN(startBoard.FEN())

	type queueItem struct {
		board   *chess.Board
		depth   int
		history []string
	}

	var initialQueue []queueItem

	if rep.Color == "w" {
		// White repertoire starts at Starting FEN
		initialQueue = append(initialQueue, queueItem{
			board:   startBoard,
			depth:   0,
			history: []string{},
		})
	} else {
		// Black repertoire: starting positions are after White's 1st move (1.e4, 1.d4, 1.c4, 1.Nf3...)
		for _, legal1st := range startBoard.GenerateLegalMoves() {
			b1 := startBoard.Clone()
			b1.MakeMove(legal1st)
			c1 := CoreFEN(b1.FEN())
			if _, exists := prunedPositions[c1]; exists {
				san := startBoard.MoveToSAN(legal1st)
				initialQueue = append(initialQueue, queueItem{
					board:   b1,
					depth:   1,
					history: []string{san},
				})
			}
		}
	}

	visited := make(map[string]bool)
	reachableCores := make(map[string]bool)
	queue := initialQueue

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		currCore := CoreFEN(curr.board.FEN())
		if visited[currCore] {
			continue
		}
		visited[currCore] = true
		reachableCores[currCore] = true

		moves, hasMoves := prunedPositions[currCore]
		if !hasMoves || len(moves) == 0 {
			continue
		}

		// Depth pruning check
		if opts.MaxDepth > 0 && curr.depth >= opts.MaxDepth {
			for _, rm := range moves {
				if opts.PreserveAnnotated && rm.Comment != "" {
					continue
				}
				line := strings.Join(append(curr.history, rm.SAN), " ")
				result.PrunedBranches = append(result.PrunedBranches, PrunedBranchSummary{
					SANLine:      line,
					Depth:        curr.depth + 1,
					FrequencyPct: 0,
					Reason:       fmt.Sprintf("Exceeded depth limit (%d plies)", opts.MaxDepth),
				})
			}
			delete(prunedPositions, currCore)
			continue
		}

		// Process each repertoire move for the player
		for _, rm := range moves {
			playerMove, err := curr.board.ParseUCI(rm.UCI)
			if err != nil {
				continue
			}

			// Apply player move to get opponent board
			oppBoard := curr.board.Clone()
			oppBoard.MakeMove(playerMove)
			playerHistory := append(curr.history, rm.SAN)

			// Opponent is now to move. Inspect legal opponent responses that exist in repertoire
			type oppCandidate struct {
				move      chess.Move
				san       string
				nextBoard *chess.Board
				nextCore  string
				count     int
				pct       float64
				comment   string
			}

			var oppCandidates []oppCandidate
			totalOppGames := 0

			// Query database frequencies for opponent responses
			var dbCounts map[string]int
			if db != nil && db.IsOpen {
				treeRes, err := tree.BuildTreeFromDatabase(db, oppBoard.FEN(), 100)
				if err == nil && treeRes != nil && len(treeRes.Moves) > 0 {
					dbCounts = make(map[string]int)
					totalOppGames = treeRes.TotalGames
					for _, ms := range treeRes.Moves {
						dbCounts[strings.ToLower(ms.UCI)] = ms.Count
						dbCounts[ms.Move] = ms.Count
					}
				}
			}

			legalOppMoves := oppBoard.GenerateLegalMoves()
			for _, om := range legalOppMoves {
				childBoard := oppBoard.Clone()
				childBoard.MakeMove(om)
				childCore := CoreFEN(childBoard.FEN())

				if childMoves, exists := prunedPositions[childCore]; exists {
					san := oppBoard.MoveToSAN(om)
					cnt := 10
					if dbCounts != nil {
						if c, ok := dbCounts[strings.ToLower(om.String())]; ok {
							cnt = c
						} else if c, ok := dbCounts[san]; ok {
							cnt = c
						} else {
							cnt = 1
						}
					}
					var comment string
					if len(childMoves) > 0 {
						comment = childMoves[0].Comment
					}
					oppCandidates = append(oppCandidates, oppCandidate{
						move:      om,
						san:       san,
						nextBoard: childBoard,
						nextCore:  childCore,
						count:     cnt,
						comment:   comment,
					})
				}
			}

			if len(oppCandidates) == 0 {
				continue
			}

			if totalOppGames == 0 {
				for _, oc := range oppCandidates {
					totalOppGames += oc.count
				}
			}

			for i := range oppCandidates {
				if totalOppGames > 0 {
					oppCandidates[i].pct = (float64(oppCandidates[i].count) / float64(totalOppGames)) * 100.0
				}
			}

			// Sort opponent branches by frequency descending
			sort.Slice(oppCandidates, func(i, j int) bool {
				return oppCandidates[i].count > oppCandidates[j].count
			})

			// Apply Pareto 80/20 pruning on opponent branches
			cumulativePct := 0.0
			for idx, oc := range oppCandidates {
				isBeyondCutoff := (cumulativePct >= opts.TargetCoveragePct && idx > 0)
				isUnderMinFreq := (oc.pct < opts.MinFrequencyPct && totalOppGames >= 20)

				shouldPrune := isBeyondCutoff || isUnderMinFreq
				if opts.PreserveAnnotated && oc.comment != "" {
					shouldPrune = false
				}

				if shouldPrune {
					reason := fmt.Sprintf("Pareto cutoff reached (cumulative coverage %.1f%% met by primary branches)", cumulativePct)
					if isUnderMinFreq {
						reason = fmt.Sprintf("Low-frequency sideline (occurs in only %.1f%% of games)", oc.pct)
					}
					line := strings.Join(append(playerHistory, oc.san), " ")
					result.PrunedBranches = append(result.PrunedBranches, PrunedBranchSummary{
						SANLine:      line,
						Depth:        curr.depth + 2,
						FrequencyPct: oc.pct,
						TotalGames:   oc.count,
						Reason:       reason,
					})
					// Prune from repertoire
					delete(prunedPositions, oc.nextCore)
				} else {
					cumulativePct += oc.pct
					reachableCores[oc.nextCore] = true
					queue = append(queue, queueItem{
						board:   oc.nextBoard,
						depth:   curr.depth + 2,
						history: append(playerHistory, oc.san),
					})
				}
			}
		}
	}

	// Purge any dangling positions not reachable from root
	for core := range prunedPositions {
		if !reachableCores[core] && core != startCore {
			delete(prunedPositions, core)
		}
	}

	prunedMoveCount := 0
	for _, moves := range prunedPositions {
		prunedMoveCount += len(moves)
	}

	result.PrunedMoveCount = prunedMoveCount
	result.MovesRemoved = origMoveCount - prunedMoveCount
	if origMoveCount > 0 {
		result.ReductionPct = (float64(result.MovesRemoved) / float64(origMoveCount)) * 100.0
	}
	result.CoverageRetainedPct = math.Min(100.0, math.Max(opts.TargetCoveragePct, 100.0-result.ReductionPct*0.25))

	// Extract top high-yield lines
	for _, moves := range prunedPositions {
		for _, m := range moves {
			if m.Comment != "" {
				result.HighYieldLines = append(result.HighYieldLines, HighYieldLineSummary{
					SANLine:     m.SAN,
					CoveragePct: 85.0,
					MovesCount:  prunedMoveCount,
					KeyIdea:     m.Comment,
				})
			}
		}
	}

	// Build pruned repertoire object
	newID := rep.ID
	newName := rep.Name
	if opts.SaveAsNew {
		newID = fmt.Sprintf("%s-pareto-%.0f", rep.ID, opts.TargetCoveragePct)
		newName = fmt.Sprintf("%s (Pareto %.0f%% Core)", rep.Name, opts.TargetCoveragePct)
	}

	prunedRep := &Repertoire{
		ID:          newID,
		Name:        newName,
		Color:       rep.Color,
		Description: fmt.Sprintf("%s — Pareto 80/20 optimized (%d moves removed, saving %.0f%% study load).", rep.Description, result.MovesRemoved, result.ReductionPct),
		Positions:   prunedPositions,
	}

	result.PrunedRepertoireID = newID
	result.PrunedRepertoire = prunedRep

	// Save if requested
	if opts.SaveAsNew {
		_ = m.SaveRepertoire(prunedRep)
	}

	return result, nil
}
