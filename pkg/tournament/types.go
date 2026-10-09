package tournament

// TournamentType describes the format of the chess tournament.
type TournamentType string

const (
	TypeRoundRobin TournamentType = "Round-Robin"
	TypeSwiss      TournamentType = "Swiss"
	TypeMatch      TournamentType = "Match"
)

// EventSummary provides metadata about a distinct tournament event found in a database.
type EventSummary struct {
	Event      string         `json:"event"`
	Site       string         `json:"site"`
	Date       string         `json:"date"`
	Year       int            `json:"year"`
	NumGames   int            `json:"numGames"`
	NumPlayers int            `json:"numPlayers"`
	Rounds     int            `json:"rounds"`
	Type       TournamentType `json:"type"`
}

// SwissRoundResult stores the outcome of a single round for a player in a Swiss or round-by-round tournament.
type SwissRoundResult struct {
	Round        string  `json:"round"`
	OpponentRank int     `json:"opponentRank"`
	OpponentName string  `json:"opponentName"`
	Color        string  `json:"color"`  // "w" or "b"
	Result       string  `json:"result"` // "1", "0", "½", "*"
	Score        float64 `json:"score"`  // 1.0, 0.5, 0.0
	GameID       int     `json:"gameId"`
}

// PlayerStanding represents one competitor's aggregated tournament performance and tiebreak metrics.
type PlayerStanding struct {
	Rank              int                 `json:"rank"`
	PlayerName        string              `json:"playerName"`
	Title             string              `json:"title,omitempty"`
	Elo               int                 `json:"elo"`
	Points            float64             `json:"points"`
	Played            int                 `json:"played"`
	Wins              int                 `json:"wins"`
	Draws             int                 `json:"draws"`
	Losses            int                 `json:"losses"`
	SonnebornBerger   float64             `json:"sonnebornBerger"`
	Buchholz          float64             `json:"buchholz"`
	BuchholzCut1      float64             `json:"buchholzCut1"`
	PerformanceRating int                 `json:"performanceRating"`
	RoundResults      []*SwissRoundResult `json:"roundResults"`
	ScoresAgainst     map[string]float64  `json:"scoresAgainst"` // Opponent Name -> Total Points scored against
}

// CrosstableCell represents the matchup between row player and column player in a Round-Robin matrix.
type CrosstableCell struct {
	Result string  `json:"result"` // "1", "0", "½", "1 0", "X", etc.
	Score  float64 `json:"score"`
	Color  string  `json:"color"`  // "w", "b", or ""
	GameID int     `json:"gameId"` // First or primary game ID for jump-to-game navigation
}

// TournamentCrosstable contains the complete calculated standings, matrix, and details of an event.
type TournamentCrosstable struct {
	Event          string             `json:"event"`
	Site           string             `json:"site"`
	Date           string             `json:"date"`
	Category       int                `json:"category"`
	AverageElo     int                `json:"averageElo"`
	NumPlayers     int                `json:"numPlayers"`
	NumGames       int                `json:"numGames"`
	NumRounds      int                `json:"numRounds"`
	Type           TournamentType     `json:"type"`
	Standings      []*PlayerStanding  `json:"standings"`
	Matrix         [][]CrosstableCell `json:"matrix"`
	CrossTableText string             `json:"crossTableText"`
}
