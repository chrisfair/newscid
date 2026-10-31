import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import OpeningTrainerModal from './OpeningTrainerModal.vue';
import * as AppAPI from '../../wailsjs/go/main/App';

vi.mock('../../wailsjs/go/main/App', () => ({
  GetRepertoires: vi.fn(),
  GetRepertoire: vi.fn(),
  SaveRepertoire: vi.fn(),
  DeleteRepertoire: vi.fn(),
  VerifyRepertoireMove: vi.fn(),
  RecordRepertoirePractice: vi.fn(),
  AddRepertoireMove: vi.fn(),
  AddRepertoireLineSAN: vi.fn(),
  DeleteRepertoireMove: vi.fn(),
  UpdateRepertoireMoveComment: vi.fn(),
  GetRepertoireMastery: vi.fn(),
  ResetRepertoireMastery: vi.fn(),
  CompileRepertoireToPolyglot: vi.fn(),
  GetMainUserOnlineDefaults: vi.fn(),
  ScanRepertoireDeviations: vi.fn(),
  AddBlunderToVault: vi.fn(),
  ParseFEN: vi.fn(),
  MakeMoveOnBoard: vi.fn(),
}));

describe('OpeningTrainerModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(AppAPI.GetMainUserOnlineDefaults).mockResolvedValue({
      displayName: 'HeroPlayer',
      lichessUsername: 'hero_lichess',
      chessComUsername: 'hero_chesscom',
      whiteRepertoireId: 'white-openings-italian',
      blackRepertoireId: 'black-openings-sicilian-kid',
    });

    vi.mocked(AppAPI.GetRepertoires).mockResolvedValue([
      {
        id: 'white-openings-italian',
        name: 'White: Italian Game & Open Repertoire',
        color: 'w',
        description: 'Classical 1.e4 lines',
        numMoves: 18,
      },
    ] as any);

    vi.mocked(AppAPI.GetRepertoire).mockResolvedValue({
      id: 'white-openings-italian',
      name: 'White: Italian Game & Open Repertoire',
      color: 'w',
      description: 'Classical 1.e4 lines',
      positions: {
        'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq': [
          {
            uci: 'e2e4',
            san: 'e4',
            nextFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq',
            comment: 'Italian Game: Giuoco Piano',
            practice: 2,
            correct: 2,
            streak: 2,
            mastered: true,
          },
        ],
      },
    } as any);

    vi.mocked(AppAPI.GetRepertoireMastery).mockResolvedValue({
      repertoireId: 'white-openings-italian',
      repertoireName: 'White: Italian Game & Open Repertoire',
      color: 'w',
      totalUserNodes: 5,
      masteredUserNodes: 3,
      untestedUserNodes: 2,
      needsWorkUserNodes: 0,
      masteryPct: 60.0,
      totalLines: 2,
      masteredLines: 1,
      nextTargetLineIndex: 1,
      lines: [
        {
          index: 0,
          name: 'Italian Game: Giuoco Piano',
          sanLine: '1.e4 e5 2.Nf3 Nc6 3.Bc4 Bc5',
          steps: [],
          userMovesCount: 3,
          masteredCount: 3,
          masteryPct: 100.0,
          isMastered: true,
        },
        {
          index: 1,
          name: 'Two Knights: Fried Liver Attack',
          sanLine: '1.e4 e5 2.Nf3 Nc6 3.Bc4 Nf6 4.Ng5',
          steps: [],
          userMovesCount: 2,
          masteredCount: 0,
          masteryPct: 0.0,
          isMastered: false,
        },
      ],
    });

    vi.mocked(AppAPI.ParseFEN).mockResolvedValue({ squares: [] } as any);
  });

  it('displays 0-100% Book Mastery meter and systematic variation checklist', async () => {
    const wrapper = mount(OpeningTrainerModal, {
      global: {
        stubs: { Chessboard: true },
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('Book Mastery:');
    expect(wrapper.text()).toContain('60%');
    expect(wrapper.text()).toContain('Italian Game: Giuoco Piano');
    expect(wrapper.text()).toContain('Two Knights: Fried Liver Attack');
  });

  it('imports a SAN/PGN line in Define & Update Book mode', async () => {
    vi.mocked(AppAPI.AddRepertoireLineSAN).mockResolvedValue({
      id: 'white-openings-italian',
      name: 'White: Italian Game & Open Repertoire',
      color: 'w',
      positions: {},
    } as any);

    const wrapper = mount(OpeningTrainerModal, {
      global: {
        stubs: { Chessboard: true },
      },
    });
    await flushPromises();

    const tabs = wrapper.findAll('.mode-tab');
    await tabs[1].trigger('click');
    await flushPromises();

    const textarea = wrapper.find('.line-textarea');
    await textarea.setValue('1. e4 e5 2. Nf3 Nc6 3. Bb5 a6');
    const importBtn = wrapper.find('.import-row .btn-primary');
    await importBtn.trigger('click');
    await flushPromises();

    expect(AppAPI.AddRepertoireLineSAN).toHaveBeenCalledWith(
      'white-openings-italian',
      '1. e4 e5 2. Nf3 Nc6 3. Bb5 a6',
      ''
    );
  });

  it('scans online accounts for book deviations and displays how to avoid bad moves', async () => {
    vi.mocked(AppAPI.ScanRepertoireDeviations).mockResolvedValue({
      repertoireId: 'white-openings-italian',
      repertoireName: 'White: Italian Game & Open Repertoire',
      color: 'w',
      totalGamesScanned: 40,
      matchingColorGames: 20,
      gamesEnteredBook: 18,
      compliantGames: 14,
      userDeviatedGames: 4,
      opponentSidelineGames: 2,
      compliancePct: 77.8,
      userDeviations: [
        {
          key: 'pos1|f3e5',
          fen: 'r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3',
          coreFen: 'r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq',
          ply: 5,
          moveNumber: 3,
          lineBeforeSan: '1.e4 e5 2.Nf3 Nc6',
          playedMoveUci: 'f3e5',
          playedMoveSan: 'Nxe5',
          primaryBookUci: 'f1c4',
          primaryBookSan: 'Bc4',
          occurrences: 4,
          wins: 0,
          draws: 1,
          losses: 3,
          winRatePct: 12.5,
          evalLossCp: 265,
          punishingReplySan: 'Nxe5',
          avoidanceAdvice: 'On move 3 (after 1.e4 e5 2.Nf3 Nc6), you played Nxe5 instead of your book move Bc4. Avoid Nxe5: it concedes 265 cp and allows the opponent to respond with Nxe5.',
          expectedMoves: [{ uci: 'f1c4', san: 'Bc4' }],
          sampleGames: [],
        },
      ],
      opponentSidelines: [
        {
          key: 'pos2|d7d5',
          fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1',
          coreFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq',
          afterOpponentFen: 'rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2',
          ply: 2,
          moveNumber: 1,
          lineBeforeSan: '1.e4',
          opponentMoveUci: 'd7d5',
          opponentMoveSan: 'd5',
          fullLineWithReply: '1.e4 d5 2.exd5',
          occurrences: 2,
          wins: 1,
          draws: 0,
          losses: 1,
          winRatePct: 50.0,
          recommendedReplyUci: 'e4d5',
          recommendedReplySan: 'exd5',
          remediationAdvice: 'Opponents played 1...d5 (d7d5) leaving your prepared book. Add exd5 to your book to cover this sideline.',
        },
      ],
    });

    const wrapper = mount(OpeningTrainerModal, {
      global: {
        stubs: { Chessboard: true },
      },
    });
    await flushPromises();

    // Switch to Online Book Breaks tab
    const tabs = wrapper.findAll('.mode-tab');
    await tabs[2].trigger('click');
    await flushPromises();

    const scanBtn = wrapper.find('.scan-btn');
    await scanBtn.trigger('click');
    await flushPromises();

    expect(AppAPI.ScanRepertoireDeviations).toHaveBeenCalledWith({
      repertoireId: 'white-openings-italian',
      source: 'both',
      lichessUsername: 'hero_lichess',
      chessComUsername: 'hero_chesscom',
      playerName: 'HeroPlayer',
      maxGames: 150,
    });

    expect(wrapper.text()).toContain('77.8%');
    expect(wrapper.text()).toContain('You Played: Nxe5');
    expect(wrapper.text()).toContain('Book Move: Bc4');
    expect(wrapper.text()).toContain('How to Avoid:');
    expect(wrapper.text()).toContain('allows the opponent to respond with Nxe5');
  });
});
