package scid

import (
	"path/filepath"
	"strings"
	"testing"

	"newscid/pkg/chess"
)

func BenchmarkImportGames(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 48; i++ {
		sb.WriteString("[Event \"Bench\"]\n[White \"Alpha\"]\n[Black \"Beta\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0\n\n")
	}
	games, err := chess.ParsePGNDatabase(strings.NewReader(sb.String()))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db := NewDatabase()
		if err := db.Create(filepath.Join(b.TempDir(), "bench"), "Bench"); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		stats, err := db.ImportGames(games)
		if err != nil || stats.Imported != 48 {
			b.Fatalf("import failed: %+v err=%v", stats, err)
		}
		b.StopTimer()
		_ = db.Close()
	}
}
