package fairplay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestRadarParity_BandsMultipliersAndOffsets(t *testing.T) {
	ccBlitz := GetFormatAccuracyBand("blitz", 1400, "chesscom")
	assert.Equal(t, 80.0, ccBlitz.Expected)
	assert.Equal(t, 88.0, ccBlitz.Threshold)

	liRapid := GetFormatAccuracyBand("rapid", 2100, "lichess")
	assert.Equal(t, 84.0, liRapid.Expected)
	assert.Equal(t, 90.0, liRapid.Threshold)

	// Classical falls back to rapid band
	liClassical := GetFormatAccuracyBand("classical", 2900, "lichess")
	assert.Equal(t, 96.0, liClassical.Threshold)

	// Unknown format falls back to blitz
	fallback := GetFormatAccuracyBand("correspondence", 3500, "chesscom")
	assert.Equal(t, 94.0, fallback.Threshold)

	// CRS account age multipliers
	assert.Equal(t, 1.0, GetCRSAccountAgeFactor(0))
	assert.Equal(t, 5.0, GetCRSAccountAgeFactor(5))
	assert.Equal(t, 4.0, GetCRSAccountAgeFactor(10))
	assert.Equal(t, 3.5, GetCRSAccountAgeFactor(25))
	assert.Equal(t, 2.5, GetCRSAccountAgeFactor(45))
	assert.Equal(t, 2.0, GetCRSAccountAgeFactor(80))
	assert.Equal(t, 1.5, GetCRSAccountAgeFactor(150))
	assert.Equal(t, 1.2, GetCRSAccountAgeFactor(300))
	assert.Equal(t, 1.0, GetCRSAccountAgeFactor(500))

	// Elo-pool win-rate offset
	assert.Equal(t, 0.25, GetEloPoolWinRateOffset(2200, 1000, 500, "GM"))
	assert.Equal(t, 0.28, GetEloPoolWinRateOffset(2600, 1000, 500, "GM"))
	assert.Equal(t, 0.21, GetEloPoolWinRateOffset(2100, 1000, 500, "IM"))
	assert.Equal(t, 0.18, GetEloPoolWinRateOffset(2000, 1000, 500, "FM"))
	assert.Equal(t, 0.0, GetEloPoolWinRateOffset(1500, 500, 200, ""))
	assert.Greater(t, GetEloPoolWinRateOffset(2200, 500, 300, ""), 0.10)

	// ACPL -> Accuracy %
	assert.Equal(t, 99.5, ACPLToAccuracyPct(0))
	assert.Greater(t, ACPLToAccuracyPct(12), 90.0)
	assert.Equal(t, 10.0, ACPLToAccuracyPct(900))
}

func TestRadarParity_BuildRadarAssessment_HighRiskAndOfficialBan(t *testing.T) {
	assert.Nil(t, BuildRadarAssessment(nil, RadarProfileHints{}))

	dossier := &PlayerFairPlayDossier{
		PlayerName:             "SuspectAccount",
		TotalGamesAnalyzed:     20,
		OverallIssueScore:      78.5,
		HighestRiskTimeControl: "Rapid",
		TimeControlProfiles: []TimeControlIssueProfile{
			{
				Category:       "Rapid",
				GamesAnalyzed:  20,
				WinRatePct:     90.0,
				AvgOfficialElo: 1450,
			},
		},
		CrossDivergence: CrossControlDivergence{
			HasMultipleControls: true,
			DivergenceScore:     75.0,
			Explanation:         "Rapid +380 IPR above Bullet",
		},
		AllGames: []GameFairPlayRecord{
			{Site: "https://lichess.org/abc", TimeControlCategory: "Rapid", PlayerElo: 1450, NonBookMoves: 25, HasEvalData: true, ACPL: 9.0, HasClockData: true, MoveTimeCV: 0.25, MicroBurstWindowCV: 0.18, ForcedMoveHesitationCount: 3},
			{Site: "https://lichess.org/def", TimeControlCategory: "Rapid", PlayerElo: 1450, NonBookMoves: 28, HasEvalData: true, ACPL: 8.5, HasClockData: true, MoveTimeCV: 0.29, MicroBurstWindowCV: 0.20, ForcedMoveHesitationCount: 2},
			{Site: "https://lichess.org/ghi", TimeControlCategory: "Rapid", PlayerElo: 1450, NonBookMoves: 30, HasEvalData: true, ACPL: 10.0, HasClockData: true, MoveTimeCV: 0.31, MicroBurstWindowCV: 0.21, ForcedMoveHesitationCount: 2},
		},
	}

	res := BuildRadarAssessment(dossier, RadarProfileHints{
		AccountAgeDays:  12,
		OverallGames:    35,
		PuzzleRating:    1100,
		PuzzleCount:     5,
		SevenDayEloGain: 220,
	})
	require.NotNil(t, res)
	assert.Equal(t, "lichess", res.Platform)
	assert.Len(t, res.Signals, 6)
	assert.Equal(t, 3, res.HighAccuracyGameCount)
	assert.Equal(t, 3, res.MaxHighAccuracyStreak)
	assert.GreaterOrEqual(t, res.CheatLikelihoodPct, 65)
	assert.Contains(t, []string{"high", "critical"}, res.RiskTier)

	// Official ban forces critical 99%
	banned := BuildRadarAssessment(dossier, RadarProfileHints{
		Platform:        "chesscom",
		OfficialBanned:  true,
		OfficialBanNote: "Closed: Fair Play",
	})
	require.NotNil(t, banned)
	assert.Equal(t, "critical", banned.RiskTier)
	assert.Equal(t, 99, banned.CheatLikelihoodPct)
	assert.True(t, banned.Signals[0].Flagged)
}

func TestCommunityTelemetryClient_LiveAndOfflineFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/intel":
			_ = json.NewEncoder(w).Encode(CommunitySuspectIntel{
				Platform:            r.URL.Query().Get("platform"),
				Username:            r.URL.Query().Get("username"),
				CommunityFlagsCount: 4,
				ScansLast24h:        9,
				AvgReportedRiskPct:  78,
				CrowdVerdict:        "COMMUNITY_HOTLIST",
			})
		case "/v1/flag":
			_ = json.NewEncoder(w).Encode(CommunitySuspectIntel{
				Platform:            "lichess",
				Username:            "flagged_user",
				CommunityFlagsCount: 5,
				ScansLast24h:        10,
				AvgReportedRiskPct:  82,
				CrowdVerdict:        "COMMUNITY_HOTLIST",
			})
		case "/v1/hotlist":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"hotlist": []CommunitySuspectIntel{
					{Platform: "lichess", Username: "flagged_user", CrowdVerdict: "COMMUNITY_HOTLIST"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewCommunityTelemetryClient(srv.URL)
	intel := client.FetchIntel(context.Background(), "lichess", "flagged_user", 70, false)
	assert.Equal(t, "COMMUNITY_HOTLIST", intel.CrowdVerdict)
	assert.Equal(t, 4, intel.CommunityFlagsCount)
	assert.False(t, intel.IsOfflineEstimate)

	flagged, err := client.SubmitFlag(context.Background(), "lichess", "flagged_user", 82, false, "")
	require.NoError(t, err)
	assert.Equal(t, 5, flagged.CommunityFlagsCount)

	hotlist, err := client.FetchHotlist(context.Background())
	require.NoError(t, err)
	assert.Len(t, hotlist, 1)

	_, err = client.SubmitFlag(context.Background(), "lichess", "", 50, false, "")
	assert.Error(t, err)

	// Offline fallback when server is closed
	srv.Close()
	offline := client.FetchIntel(context.Background(), "chesscom", "offline_user", 78, false)
	assert.True(t, offline.IsOfflineEstimate)
	assert.Equal(t, "COMMUNITY_HOTLIST", offline.CrowdVerdict)

	offlineBanned := client.FetchIntel(context.Background(), "", "banned_user", 40, true)
	assert.Equal(t, "CONFIRMED_BANNED", offlineBanned.CrowdVerdict)
}

func TestCommunityTelemetryClient_EnvAndErrors(t *testing.T) {
	t.Setenv("NEWSCID_TELEMETRY_API", "https://example.invalid/telemetry/")
	cEnv := NewCommunityTelemetryClient("")
	assert.Equal(t, "https://example.invalid/telemetry", cEnv.BaseURL)

	t.Setenv("NEWSCID_TELEMETRY_API", "")
	cDef := NewCommunityTelemetryClient("")
	assert.Equal(t, DefaultCommunityAPIBase, cDef.BaseURL)

	var mode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "500":
			w.WriteHeader(http.StatusInternalServerError)
		case "bad_json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not-json`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true,"hotlist":[]}`))
		}
	}))
	defer srv.Close()

	client := NewCommunityTelemetryClient(srv.URL)

	// Offline WATCHED and CLEAN branches with nil context
	mode = "500"
	watched := client.FetchIntel(nil, "lichess", "user_w", 55, false)
	assert.True(t, watched.IsOfflineEstimate)
	assert.Equal(t, "WATCHED", watched.CrowdVerdict)
	assert.Equal(t, 1, watched.CommunityFlagsCount)

	clean := client.FetchIntel(nil, "lichess", "user_c", 20, false)
	assert.Equal(t, "CLEAN", clean.CrowdVerdict)

	// SubmitFlag & FetchHotlist HTTP 500 + bad JSON + nil ctx + empty platform
	_, err := client.SubmitFlag(nil, "", "u1", 80, false, "reason")
	assert.Error(t, err)
	_, err = client.FetchHotlist(nil)
	assert.Error(t, err)

	mode = "bad_json"
	_, err = client.SubmitFlag(context.Background(), "lichess", "u1", 80, false, "")
	assert.Error(t, err)
	_, err = client.FetchHotlist(context.Background())
	assert.Error(t, err)

	// Closed server network error
	srv.Close()
	_, err = client.SubmitFlag(context.Background(), "lichess", "u1", 80, false, "")
	assert.Error(t, err)
	_, err = client.FetchHotlist(context.Background())
	assert.Error(t, err)
}

func TestHardMoveBenchmark_AnalyzeAndHelpers(t *testing.T) {
	// Error cases for AnalyzeHardMoveBenchmark
	_, err := AnalyzeHardMoveBenchmark(nil, "Hero", "ALL", 10, false)
	assert.Error(t, err)

	tmpDir := t.TempDir()
	db := scid.NewDatabase()
	require.NoError(t, db.Create(filepath.Join(tmpDir, "fp_bench"), "FairPlay Bench"))
	defer db.Close()

	_, err = AnalyzeHardMoveBenchmark(db, "   ", "ALL", 10, false)
	assert.Error(t, err)
	_, err = AnalyzeHardMoveBenchmark(db, "Nobody", "ALL", 10, false)
	assert.Error(t, err)

	pgnBlitz := `[Event "Rated Blitz game"]
[Site "https://lichess.org/abcdefgh"]
[Date "2026.05.01"]
[White "BenchHero"]
[Black "OppOne"]
[Result "1-0"]
[WhiteElo "1800"]
[BlackElo "1800"]
[TimeControl "180+0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Na5 10. Bc2 c5 11. d4 Qc7 1-0`

	pgnRapid := `[Event "Rated Rapid game"]
[Site "https://lichess.org/ijklmnop"]
[Date "2026.05.02"]
[White "OppTwo"]
[Black "BenchHero"]
[Result "0-1"]
[WhiteElo "1800"]
[BlackElo "1800"]
[TimeControl "600+0"]

1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. Bg5 Be7 5. e3 O-O 6. Nf3 h6 7. Bh4 b6 8. cxd5 Nxd5 9. Bxe7 Qxe7 10. Nxd5 exd5 0-1`

	g1, err := chess.ParsePGN(pgnBlitz)
	require.NoError(t, err)
	g2, err := chess.ParsePGN(pgnRapid)
	require.NoError(t, err)
	_, err = db.SaveGame(g1)
	require.NoError(t, err)
	_, err = db.SaveGame(g2)
	require.NoError(t, err)

	repAll, err := AnalyzeHardMoveBenchmark(db, "BenchHero", "", 0, false)
	require.NoError(t, err)
	assert.Equal(t, 2, repAll.TotalGamesScanned)

	repBlitz, err := AnalyzeHardMoveBenchmark(db, "BenchHero", "Blitz", 10, false)
	require.NoError(t, err)
	assert.Equal(t, 1, repBlitz.TotalGamesScanned)

	// Math & verdict helpers
	assert.Equal(t, 1.0, poissonBinomialTail([]float64{0.5}, 0))
	assert.Equal(t, 0.0, poissonBinomialTail([]float64{0.5}, 2))
	pClamp := poissonBinomialTail([]float64{-0.1, 1.5}, 1)
	assert.InDelta(t, 1.0, pClamp, 1e-9)

	assert.Equal(t, 1.0, poissonTailProb(0, 2.0))
	assert.Equal(t, 0.5, poissonTailProb(2, 3.0))
	assert.Less(t, poissonTailProb(10, 2.0), 0.01)

	assert.Equal(t, 0.0, empiricalQuantile(nil, 0.5))
	assert.Equal(t, 1.0, empiricalQuantile([]float64{1, 2, 3}, 0))
	assert.Equal(t, 3.0, empiricalQuantile([]float64{1, 2, 3}, 1))
	assert.Equal(t, 2.0, empiricalQuantile([]float64{1, 2, 3}, 0.5))
	assert.InDelta(t, 1.5, empiricalQuantile([]float64{1, 2, 3}, 0.25), 1e-9)

	rZero := &HardMoveBenchmarkReport{}
	buildHardMoveVerdict(rZero)
	assert.Contains(t, rZero.VerdictHeadline, "No qualified")

	rExcess := &HardMoveBenchmarkReport{
		OpponentGamesQualified: 50,
		ExcessOpponentGames:    5.0,
		OddsOneIn:              100,
		BannedValidation: BannedValidationSummary{
			GamesVsBanned:        3,
			ConfirmedBannedCount: 2,
			BannedTopTwoRatePct:  66.7,
			MultiplierVsOpen:     8.0,
		},
	}
	buildHardMoveVerdict(rExcess)
	assert.Contains(t, rExcess.VerdictHeadline, "more unusual")
	assert.Len(t, rExcess.VerdictFindings, 4)

	// Inspector Elo & title calibration helpers
	for _, elo := range []int{2700, 2500, 2300, 2100, 1900, 1500} {
		band := getOTBEmpiricalBand(elo)
		assert.Greater(t, band.p99WinPct, 0.0)
		sev, elev := expectedSuspectACPLThreshold(elo, "Bullet")
		assert.Less(t, sev, elev)
	}
	for _, title := range []string{"GM", "IM", "LM", "WGM", "FM", "WIM", "NM", "CM", "WFM", "WNM", "WCM", ""} {
		_ = verifiedMasterRatingFloor(title)
		d := verifiedMasterTitleDampener(title)
		assert.Greater(t, d, 0.0)
	}
}
