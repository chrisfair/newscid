package academy

// PersonalitySliders tunes how a personality selects moves from engine Multi-PV lines
// and opening books. All values are on a 0..100 scale.
type PersonalitySliders struct {
	Aggression         int `json:"aggression"`         // 0=quiet, 100=all-out attack/sacrifices
	TradeTendency      int `json:"tradeTendency"`      // 0=keep pieces on, 100=trade down to endgame
	TacticalSharpness  int `json:"tacticalSharpness"`  // 0=misses tactics, 100=sharp calculator
	PositionalSolidity int `json:"positionalSolidity"` // 0=chaotic, 100=rock-solid structure
	BlunderRate        int `json:"blunderRate"`        // 0=grandmaster accuracy, 100=frequent oversights
}

// Personality defines a playable opponent in the NewScid Academy & Career Ladder.
type Personality struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Title              string              `json:"title"`
	Tier               int                 `json:"tier"` // 1..6
	IsBoss             bool                `json:"isBoss"`
	Elo                int                 `json:"elo"`
	AvatarColor        string              `json:"avatarColor"`
	AvatarSymbol       string              `json:"avatarSymbol"`
	StyleTag           string              `json:"styleTag"`
	Bio                string              `json:"bio"`
	Source             string              `json:"source"` // "builtin", "custom", "database"
	SourcePlayer       string              `json:"sourcePlayer,omitempty"`
	SourceGames        int                 `json:"sourceGames,omitempty"`
	Sliders            PersonalitySliders  `json:"sliders"`
	OpeningBook        map[string][]string `json:"openingBook,omitempty"` // board-FEN -> preferred UCI moves
	RecommendedLessons []string            `json:"recommendedLessons,omitempty"`
}

// TierInfo describes one of the 6 Academy & Career tiers.
type TierInfo struct {
	Tier        int    `json:"tier"`
	Name        string `json:"name"`
	RatingRange string `json:"ratingRange"`
	TargetElo   int    `json:"targetElo"`
	Description string `json:"description"`
	BossID      string `json:"bossId"`
}

// Tiers returns the 6 progressive Career & University tiers.
func Tiers() []TierInfo {
	return []TierInfo{
		{
			Tier:        1,
			Name:        "Coffeehouse",
			RatingRange: "700–1000",
			TargetElo:   950,
			Description: "Build clean habits: stop hanging pieces, finish basic checkmates, and develop smoothly.",
			BossID:      "t1_milo_boss",
		},
		{
			Tier:        2,
			Name:        "Club Floor",
			RatingRange: "1000–1300",
			TargetElo:   1250,
			Description: "Master core tactical motifs (forks, pins, back-rank mates) and refute cheap opening traps.",
			BossID:      "t2_vera_boss",
		},
		{
			Tier:        3,
			Name:        "City League",
			RatingRange: "1300–1600",
			TargetElo:   1550,
			Description: "Turn tactical vision into strategy: outposts, discovered attacks, and active rook endgames.",
			BossID:      "t3_marcus_boss",
		},
		{
			Tier:        4,
			Name:        "Regional Open",
			RatingRange: "1600–1900",
			TargetElo:   1880,
			Description: "Face classic stylistic archetypes and sharpen technical endgame conversion (Lucena & Philidor).",
			BossID:      "t4_elena_boss",
		},
		{
			Tier:        5,
			Name:        "Candidate Master",
			RatingRange: "1900–2200",
			TargetElo:   2180,
			Description: "Master prophylaxis, minority attacks, and complex middlegame imbalances against strong masters.",
			BossID:      "t5_viktor_boss",
		},
		{
			Tier:        6,
			Name:        "Grandmaster Hall",
			RatingRange: "2200–2600+",
			TargetElo:   2550,
			Description: "Test your full repertoire against historic grandmaster styles and top engine strength.",
			BossID:      "t6_grandmaster_boss",
		},
	}
}

const startBoardFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"

// BuiltinPersonalities returns the 24 original and public-domain historical-style personalities.
func BuiltinPersonalities() []Personality {
	return []Personality{
		// ==================== TIER 1: COFFEEHOUSE (700-1000) ====================
		{
			ID: "t1_toby", Name: "Toby", Title: "Pawn Pusher", Tier: 1, Elo: 750,
			AvatarColor: "#f59e0b", AvatarSymbol: "♙", StyleTag: "Overextends Pawns",
			Bio:     "Loves pushing flank pawns early and forgets to castle. Great warm-up for center control and piece safety.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 65, TradeTendency: 35, TacticalSharpness: 18, PositionalSolidity: 20, BlunderRate: 78},
			OpeningBook: map[string][]string{
				startBoardFEN: {"a2a4", "h2h4", "e2e4"},
			},
			RecommendedLessons: []string{"t1_piece_safety", "t1_opening_principles"},
		},
		{
			ID: "t1_clara", Name: "Clara", Title: "Early Queen Raider", Tier: 1, Elo: 820,
			AvatarColor: "#ec4899", AvatarSymbol: "♕", StyleTag: "Scholar's Mate Fan",
			Bio:     "Brings her queen out on move 2 looking for quick tricks. Punish her with calm development.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 82, TradeTendency: 20, TacticalSharpness: 25, PositionalSolidity: 25, BlunderRate: 68},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4"},
				"rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR": {"d1h5", "f1c4"},
			},
			RecommendedLessons: []string{"t1_opening_principles", "t1_piece_safety"},
		},
		{
			ID: "t1_ben", Name: "Ben", Title: "Happy Trader", Tier: 1, Elo: 900,
			AvatarColor: "#14b8a6", AvatarSymbol: "♘", StyleTag: "Trades Everything",
			Bio:     "Offers piece trades at every opportunity hoping to reach a simple endgame. Practice basic checkmates against him.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 25, TradeTendency: 92, TacticalSharpness: 32, PositionalSolidity: 45, BlunderRate: 58},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4", "e2e4"},
			},
			RecommendedLessons: []string{"t1_basic_checkmates", "t1_piece_safety"},
		},
		{
			ID: "t1_milo_boss", Name: "Old Man Milo", Title: "Tier 1 Boss", Tier: 1, IsBoss: true, Elo: 980,
			AvatarColor: "#d97706", AvatarSymbol: "♔", StyleTag: "Coffeehouse Veteran",
			Bio:     "Has played at the corner park table for forty years. Plays solid classical openings and waits for you to drop a piece.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 48, TradeTendency: 50, TacticalSharpness: 42, PositionalSolidity: 55, BlunderRate: 45},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4"},
			},
			RecommendedLessons: []string{"t1_piece_safety", "t1_basic_checkmates", "t1_opening_principles"},
		},

		// ==================== TIER 2: CLUB FLOOR (1000-1300) ====================
		{
			ID: "t2_samir", Name: "Samir", Title: "Gambit Swindler", Tier: 2, Elo: 1080,
			AvatarColor: "#ef4444", AvatarSymbol: "⚔", StyleTag: "Tactical Gambiteer",
			Bio:     "Throws pawns forward for open lines and loves knight forks on f7. Stay alert for tactics!",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 88, TradeTendency: 25, TacticalSharpness: 55, PositionalSolidity: 38, BlunderRate: 42},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4"},
				"rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR": {"f2f4", "g1f3"},
			},
			RecommendedLessons: []string{"t2_forks_pins", "t2_back_rank"},
		},
		{
			ID: "t2_hannah", Name: "Hannah", Title: "Italian Tactician", Tier: 2, Elo: 1150,
			AvatarColor: "#3b82f6", AvatarSymbol: "♗", StyleTag: "Classical Open Games",
			Bio:     "Develops rapidly with Bc4 and Nf3, targeting f7 and pinning knights on f6.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 68, TradeTendency: 45, TacticalSharpness: 58, PositionalSolidity: 58, BlunderRate: 36},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4"},
				"r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R": {"f1c4"},
			},
			RecommendedLessons: []string{"t2_forks_pins", "t2_opposition"},
		},
		{
			ID: "t2_derek", Name: "Derek", Title: "Back-Rank Hunter", Tier: 2, Elo: 1210,
			AvatarColor: "#8b5cf6", AvatarSymbol: "♖", StyleTag: "Heavy Piece Pressure",
			Bio:     "Doubles rooks quickly and punishes any unguarded back rank or sloppy pawn endgame.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 58, TradeTendency: 65, TacticalSharpness: 60, PositionalSolidity: 62, BlunderRate: 32},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4", "c2c4"},
			},
			RecommendedLessons: []string{"t2_back_rank", "t2_opposition"},
		},
		{
			ID: "t2_vera_boss", Name: "Coach Vera", Title: "Tier 2 Boss", Tier: 2, IsBoss: true, Elo: 1280,
			AvatarColor: "#0ea5e9", AvatarSymbol: "♛", StyleTag: "Disciplined Club Captain",
			Bio:     "Rarely blunders outright and tests whether you spot pins, forks, and king-and-pawn opposition.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 55, TradeTendency: 55, TacticalSharpness: 66, PositionalSolidity: 70, BlunderRate: 26},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4"},
			},
			RecommendedLessons: []string{"t2_forks_pins", "t2_back_rank", "t2_opposition"},
		},

		// ==================== TIER 3: CITY LEAGUE (1300-1600) ====================
		{
			ID: "t3_nina", Name: "Nina", Title: "London System Specialist", Tier: 3, Elo: 1380,
			AvatarColor: "#10b981", AvatarSymbol: "♝", StyleTag: "Solid & Patient",
			Bio:     "Builds a rock-solid d4/Bf4 pyramid and grinds down impatient attacks.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 42, TradeTendency: 60, TacticalSharpness: 66, PositionalSolidity: 82, BlunderRate: 24},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4"},
				"rnbqkbnr/ppp1pppp/8/3p4/3P4/8/PPP1PPPP/RNBQKBNR": {"c1f4", "g1f3"},
			},
			RecommendedLessons: []string{"t3_outposts", "t3_pawn_structures"},
		},
		{
			ID: "t3_carlos", Name: "Carlos", Title: "Sicilian Dragon Rider", Tier: 3, Elo: 1460,
			AvatarColor: "#f97316", AvatarSymbol: "♞", StyleTag: "Opposite-Side Castling",
			Bio:     "Plays sharp Sicilian structures and launches pawn storms whenever kings castle on opposite wings.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 86, TradeTendency: 28, TacticalSharpness: 74, PositionalSolidity: 58, BlunderRate: 22},
			OpeningBook: map[string][]string{
				"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR": {"c7c5"},
			},
			RecommendedLessons: []string{"t3_discovered_attacks", "t3_pawn_structures"},
		},
		{
			ID: "t3_priya", Name: "Priya", Title: "Rook Endgame Technician", Tier: 3, Elo: 1510,
			AvatarColor: "#06b6d4", AvatarSymbol: "♜", StyleTag: "Active Rooks",
			Bio:     "Happily simplifies into rook endgames and activates her rook behind passed pawns.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 45, TradeTendency: 78, TacticalSharpness: 70, PositionalSolidity: 80, BlunderRate: 20},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4", "c2c4"},
			},
			RecommendedLessons: []string{"t3_outposts", "t4_lucena_philidor"},
		},
		{
			ID: "t3_marcus_boss", Name: "Marcus", Title: "Tier 3 Boss", Tier: 3, IsBoss: true, Elo: 1580,
			AvatarColor: "#6366f1", AvatarSymbol: "♚", StyleTag: "City League Champion",
			Bio:     "Balanced, practical player who exploits weak squares and punishes loose pieces with discovered attacks.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 62, TradeTendency: 55, TacticalSharpness: 76, PositionalSolidity: 78, BlunderRate: 17},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4", "c2c4"},
			},
			RecommendedLessons: []string{"t3_discovered_attacks", "t3_outposts", "t3_pawn_structures"},
		},

		// ==================== TIER 4: REGIONAL OPEN (1600-1900) ====================
		{
			ID: "t4_morphy_style", Name: "Paul Morphy", Title: "Classic Romantic Style", Tier: 4, Elo: 1720,
			AvatarColor: "#dc2626", AvatarSymbol: "♛", StyleTag: "Rapid Development & Sacrifices",
			Bio:     "Inspired by 19th-century open-game brilliance: prioritizes lead in development, open files, and direct king attacks.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 95, TradeTendency: 22, TacticalSharpness: 86, PositionalSolidity: 72, BlunderRate: 14},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4"},
				"rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR": {"g1f3", "f2f4"},
			},
			RecommendedLessons: []string{"t4_sacrificial_attacks", "t4_bishop_vs_knight"},
		},
		{
			ID: "t4_capablanca_style", Name: "J.R. Capablanca", Title: "Classic Endgame Style", Tier: 4, Elo: 1810,
			AvatarColor: "#0284c7", AvatarSymbol: "♔", StyleTag: "Crystal Clarity & Endgames",
			Bio:     "Inspired by classic positional clarity: avoids unnecessary complications and converts tiny structural edges in the endgame.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 48, TradeTendency: 76, TacticalSharpness: 84, PositionalSolidity: 94, BlunderRate: 12},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4", "e2e4"},
			},
			RecommendedLessons: []string{"t4_lucena_philidor", "t4_bishop_vs_knight"},
		},
		{
			ID: "t4_kenji", Name: "Kenji", Title: "Dynamic Counterattacker", Tier: 4, Elo: 1840,
			AvatarColor: "#eab308", AvatarSymbol: "♞", StyleTag: "King's Indian & Counterplay",
			Bio:     "Concedes space early only to blow open the center with f5 or c5 breaks.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 80, TradeTendency: 35, TacticalSharpness: 84, PositionalSolidity: 76, BlunderRate: 12},
			OpeningBook: map[string][]string{
				"rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR": {"g8f6"},
			},
			RecommendedLessons: []string{"t4_sacrificial_attacks", "t4_lucena_philidor"},
		},
		{
			ID: "t4_elena_boss", Name: "Elena", Title: "Tier 4 Boss", Tier: 4, IsBoss: true, Elo: 1890,
			AvatarColor: "#7c3aed", AvatarSymbol: "♖", StyleTag: "Endgame Surgeon",
			Bio:     "Regional Open finalist who squeezes water from stone in rook and minor-piece endgames.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 58, TradeTendency: 72, TacticalSharpness: 86, PositionalSolidity: 88, BlunderRate: 10},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4", "c2c4", "g1f3"},
			},
			RecommendedLessons: []string{"t4_lucena_philidor", "t4_sacrificial_attacks", "t4_bishop_vs_knight"},
		},

		// ==================== TIER 5: CANDIDATE MASTER (1900-2200) ====================
		{
			ID: "t5_nimzowitsch_style", Name: "A. Nimzowitsch", Title: "Hypermodern Style", Tier: 5, Elo: 2020,
			AvatarColor: "#059669", AvatarSymbol: "♝", StyleTag: "Blockade & Prophylaxis",
			Bio:     "Inspired by hypermodern strategy: controls the center from afar, blockades passed pawns, and overprotects key squares.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 54, TradeTendency: 48, TacticalSharpness: 86, PositionalSolidity: 95, BlunderRate: 8},
			OpeningBook: map[string][]string{
				startBoardFEN: {"g1f3", "b2b3", "c2c4"},
			},
			RecommendedLessons: []string{"t5_prophylaxis", "t5_minority_attack"},
		},
		{
			ID: "t5_alekhine_style", Name: "A. Alekhine", Title: "Dynamic Master Style", Tier: 5, Elo: 2110,
			AvatarColor: "#b91c1c", AvatarSymbol: "⚔", StyleTag: "Deep Tactical Storms",
			Bio:     "Inspired by relentless initiative: weaves complex multi-piece attacks from quiet-looking positions.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 94, TradeTendency: 30, TacticalSharpness: 94, PositionalSolidity: 86, BlunderRate: 7},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4"},
			},
			RecommendedLessons: []string{"t5_exchange_sac", "t5_prophylaxis"},
		},
		{
			ID: "t5_sofia", Name: "Sofia", Title: "Carlsbad Strategist", Tier: 5, Elo: 2140,
			AvatarColor: "#2563eb", AvatarSymbol: "♙", StyleTag: "Queenside Minority Attack",
			Bio:     "Specializes in Queen's Gambit structures, minority attacks on the queenside, and surgical exchange sacrifices.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 58, TradeTendency: 62, TacticalSharpness: 88, PositionalSolidity: 92, BlunderRate: 6},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4"},
			},
			RecommendedLessons: []string{"t5_minority_attack", "t5_exchange_sac"},
		},
		{
			ID: "t5_viktor_boss", Name: "CM Viktor", Title: "Tier 5 Boss", Tier: 5, IsBoss: true, Elo: 2190,
			AvatarColor: "#4f46e5", AvatarSymbol: "♔", StyleTag: "Universal Candidate Master",
			Bio:     "Switches effortlessly between quiet prophylactic maneuvering and concrete calculation.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 68, TradeTendency: 56, TacticalSharpness: 92, PositionalSolidity: 92, BlunderRate: 5},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4", "c2c4"},
			},
			RecommendedLessons: []string{"t5_prophylaxis", "t5_minority_attack", "t5_exchange_sac"},
		},

		// ==================== TIER 6: GRANDMASTER HALL (2200-2600+) ====================
		{
			ID: "t6_lasker_style", Name: "Emanuel Lasker", Title: "Practical Fighter Style", Tier: 6, Elo: 2320,
			AvatarColor: "#9333ea", AvatarSymbol: "♛", StyleTag: "Tenacious Counterplay",
			Bio:     "Inspired by pragmatic fighting chess: poses maximum practical problems even in difficult positions.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 76, TradeTendency: 46, TacticalSharpness: 95, PositionalSolidity: 92, BlunderRate: 4},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4"},
			},
			RecommendedLessons: []string{"t6_fortress_defense", "t6_deep_calculation"},
		},
		{
			ID: "t6_rubinstein_style", Name: "Akiba Rubinstein", Title: "Endgame Virtuoso Style", Tier: 6, Elo: 2400,
			AvatarColor: "#0d9488", AvatarSymbol: "♖", StyleTag: "Seamless Opening-to-Endgame",
			Bio:     "Inspired by harmonic rook play and textbook endgame artistry.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 52, TradeTendency: 78, TacticalSharpness: 95, PositionalSolidity: 98, BlunderRate: 3},
			OpeningBook: map[string][]string{
				startBoardFEN: {"d2d4", "c2c4"},
			},
			RecommendedLessons: []string{"t6_technical_conversion", "t6_fortress_defense"},
		},
		{
			ID: "t6_apex_tactician", Name: "Apex Tactician", Title: "Grandmaster Specialist", Tier: 6, Elo: 2490,
			AvatarColor: "#e11d48", AvatarSymbol: "⚔", StyleTag: "Zero-Mercy Calculation",
			Bio:     "Sharp, uncompromising master profile that punishes even a single tempo loss.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 90, TradeTendency: 35, TacticalSharpness: 98, PositionalSolidity: 92, BlunderRate: 2},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4"},
			},
			RecommendedLessons: []string{"t6_deep_calculation", "t6_fortress_defense"},
		},
		{
			ID: "t6_grandmaster_boss", Name: "The Grandmaster", Title: "Academy Final Boss", Tier: 6, IsBoss: true, Elo: 2600,
			AvatarColor: "#f59e0b", AvatarSymbol: "♔", StyleTag: "Universal Championship Strength",
			Bio:     "The pinnacle of the NewScid Academy Career Ladder. Full-strength tactical precision and endgame technique.",
			Source:  "builtin",
			Sliders: PersonalitySliders{Aggression: 70, TradeTendency: 55, TacticalSharpness: 100, PositionalSolidity: 100, BlunderRate: 0},
			OpeningBook: map[string][]string{
				startBoardFEN: {"e2e4", "d2d4", "c2c4", "g1f3"},
			},
			RecommendedLessons: []string{"t6_deep_calculation", "t6_technical_conversion", "t6_fortress_defense"},
		},
	}
}

// FindBuiltinPersonality looks up a built-in personality by ID.
func FindBuiltinPersonality(id string) (Personality, bool) {
	for _, p := range BuiltinPersonalities() {
		if p.ID == id {
			return p, true
		}
	}
	return Personality{}, false
}
