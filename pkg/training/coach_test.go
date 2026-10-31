package training_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/training"
)

func TestAdaptiveCoachManager_DiagnosesHangingPiecesAndDrillsHarder(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "adaptive_coach.json")
	mgr, err := training.NewAdaptiveCoachManager(storePath)
	require.NoError(t, err)

	// Game 1: HeroPlayer misses capturing an undefended Black Bishop on b4 (plays 6.Be2 instead of 6.axb4)
	g1, err := chess.ParsePGN(`[Event "Lichess Blitz"]
[White "HeroPlayer"]
[Black "Villain1"]
[Result "0-1"]

1. e4 e5 2. Nf3 Nf6 3. Nc3 Bb4 4. Nxe5 O-O 5. a3 Re8 6. Be2 Rxe5 0-1`)
	require.NoError(t, err)

	// Game 2: HeroPlayer blunders by hanging their Knight on g5 / missing Nxf7 fork
	g2, err := chess.ParsePGN(`[Event "Lichess Rapid"]
[White "HeroPlayer"]
[Black "Villain2"]
[Result "0-1"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 Bc5 5. a3 h6 6. Nh3 d6 0-1`)
	require.NoError(t, err)

	dash, err := mgr.ScanAndBuildCurriculum(
		[]*chess.Game{g1, g2},
		[]string{"HeroPlayer"},
		"auto",
		8,
	)
	require.NoError(t, err)
	require.NotNil(t, dash)

	assert.Equal(t, 2, dash.GamesAnalyzed)
	assert.GreaterOrEqual(t, dash.TotalGameMistakes, 1)
	require.NotEmpty(t, dash.TopWeaknesses)
	require.NotEmpty(t, dash.ActiveQueue)

	// Every drill must have high-instructional coaching fields populated
	firstDrill := dash.ActiveQueue[0]
	assert.NotEmpty(t, firstDrill.DiagnosisHeadline)
	assert.NotEmpty(t, firstDrill.WhatWentWrong)
	assert.NotEmpty(t, firstDrill.VisualCues)
	assert.NotEmpty(t, firstDrill.StepByStepCoach)
	assert.NotEmpty(t, firstDrill.PostSolveTakeaway)

	initialQueueLen := len(dash.ActiveQueue)

	// Simulate failing the first drill -> should push an immediate reinforcement drill of that motif and increase priority weight
	failRes, err := mgr.RecordDrillAttempt(firstDrill.ID, false, 15)
	require.NoError(t, err)
	require.NotNil(t, failRes)
	assert.True(t, failRes.ReinforcementAdded)
	assert.NotNil(t, failRes.AddedDrill)
	assert.Equal(t, firstDrill.PrimaryTheme, failRes.AddedDrill.PrimaryTheme)
	assert.Equal(t, initialQueueLen+1, len(failRes.Dashboard.ActiveQueue))
	assert.Equal(t, 1, failRes.UpdatedTheme.DrillsFailed)

	// Simulate solving the next drill on first try
	secondDrill := failRes.Dashboard.ActiveQueue[1]
	winRes, err := mgr.RecordDrillAttempt(secondDrill.ID, true, 8)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, winRes.UpdatedTheme.DrillsSolved, 1)

	// Verify persistence from disk
	mgrReloaded, err := training.NewAdaptiveCoachManager(storePath)
	require.NoError(t, err)
	reloadedDash := mgrReloaded.GetDashboard()
	assert.Equal(t, 2, reloadedDash.GamesAnalyzed)
	assert.Len(t, reloadedDash.ActiveQueue, initialQueueLen+1)
}
