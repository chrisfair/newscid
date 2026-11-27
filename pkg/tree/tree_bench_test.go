package tree_test

import (
	"testing"

	"newscid/pkg/chess"
	"newscid/pkg/tree"
)

func BenchmarkBuildTreeAndSearch(b *testing.B) {
	pgns := []string{
		`[White "A"][Black "B"][Result "1-0"] 1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0`,
		`[White "C"][Black "D"][Result "0-1"] 1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. Bg5 Be7 5. e3 O-O 0-1`,
		`[White "E"][Black "F"][Result "1/2-1/2"] 1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 1/2-1/2`,
	}
	var games []*chess.Game
	for i := 0; i < 120; i++ {
		g, err := chess.ParsePGN(pgns[i%len(pgns)])
		if err != nil {
			b.Fatal(err)
		}
		games = append(games, g)
	}

	b.Run("BuildTreeFromGames", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			res := tree.BuildTreeFromGames(chess.StartingFEN, games)
			if res.TotalGames != 120 {
				b.Fatalf("unexpected total: %d", res.TotalGames)
			}
		}
	})

	b.Run("SearchPositionInGames", func(b *testing.B) {
		bd := chess.NewBoard()
		m, _ := bd.ParseSAN("d4")
		bd.MakeMove(m)
		target := bd.FEN()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := tree.SearchPositionInGames(target, games)
			if len(idx) != 40 {
				b.Fatalf("unexpected matches: %d", len(idx))
			}
		}
	})
}
