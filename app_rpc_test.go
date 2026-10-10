package main

import (
	"encoding/json"
	"testing"
)

func TestDispatchRPC_ParseFEN(t *testing.T) {
	app := NewApp()

	// Test ParseFEN with standard starting position
	fenParam, err := json.Marshal("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	result, err := app.DispatchRPC("ParseFEN", []json.RawMessage{fenParam})
	if err != nil {
		t.Fatalf("DispatchRPC ParseFEN failed: %v", err)
	}

	state, ok := result.(*BoardState)
	if !ok {
		t.Fatalf("Expected *BoardState, got %T", result)
	}

	if state.Turn != "w" {
		t.Errorf("Expected turn 'w', got '%s'", state.Turn)
	}
	if len(state.LegalMoves) != 20 {
		t.Errorf("Expected 20 legal moves from start position, got %d", len(state.LegalMoves))
	}
}

func TestDispatchRPC_ClassifyOpening(t *testing.T) {
	app := NewApp()

	movesParam, err := json.Marshal([]string{"e4", "e5", "Nf3", "Nc6", "Bb5"})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	result, err := app.DispatchRPC("ClassifyOpening", []json.RawMessage{movesParam})
	if err != nil {
		t.Fatalf("DispatchRPC ClassifyOpening failed: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected non-nil eco entry")
	}
}

func TestDispatchRPC_MethodNotFound(t *testing.T) {
	app := NewApp()

	_, err := app.DispatchRPC("NonExistentMethod123", nil)
	if err == nil {
		t.Fatalf("Expected error for non-existent method, got nil")
	}
}

func TestDispatchRPC_ParamCountMismatch(t *testing.T) {
	app := NewApp()

	// ParseFEN takes 1 parameter, passing 0
	_, err := app.DispatchRPC("ParseFEN", []json.RawMessage{})
	if err == nil {
		t.Fatalf("Expected error for parameter count mismatch, got nil")
	}
}
