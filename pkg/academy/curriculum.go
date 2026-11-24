package academy

// LessonArrow draws a visual guide arrow on the interactive lesson board.
type LessonArrow struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Color string `json:"color"`
}

// LessonStep is one interactive board checkpoint inside a University lesson.
type LessonStep struct {
	FEN             string        `json:"fen"`
	Title           string        `json:"title"`
	Instruction     string        `json:"instruction"`
	ExpectedMoves   []string      `json:"expectedMoves,omitempty"` // UCI moves; empty = demo/read step
	Hint            string        `json:"hint,omitempty"`
	SuccessFeedback string        `json:"successFeedback,omitempty"`
	Arrows          []LessonArrow `json:"arrows,omitempty"`
	Highlights      []string      `json:"highlights,omitempty"`
}

// Lesson is a structured, interactive Chess University module tied to a Career Tier.
type Lesson struct {
	ID               string       `json:"id"`
	Tier             int          `json:"tier"`
	Order            int          `json:"order"`
	Title            string       `json:"title"`
	Category         string       `json:"category"` // "tactics", "endgame", "opening", "strategy", "defense"
	EstimatedMinutes int          `json:"estimatedMinutes"`
	Summary          string       `json:"summary"`
	Steps            []LessonStep `json:"steps"`
}

// Curriculum returns the complete 18-lesson Chess University curriculum (3 per tier).
func Curriculum() []Lesson {
	return []Lesson{
		// ==================== TIER 1: COFFEEHOUSE ====================
		{
			ID: "t1_piece_safety", Tier: 1, Order: 1, Title: "Hanging Pieces & Counting Defenders",
			Category: "tactics", EstimatedMinutes: 4,
			Summary: "Before every move, scan for undefended pieces. Winning free material is the fastest jump from 700 to 1000.",
			Steps: []LessonStep{
				{
					FEN: "r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3",
					Title: "Check Who Defends e5",
					Instruction: "Black's knight on c6 defends e5. Before capturing anything, always count attackers vs. defenders.",
					Arrows: []LessonArrow{{From: "c6", To: "e5", Color: "#3b82f6"}, {From: "f3", To: "e5", Color: "#ef4444"}},
				},
				{
					FEN: "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
					Title: "Spot the Unprotected Target",
					Instruction: "Black played Nf6 but left e5 defended only once—Wait: c6 defends e5. Instead, play the classic Ng5 to attack the weak f7 pawn twice!",
					ExpectedMoves: []string{"f3g5"},
					Hint: "Move the f3 knight toward f7 so both the knight and c4 bishop aim at f7.",
					SuccessFeedback: "Exactly! Ng5 creates a double attack on f7, which is only defended by Black's king.",
					Arrows: []LessonArrow{{From: "c4", To: "f7", Color: "#10b981"}},
				},
			},
		},
		{
			ID: "t1_basic_checkmates", Tier: 1, Order: 2, Title: "Essential Checkmates (Queen & Rook)",
			Category: "endgame", EstimatedMinutes: 5,
			Summary: "Learn to finish winning endgames cleanly without stalemating your opponent.",
			Steps: []LessonStep{
				{
					FEN: "6k1/5Q2/6K1/8/8/8/8/8 w - - 0 1",
					Title: "King & Queen Mate in 1",
					Instruction: "Your king on g6 already guards f7, g7, and h7. Deliver checkmate in one move with the queen!",
					ExpectedMoves: []string{"f7g7", "f7e8", "f7f8"},
					Hint: "Bring the queen right in front of Black's king on g7, protected by your king on g6.",
					SuccessFeedback: "Checkmate! The queen cuts off every escape square while your king protects her.",
				},
				{
					FEN: "4k3/R7/4K3/8/8/8/8/8 w - - 0 1",
					Title: "King & Rook Mate in 1",
					Instruction: "The kings are in direct opposition (e6 vs e8). Deliver the decisive back-rank rook mate!",
					ExpectedMoves: []string{"a7a8"},
					Hint: "Swing the rook all the way to the 8th rank.",
					SuccessFeedback: "Clean checkmate! Your king covers d7, e7, and f7 while the rook sweeps the 8th rank.",
				},
			},
		},
		{
			ID: "t1_opening_principles", Tier: 1, Order: 3, Title: "Three Golden Opening Rules",
			Category: "opening", EstimatedMinutes: 4,
			Summary: "Control the center, develop knights and bishops quickly, and castle your king to safety.",
			Steps: []LessonStep{
				{
					FEN: "rnbqkbnr/pppp1ppp/8/4p2Q/4P3/8/PPPP1PPP/RNB1KBNR b KQkq - 1 2",
					Title: "Refuting Early Queen Raids",
					Instruction: "White brought the queen out on move 2 (Qh5), threatening Qxe5+. Don't panic and push g6 (which drops the rook on h8)! Defend e5 with a developing move.",
					ExpectedMoves: []string{"b8c6"},
					Hint: "Develop your queenside knight to defend the e5 pawn.",
					SuccessFeedback: "Perfect! Nc6 defends e5 with tempo and prepares to chase the exposed queen.",
				},
				{
					FEN: "r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
					Title: "Castle Early",
					Instruction: "Your kingside minor pieces are developed. Tuck your king into safety right now.",
					ExpectedMoves: []string{"e1g1"},
					Hint: "Castle kingside (O-O).",
					SuccessFeedback: "Well done! Your king is safe and your rook is ready to join the center.",
				},
			},
		},

		// ==================== TIER 2: CLUB FLOOR ====================
		{
			ID: "t2_forks_pins", Tier: 2, Order: 1, Title: "Forks & Pins Lab",
			Category: "tactics", EstimatedMinutes: 5,
			Summary: "Double attacks and absolute pins win more club games than any opening book.",
			Steps: []LessonStep{
				{
					FEN: "r3k2r/ppp2ppp/8/3N4/8/8/PPP2PPP/R3K2R w KQkq - 0 1",
					Title: "The Royal Fork",
					Instruction: "Find the knight move that simultaneously attacks Black's king and queenside rook.",
					ExpectedMoves: []string{"d5c7"},
					Hint: "Look at the c7 square.",
					SuccessFeedback: "Fork! Nxc7+ wins the rook on a8 next move.",
				},
				{
					FEN: "4k3/4r3/8/8/8/8/4B3/4R1K1 w - - 0 1",
					Title: "Pinning to the King",
					Instruction: "Black's rook on e7 is lined up in front of the king on e8. Attack the pinned rook with your bishop to win it!",
					ExpectedMoves: []string{"e2b5"},
					Hint: "Check the king from b5 or use the e-file pin—wait, Bb5+ is a check! Wait: let's pin or win the rook via Bb5+.",
					SuccessFeedback: "Bb5+ exploits the pin on the e-file so Black cannot block with the rook!",
				},
			},
		},
		{
			ID: "t2_back_rank", Tier: 2, Order: 2, Title: "Back-Rank Mate & Luft",
			Category: "tactics", EstimatedMinutes: 4,
			Summary: "Spot trapped kings behind their own pawn shield—and remember to create an escape square ('luft') when defending.",
			Steps: []LessonStep{
				{
					FEN: "6k1/5ppp/8/8/8/8/5PPP/3R2K1 w - - 0 1",
					Title: "Classic Back-Rank Checkmate",
					Instruction: "Black's pawns on f7, g7, and h7 form a prison wall. Deliver mate in one.",
					ExpectedMoves: []string{"d1d8"},
					Hint: "Send the rook straight to the 8th rank.",
					SuccessFeedback: "Checkmate! That's why creating luft (h6 or g6) is vital in heavy-piece endgames.",
				},
			},
		},
		{
			ID: "t2_opposition", Tier: 2, Order: 3, Title: "King & Pawn Opposition",
			Category: "endgame", EstimatedMinutes: 5,
			Summary: "In king-and-pawn endgames, taking direct opposition decides whether a single pawn promotes or draws.",
			Steps: []LessonStep{
				{
					FEN: "4k3/8/4K3/4P3/8/8/8/8 b - - 0 1",
					Title: "Understanding Opposition",
					Instruction: "When the kings face each other with one square between them, the side NOT to move holds the opposition. Here Black must give way.",
					Arrows: []LessonArrow{{From: "e6", To: "e8", Color: "#10b981"}},
				},
				{
					FEN: "4k3/8/8/4PK2/8/8/8/8 w - - 0 1",
					Title: "Seize the Key Square",
					Instruction: "Don't push e6 prematurely! Step your king ahead to the 6th rank (Ke6 or Kf6) to seize the key squares in front of the pawn.",
					ExpectedMoves: []string{"f5e6", "f5f6"},
					Hint: "Advance your king to e6 or f6 instead of pushing the pawn.",
					SuccessFeedback: "Great technique! Keeping your king in front of the pawn guarantees promotion.",
				},
			},
		},

		// ==================== TIER 3: CITY LEAGUE ====================
		{
			ID: "t3_discovered_attacks", Tier: 3, Order: 1, Title: "Discovered Checks & Attacks",
			Category: "tactics", EstimatedMinutes: 5,
			Summary: "Unmasking a hidden bishop or rook lets the moving piece capture or attack freely.",
			Steps: []LessonStep{
				{
					FEN: "4k3/4r3/8/3N4/8/8/8/4R1K1 w - - 0 1",
					Title: "Unmask the Rook",
					Instruction: "White's rook on e1 already eyes Black's rook on e7. Play a knight move that wins the rook cleanly!",
					ExpectedMoves: []string{"d5e7", "d5f6"},
					Hint: "Capture on e7 or jump to f6 with check to win the rook.",
					SuccessFeedback: "Sharp! Exploiting the e-file alignment wins decisive material.",
				},
			},
		},
		{
			ID: "t3_outposts", Tier: 3, Order: 2, Title: "Dominant Knight Outposts",
			Category: "strategy", EstimatedMinutes: 5,
			Summary: "A knight anchored on a central square that no enemy pawn can attack is often stronger than a rook.",
			Steps: []LessonStep{
				{
					FEN: "r1bq1rk1/pp2bppp/2n1pn2/3p4/2PP4/2N2NP1/PP2PPBP/R1BQ1RK1 w - - 0 8",
					Title: "Spot the Weak Light Squares",
					Instruction: "Look for squares in the opponent's camp that can no longer be guarded by pawns. Plant knights on supported central squares like e5 or d5.",
					Highlights: []string{"e5", "c5"},
				},
				{
					FEN: "r2q1rk1/pp3ppp/2nbpn2/3p4/3P4/2N1PN2/PP2BPPP/R2Q1RK1 w - - 0 1",
					Title: "Claim the e5 Outpost",
					Instruction: "Jump your knight into the central outpost on e5, supported by your d4 pawn.",
					ExpectedMoves: []string{"f3e5"},
					Hint: "Move the f3 knight to e5.",
					SuccessFeedback: "Excellent! From e5, your knight radiates pressure across both wings.",
				},
			},
		},
		{
			ID: "t3_pawn_structures", Tier: 3, Order: 3, Title: "Pawn Breaks & Central Tension",
			Category: "strategy", EstimatedMinutes: 5,
			Summary: "Pawns determine the plan. Use timely pawn breaks (c4, d4, e5, f5) to open files for your rooks.",
			Steps: []LessonStep{
				{
					FEN: "rnbqkb1r/pp3ppp/2p1pn2/3p4/3PP3/2N5/PPP2PPP/R1BQKBNR w KQkq - 0 5",
					Title: "Gain Space with e5",
					Instruction: "Black's knight on f6 is vulnerable to a central space grab. Push your e-pawn to drive the knight away and claim kingside space.",
					ExpectedMoves: []string{"e4e5"},
					Hint: "Advance e4 to e5.",
					SuccessFeedback: "Strong! e5 cramps Black's kingside and fixes a clear attacking plan.",
				},
			},
		},

		// ==================== TIER 4: REGIONAL OPEN ====================
		{
			ID: "t4_lucena_philidor", Tier: 4, Order: 1, Title: "Lucena & Philidor Rook Endgames",
			Category: "endgame", EstimatedMinutes: 6,
			Summary: "The two most important rook endgames in chess: building the bridge (Lucena) and 3rd-rank defense (Philidor).",
			Steps: []LessonStep{
				{
					FEN: "1K6/1P6/8/8/8/8/r7/2R2k2 w - - 0 1",
					Title: "Building the Lucena Bridge",
					Instruction: "White's king is trapped in front of the b7 pawn. Lift your rook to the 4th rank (Rc4) so it can shield the king from vertical checks!",
					ExpectedMoves: []string{"c1c4"},
					Hint: "Move the rook to c4 to prepare the 'bridge' on the 4th rank.",
					SuccessFeedback: "Textbook Lucena! Next you step the king out and block checks with the rook on b4.",
				},
				{
					FEN: "8/8/4k3/4P3/4K3/8/r7/7R b - - 0 1",
					Title: "Philidor 3rd-Rank Defense",
					Instruction: "As Black, keep White's king off the 6th rank! Place your rook on the 6th rank (your 3rd rank) right now.",
					ExpectedMoves: []string{"a2a6"},
					Hint: "Swing the rook to a6.",
					SuccessFeedback: "Spot on! Once White pushes e6, you drop the rook to a1 for unstoppable checks from behind.",
				},
			},
		},
		{
			ID: "t4_sacrificial_attacks", Tier: 4, Order: 2, Title: "The Greek Gift Sacrifice (Bxh7+)",
			Category: "tactics", EstimatedMinutes: 6,
			Summary: "When Black's f6 knight is absent and your knight + queen can jump to g5 and h5, Bxh7+ tears open the king.",
			Steps: []LessonStep{
				{
					FEN: "r1bq1rk1/ppp2ppp/2n1p3/3pP3/3P4/2PB1N2/PP3PPP/R1BQK2R w KQ - 0 9",
					Title: "Launch the Classic Bxh7+ Sacrifice",
					Instruction: "Notice Black has no knight on f6 to defend h7, and your e5 pawn blocks defenders. Play the Greek Gift sacrifice!",
					ExpectedMoves: []string{"d3h7"},
					Hint: "Sacrifice the light-squared bishop on h7.",
					SuccessFeedback: "Bxh7+! After Kxh7 Ng5+ Kg8 Qh5, Black's king is under a ferocious attack.",
				},
			},
		},
		{
			ID: "t4_bishop_vs_knight", Tier: 4, Order: 3, Title: "Good Bishop vs. Bad Knight",
			Category: "strategy", EstimatedMinutes: 5,
			Summary: "Bishops dominate open positions with pawns on both wings; knights thrive in closed, blocked centers.",
			Steps: []LessonStep{
				{
					FEN: "8/pp3pkp/6p1/8/8/2B5/PP3PPP/6K1 w - - 0 1",
					Title: "Activate the Long-Range Bishop",
					Instruction: "With pawns on both sides of the board, a bishop slices across the whole board. Centralize your king to support your active bishop.",
					ExpectedMoves: []string{"g1f1", "c3g7"},
					Hint: "Bring your king toward the center (Kf1) or liquidate if winning.",
					SuccessFeedback: "Good! In open endgames, pair your active bishop with a centralized king.",
				},
			},
		},

		// ==================== TIER 5: CANDIDATE MASTER ====================
		{
			ID: "t5_prophylaxis", Tier: 5, Order: 1, Title: "Prophylaxis: Stop Opponent's Plan First",
			Category: "strategy", EstimatedMinutes: 6,
			Summary: "Before executing your own plan, ask: 'What does my opponent want to play if it were their turn?'",
			Steps: []LessonStep{
				{
					FEN: "r1bq1rk1/pp1nbppp/2p1pn2/3p4/2PP4/2N1PN2/PP2BPPP/R1BQ1RK1 w - - 0 8",
					Title: "Ask What Black Wants",
					Instruction: "In closed structures, Black often wants to free their position with c5 or e5. Develop harmoniously (Qc2 or b3) to keep control over e4 and c5.",
					ExpectedMoves: []string{"d1c2", "b2b3"},
					Hint: "Play Qc2 to watch e4 and prepare e4 yourself, or b3 to fianchetto.",
					SuccessFeedback: "Prophylactic mastery! Qc2/b3 improves your pieces while restraining Black's freeing breaks.",
				},
			},
		},
		{
			ID: "t5_minority_attack", Tier: 5, Order: 2, Title: "The Queenside Minority Attack",
			Category: "strategy", EstimatedMinutes: 6,
			Summary: "In the Carlsbad structure, march your b-pawn to b5 to create a permanent backward c6 pawn weakness.",
			Steps: []LessonStep{
				{
					FEN: "r2q1rk1/pp3ppp/2p1pn2/3p4/1P1P4/2N1PN2/P4PPP/1RBQ1RK1 w - - 0 12",
					Title: "Strike with b4-b5",
					Instruction: "White has fewer pawns on the queenside (a+b vs a+b+c), yet advancing b5 breaks up Black's structure and leaves c6 weak.",
					ExpectedMoves: []string{"b4b5"},
					Hint: "Push the b-pawn to b5.",
					SuccessFeedback: "Classic Carlsbad minority attack! After bxc6, Black is saddled with long-term structural targets.",
				},
			},
		},
		{
			ID: "t5_exchange_sac", Tier: 5, Order: 3, Title: "Positional Exchange Sacrifices",
			Category: "strategy", EstimatedMinutes: 6,
			Summary: "Sometimes a dominant minor piece plus a ruined enemy king shelter is worth more than a rook.",
			Steps: []LessonStep{
				{
					FEN: "2r2rk1/pp2bppp/4pn2/q2p4/3P4/2NBPN2/PP3PPP/2RQ1RK1 b - - 0 12",
					Title: "The Sicilian Rxc3 Theme",
					Instruction: "When a knight on c3 holds White's entire queenside together, sacrificing Rxc3 shatters White's pawns and hands Black dark-square control.",
					Arrows: []LessonArrow{{From: "c8", To: "c3", Color: "#eab308"}},
				},
			},
		},

		// ==================== TIER 6: GRANDMASTER HALL ====================
		{
			ID: "t6_deep_calculation", Tier: 6, Order: 1, Title: "Candidate Moves & Forcing Lines",
			Category: "tactics", EstimatedMinutes: 6,
			Summary: "Always calculate Checks, Captures, and Threats (CCT) first before evaluating quiet moves.",
			Steps: []LessonStep{
				{
					FEN: "r4rk1/ppp2ppp/8/3q4/8/2N5/PPP2PPP/R2Q1RK1 w - - 0 1",
					Title: "Spot the Tactical Resource",
					Instruction: "Black's queen on d5 is exposed. Capture the queen cleanly with your knight (or queen) to enter a winning endgame.",
					ExpectedMoves: []string{"c3d5", "d1d5"},
					Hint: "Both Nxd5 and Qxd5 win the queen—which leaves your pieces better placed?",
					SuccessFeedback: "Clean calculation! Winning the exposed queen immediately decides the game.",
				},
			},
		},
		{
			ID: "t6_technical_conversion", Tier: 6, Order: 2, Title: "Converting Extra Pawns Without Counterplay",
			Category: "endgame", EstimatedMinutes: 6,
			Summary: "When ahead in material, neutralize opponent counterplay first—do not rush pawn pushes that open lines for enemy rooks.",
			Steps: []LessonStep{
				{
					FEN: "8/5pk1/6p1/8/3P4/5K2/5PP1/8 w - - 0 1",
					Title: "Centralize Before Pushing",
					Instruction: "You are a clean pawn up. Bring your king toward the center (Ke4) before committing your passed d-pawn.",
					ExpectedMoves: []string{"f3e4", "f3f4"},
					Hint: "Step the king forward to e4.",
					SuccessFeedback: "Textbook technical play! An active king escorts the passed pawn home effortlessly.",
				},
			},
		},
		{
			ID: "t6_fortress_defense", Tier: 6, Order: 3, Title: "Building an Endgame Fortress",
			Category: "defense", EstimatedMinutes: 6,
			Summary: "Even down material, a well-constructed fortress can hold a draw against a much stronger opponent.",
			Steps: []LessonStep{
				{
					FEN: "8/8/8/8/8/5k2/6p1/6K1 w - - 0 1",
					Title: "Stalemate & Opposition Rescue",
					Instruction: "Remember: in Rook+Bishop vs Rook or wrong-colored bishop + rook-pawn endgames, tucking your king into the corner square (h1/a8) creates an unbreakable fortress.",
					Highlights: []string{"g1", "h1"},
				},
			},
		},
	}
}

// FindLesson looks up a lesson by ID.
func FindLesson(id string) (Lesson, bool) {
	for _, l := range Curriculum() {
		if l.ID == id {
			return l, true
		}
	}
	return Lesson{}, false
}

// LessonsForTier returns all University lessons belonging to a given tier (1..6).
func LessonsForTier(tier int) []Lesson {
	var out []Lesson
	for _, l := range Curriculum() {
		if l.Tier == tier {
			out = append(out, l)
		}
	}
	return out
}
