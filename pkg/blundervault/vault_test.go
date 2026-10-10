package blundervault

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVault_AddAndReviewSM2(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "blundervault_test*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "vault.json")
	vault, err := NewVault(dbPath)
	if err != nil {
		t.Fatalf("NewVault failed: %v", err)
	}

	p := &Puzzle{
		ID:         "test-p1",
		FEN:        "r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3",
		PlayerName: "Hero",
		UserColor:  "white",
		PlayedMove: "f3e5",
		BestMove:   "f1b5",
		DeltaCp:    -240,
		Event:      "Club Championship",
		Date:       "2026.05.01",
	}

	if err := vault.AddPuzzle(p); err != nil {
		t.Fatalf("AddPuzzle failed: %v", err)
	}

	due := vault.GetDuePuzzles()
	if len(due) != 1 {
		t.Fatalf("expected 1 due puzzle, got %d", len(due))
	}

	// 1. Pass review with quality 4 (good recall)
	updated, err := vault.RecordReview("test-p1", 4)
	if err != nil {
		t.Fatalf("RecordReview failed: %v", err)
	}

	if updated.Repetitions != 1 {
		t.Errorf("expected 1 repetition, got %d", updated.Repetitions)
	}
	if updated.IntervalDays != 1 {
		t.Errorf("expected 1 interval day on first success, got %d", updated.IntervalDays)
	}

	// 2. Pass review with quality 5 (perfect recall)
	updated2, err := vault.RecordReview("test-p1", 5)
	if err != nil {
		t.Fatalf("RecordReview failed: %v", err)
	}
	if updated2.Repetitions != 2 {
		t.Errorf("expected 2 repetitions, got %d", updated2.Repetitions)
	}
	if updated2.IntervalDays != 6 {
		t.Errorf("expected 6 interval days on second success, got %d", updated2.IntervalDays)
	}

	// Verify persistence: reopen vault
	vault2, err := NewVault(dbPath)
	if err != nil {
		t.Fatalf("reopen vault failed: %v", err)
	}
	all := vault2.GetAllPuzzles()
	if len(all) != 1 {
		t.Errorf("expected 1 puzzle in reopened vault, got %d", len(all))
	}
}

func TestVault_FailReset(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "blundervault_test*")
	defer os.RemoveAll(tmpDir)

	vault, _ := NewVault(filepath.Join(tmpDir, "vault.json"))
	p := &Puzzle{ID: "p-fail", NextReview: time.Now()}
	vault.AddPuzzle(p)

	// First pass
	vault.RecordReview("p-fail", 4)
	// Fail (quality 1)
	updated, err := vault.RecordReview("p-fail", 1)
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if updated.Repetitions != 0 {
		t.Errorf("expected repetitions reset to 0, got %d", updated.Repetitions)
	}
	if updated.IntervalDays != 1 {
		t.Errorf("expected interval reset to 1, got %d", updated.IntervalDays)
	}
}
