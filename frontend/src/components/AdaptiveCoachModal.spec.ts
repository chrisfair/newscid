import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AdaptiveCoachModal from './AdaptiveCoachModal.vue';
import * as AppAPI from '../../wailsjs/go/main/App';

vi.mock('../../wailsjs/go/main/App', () => ({
  GetAdaptiveCoachDashboard: vi.fn(),
  GenerateAdaptiveCoachingSession: vi.fn(),
  RecordAdaptiveCoachAttempt: vi.fn(),
  GetMainUserOnlineDefaults: vi.fn(),
  ParseFEN: vi.fn(),
  MakeMoveOnBoard: vi.fn(),
}));

describe('AdaptiveCoachModal', () => {
  const sampleDrill = {
    id: 'online-game-1-ply-11',
    sourceType: 'online_game_mistake',
    white: 'HeroPlayer',
    black: 'Villain1',
    event: 'Lichess Blitz',
    date: '2026.10.30',
    ply: 11,
    moveNumber: 6,
    fen: 'r1bqr1k1/pppp1ppp/5n2/4N3/1b2P3/P1N5/1PPP1PPP/R1BQKB1R w KQ - 1 6',
    turn: 'w',
    solutionUci: ['a3b4'],
    solutionSan: ['axb4'],
    playedBlunderUci: 'f1e2',
    playedBlunderSan: 'Be2',
    primaryTheme: 'hangingPiece',
    themeName: 'Hanging Piece',
    themes: ['hangingPiece'],
    difficultyTier: 'Easy',
    estimatedRating: 1200,
    evalDeltaCp: 330,
    diagnosisHeadline: 'Your Game Mistake (Hanging Piece): Move 6 in HeroPlayer vs Villain1',
    whatWentWrong: 'On move 6 you played Be2, missing that the opponent\'s bishop on b4 was completely undefended and could be captured immediately with axb4.',
    visualCues: ['Hanging Black b on b4 (1 attacker(s), 0 defender(s)).'],
    stepByStepCoach: 'Board Vision Protocol (Hanging Pieces): Before every move, scan all undefended enemy pieces you can capture.',
    postSolveTakeaway: '[Hanging Piece] White captures the undefended bishop on b4.',
    attempted: false,
    solvedFirstTry: false,
    attemptsCount: 0,
    timeTakenSec: 0,
  };

  const sampleDashboard = {
    playerIdentifiers: ['HeroPlayer', 'hero_lichess'],
    gamesAnalyzed: 25,
    totalGameMistakes: 8,
    overallMasteryPct: 48.5,
    totalDrillsSolved: 4,
    coachSummaryText: 'Analyzed 25 game(s) (8 tactical mistakes detected). Top priority weakness: Hanging Piece (6 game misses, 35.0% mastery).',
    topWeaknesses: [
      {
        themeKey: 'hangingPiece',
        themeName: 'Hanging Piece',
        category: 'Motifs',
        gameMissCount: 6,
        drillsAttempted: 4,
        drillsSolved: 2,
        drillsFailed: 2,
        currentStreak: 0,
        bestStreak: 2,
        accuracyPct: 50.0,
        masteryScore: 35.0,
        priorityWeight: 385,
        coachingHabit: 'Board Vision Protocol (Hanging Pieces)',
      },
    ],
    allThemeStats: [
      {
        themeKey: 'hangingPiece',
        themeName: 'Hanging Piece',
        category: 'Motifs',
        gameMissCount: 6,
        drillsAttempted: 4,
        drillsSolved: 2,
        drillsFailed: 2,
        currentStreak: 0,
        bestStreak: 2,
        accuracyPct: 50.0,
        masteryScore: 35.0,
        priorityWeight: 385,
        coachingHabit: 'Board Vision Protocol (Hanging Pieces)',
      },
    ],
    activeQueue: [sampleDrill],
    sessionHistory: [],
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(AppAPI.GetMainUserOnlineDefaults).mockResolvedValue({
      displayName: 'HeroPlayer',
      lichessUsername: 'hero_lichess',
      chessComUsername: 'hero_chesscom',
    });
    vi.mocked(AppAPI.GetAdaptiveCoachDashboard).mockResolvedValue(sampleDashboard as any);
    vi.mocked(AppAPI.ParseFEN).mockResolvedValue({ squares: [] } as any);
  });

  it('displays high-instructional coaching diagnosis, visual cues, and thinking protocol', async () => {
    const wrapper = mount(AdaptiveCoachModal, {
      global: { stubs: { Chessboard: true } },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('Top priority weakness: Hanging Piece');
    expect(wrapper.text()).toContain('Your Game Mistake (Hanging Piece)');
    expect(wrapper.text()).toContain('missing that the opponent\'s bishop on b4 was completely undefended');
    expect(wrapper.text()).toContain('Hanging Black b on b4');
    expect(wrapper.text()).toContain('Board Vision Protocol (Hanging Pieces)');
  });

  it('scans online accounts and builds customized training session on request', async () => {
    vi.mocked(AppAPI.GenerateAdaptiveCoachingSession).mockResolvedValue(sampleDashboard as any);

    const wrapper = mount(AdaptiveCoachModal, {
      global: { stubs: { Chessboard: true } },
    });
    await flushPromises();

    const genBtn = wrapper.find('.generate-btn');
    await genBtn.trigger('click');
    await flushPromises();

    expect(AppAPI.GenerateAdaptiveCoachingSession).toHaveBeenCalledWith({
      source: 'both',
      lichessUsername: 'hero_lichess',
      chessComUsername: 'hero_chesscom',
      playerName: 'HeroPlayer',
      focusTheme: 'auto',
      maxGames: 120,
      batchSize: 12,
    });
  });

  it('renders weakness profile and triggers focused drill-harder session', async () => {
    vi.mocked(AppAPI.GenerateAdaptiveCoachingSession).mockResolvedValue(sampleDashboard as any);

    const wrapper = mount(AdaptiveCoachModal, {
      global: { stubs: { Chessboard: true } },
    });
    await flushPromises();

    const tabs = wrapper.findAll('.mode-tab');
    await tabs[1].trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Diagnosed Tactical Profile & Adaptive Push Weights');
    expect(wrapper.text()).toContain('385');

    const drillHarderBtn = wrapper.find('.weakness-table .btn-primary');
    await drillHarderBtn.trigger('click');
    await flushPromises();

    expect(AppAPI.GenerateAdaptiveCoachingSession).toHaveBeenCalledWith(
      expect.objectContaining({ focusTheme: 'hangingPiece' })
    );
  });
});
