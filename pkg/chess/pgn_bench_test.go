package chess

import (
	"strings"
	"testing"
)

func BenchmarkParsePGNDatabase(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 64; i++ {
		sb.WriteString("[Event \"Bench\"]\n[White \"Alpha\"]\n[Black \"Beta\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 1-0\n\n")
	}
	payload := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		games, err := ParsePGNDatabase(strings.NewReader(payload))
		if err != nil || len(games) != 64 {
			b.Fatalf("unexpected result: len=%d err=%v", len(games), err)
		}
	}
}
