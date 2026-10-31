<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import Chessboard from './Chessboard.vue';
import {
  GetRepertoires,
  GetRepertoire,
  SaveRepertoire,
  DeleteRepertoire,
  VerifyRepertoireMove,
  RecordRepertoirePractice,
  AddRepertoireMove,
  AddRepertoireLineSAN,
  DeleteRepertoireMove,
  UpdateRepertoireMoveComment,
  GetRepertoireMastery,
  ResetRepertoireMastery,
  CompileRepertoireToPolyglot,
  GetMainUserOnlineDefaults,
  ScanRepertoireDeviations,
  AddBlunderToVault,
  ParseFEN,
  MakeMove,
} from '../../wailsjs/go/main/App';

const props = withDefaults(
  defineProps<{
    currentPieceSet?: string;
    boardTheme?: {
      light: string;
      dark: string;
      highlight: string;
    };
  }>(),
  {
    currentPieceSet: 'cburnett',
    boardTheme: () => ({
      light: '#f0d9b5',
      dark: '#b58863',
      highlight: 'rgba(255, 255, 0, 0.4)',
    }),
  }
);

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'loadIntoApp', fen: string): void;
}>();

const STARTING_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1';

const repertoires = ref<any[]>([]);
const selectedId = ref<string>('');
const activeRep = ref<any | null>(null);
const masteryReport = ref<any | null>(null);

// Active workbench tab: 'drill' | 'builder' | 'deviations'
const mode = ref<'drill' | 'builder' | 'deviations'>('drill');

// Drill mode sub-type: 'systematic' (100% coverage) | 'random'
const drillStrategy = ref<'systematic' | 'random'>('systematic');
const currentLineIndex = ref<number>(0);
const currentStepIndex = ref<number>(0);
const currentLineMistakes = ref<number>(0);

// Board state
const currentFen = ref<string>(STARTING_FEN);
const currentBoard = ref<any | null>(null);
const legalMoves = ref<string[]>([]);
const inCheck = ref<boolean>(false);
const lastMove = ref<{ from: string; to: string } | null>(null);
const flipped = ref<boolean>(false);
const moveHistory = ref<string[]>([]);
const fenHistory = ref<string[]>([STARTING_FEN]);

// Feedback & status
const statusType = ref<'neutral' | 'correct' | 'wrong' | 'complete'>('neutral');
const statusMessage = ref<string>('Select a repertoire and start systematic training.');
const hintMoves = ref<any[]>([]);
const sessionCorrect = ref<number>(0);
const sessionTotal = ref<number>(0);

// Builder inputs
const moveComment = ref<string>('');
const quickLineSAN = ref<string>('');
const quickLineComment = ref<string>('');
const editingComments = ref<Record<string, string>>({});
const polyglotStatus = ref<string>('');

// Create new repertoire form
const showCreateForm = ref<boolean>(false);
const newRepName = ref<string>('');
const newRepColor = ref<string>('w');
const newRepDesc = ref<string>('');

// Online & DB Book Deviation Scanner state
const scanSource = ref<'both' | 'online' | 'database'>('both');
const scanLichessUser = ref<string>('');
const scanChessComUser = ref<string>('');
const scanPlayerName = ref<string>('');
const scanMaxGames = ref<number>(150);
const isScanning = ref<boolean>(false);
const deviationReport = ref<any | null>(null);
const vaultSavedKeys = ref<Record<string, boolean>>({});
const sidelineAddedKeys = ref<Record<string, boolean>>({});

const accuracyPercent = computed(() => {
  if (sessionTotal.value === 0) return 100;
  return Math.round((sessionCorrect.value / sessionTotal.value) * 100);
});

const masteryPct = computed(() => {
  return masteryReport.value?.masteryPct ?? 0;
});

const activeSystematicLine = computed(() => {
  if (!masteryReport.value?.lines || masteryReport.value.lines.length === 0) return null;
  const idx = Math.max(0, Math.min(currentLineIndex.value, masteryReport.value.lines.length - 1));
  return masteryReport.value.lines[idx] || null;
});

function coreFEN(fen: string): string {
  return fen.trim().split(/\s+/).slice(0, 3).join(' ');
}

const currentCoreMoves = computed(() => {
  if (!activeRep.value?.positions) return [];
  const core = coreFEN(currentFen.value);
  return activeRep.value.positions[core] || [];
});

async function loadRepertoireList() {
  try {
    repertoires.value = (await GetRepertoires()) || [];
    if (repertoires.value.length > 0 && !selectedId.value) {
      selectedId.value = repertoires.value[0].id;
      await selectRepertoire(selectedId.value);
    }
  } catch (e) {
    console.error('Failed to load repertoires:', e);
  }
}

async function loadUserOnlineDefaults() {
  try {
    const defaults = await GetMainUserOnlineDefaults();
    if (defaults) {
      if (defaults.lichessUsername && !scanLichessUser.value) {
        scanLichessUser.value = defaults.lichessUsername;
      }
      if (defaults.chessComUsername && !scanChessComUser.value) {
        scanChessComUser.value = defaults.chessComUsername;
      }
      if (defaults.displayName && !scanPlayerName.value) {
        scanPlayerName.value = defaults.displayName;
      }
    }
  } catch (e) {
    // Ignore if defaults unavailable in test environment
  }
}

async function refreshMastery(repId: string) {
  if (!repId) return;
  try {
    masteryReport.value = await GetRepertoireMastery(repId);
    if (masteryReport.value && typeof masteryReport.value.nextTargetLineIndex === 'number' && masteryReport.value.nextTargetLineIndex >= 0) {
      if (currentLineIndex.value >= (masteryReport.value.lines?.length || 0)) {
        currentLineIndex.value = masteryReport.value.nextTargetLineIndex;
      }
    }
  } catch (e) {
    console.error('Failed to load mastery report:', e);
  }
}

async function selectRepertoire(id: string) {
  try {
    selectedId.value = id;
    activeRep.value = await GetRepertoire(id);
    flipped.value = activeRep.value?.color === 'b';
    await refreshMastery(id);
    if (masteryReport.value?.nextTargetLineIndex >= 0) {
      currentLineIndex.value = masteryReport.value.nextTargetLineIndex;
    } else {
      currentLineIndex.value = 0;
    }
    syncEditingComments();
    await startLineOrResetBoard();
  } catch (e) {
    console.error('Failed to load repertoire:', e);
  }
}

function syncEditingComments() {
  const map: Record<string, string> = {};
  for (const m of currentCoreMoves.value) {
    map[m.uci] = m.comment || '';
  }
  editingComments.value = map;
}

watch(currentFen, () => {
  syncEditingComments();
});

async function applyBoardStateFromFEN(fen: string, moveUci?: string) {
  currentFen.value = fen;
  const state = await ParseFEN(fen);
  currentBoard.value = state;
  legalMoves.value = state?.legalMoves || [];
  inCheck.value = Boolean(state?.inCheck);
  if (moveUci && moveUci.length >= 4) {
    lastMove.value = { from: moveUci.slice(0, 2), to: moveUci.slice(2, 4) };
  } else {
    lastMove.value = null;
  }
}

async function startLineOrResetBoard(targetLineIdx?: number) {
  if (typeof targetLineIdx === 'number') {
    currentLineIndex.value = targetLineIdx;
  }
  currentStepIndex.value = 0;
  currentLineMistakes.value = 0;
  await applyBoardStateFromFEN(STARTING_FEN);
  moveHistory.value = [];
  fenHistory.value = [STARTING_FEN];
  hintMoves.value = [];

  if (!activeRep.value) return;

  if (mode.value === 'drill') {
    statusType.value = 'neutral';
    if (drillStrategy.value === 'systematic' && activeSystematicLine.value) {
      const line = activeSystematicLine.value;
      statusMessage.value = `Line #${line.index + 1}/${masteryReport.value.totalLines}: ${line.name} (${line.masteredCount}/${line.userMovesCount} mastered)`;
      if (activeRep.value.color === 'b') {
        setTimeout(() => playNextSystematicOpponentStep(), 300);
      }
    } else {
      if (activeRep.value.color === 'w') {
        statusMessage.value = 'Your turn (White). Play your book move.';
      } else {
        statusMessage.value = 'Black repertoire active. Playing White opener...';
        setTimeout(() => handleRandomOpponentMove(), 350);
      }
    }
  } else if (mode.value === 'builder') {
    statusType.value = 'neutral';
    statusMessage.value = 'Define & Update Mode: play moves on the board or paste a PGN/SAN line below.';
  }
}

async function playNextSystematicOpponentStep() {
  if (mode.value !== 'drill' || drillStrategy.value !== 'systematic') return;
  const line = activeSystematicLine.value;
  if (!line || !line.steps || currentStepIndex.value >= line.steps.length) {
    await finishSystematicLine();
    return;
  }

  const step = line.steps[currentStepIndex.value];
  if (step.isUserMove) {
    statusType.value = 'neutral';
    statusMessage.value = `Your turn (${activeRep.value.color === 'w' ? 'White' : 'Black'}). Play book move #${Math.floor(step.ply / 2) + 1} in ${line.name}.`;
    return;
  }

  try {
    const res = await MakeMove(currentFen.value, step.uci);
    const san = res.moveSan || step.san || step.uci;
    currentFen.value = res.fen;
    currentBoard.value = res;
    legalMoves.value = res.legalMoves || [];
    inCheck.value = Boolean(res.inCheck);
    lastMove.value = { from: step.uci.slice(0, 2), to: step.uci.slice(2, 4) };
    moveHistory.value.push(san);
    fenHistory.value.push(res.fen);
    currentStepIndex.value++;

    if (currentStepIndex.value >= line.steps.length) {
      await finishSystematicLine();
    } else {
      const nextStep = line.steps[currentStepIndex.value];
      if (!nextStep.isUserMove) {
        setTimeout(() => playNextSystematicOpponentStep(), 300);
      } else {
        statusType.value = 'neutral';
        statusMessage.value = `Opponent played ${san}. Your turn — find your book response.`;
      }
    }
  } catch (e) {
    console.error('Failed to play systematic opponent step:', e);
  }
}

async function finishSystematicLine() {
  if (!activeRep.value) return;
  activeRep.value = await GetRepertoire(activeRep.value.id);
  await refreshMastery(activeRep.value.id);

  const line = activeSystematicLine.value;
  statusType.value = 'complete';
  if (masteryReport.value?.masteryPct >= 100) {
    statusMessage.value = '100% Opening Book Mastery achieved across all lines and positions.';
  } else if (currentLineMistakes.value === 0) {
    statusMessage.value = `Completed ${line?.name || 'line'} cleanly. Overall Book Mastery: ${masteryReport.value?.masteryPct ?? 0}%. Click "Next Unmastered Line" to continue.`;
  } else {
    statusMessage.value = `Line finished with ${currentLineMistakes.value} mistake(s). Repeat this line until you complete it with zero errors.`;
  }
}

async function advanceToNextUnmasteredLine() {
  if (!masteryReport.value?.lines || masteryReport.value.lines.length === 0) return;
  const total = masteryReport.value.lines.length;
  for (let offset = 1; offset <= total; offset++) {
    const candidateIdx = (currentLineIndex.value + offset) % total;
    if (!masteryReport.value.lines[candidateIdx].isMastered) {
      await startLineOrResetBoard(candidateIdx);
      return;
    }
  }
  await startLineOrResetBoard((currentLineIndex.value + 1) % total);
}

async function handleRandomOpponentMove() {
  if (!activeRep.value || mode.value !== 'drill') return;

  const core = coreFEN(currentFen.value);
  const options = activeRep.value.positions?.[core];

  if (!options || options.length === 0) {
    statusType.value = 'complete';
    statusMessage.value = 'End of repertoire line reached. Click Reset Position to drill again.';
    return;
  }

  const chosen = options[Math.floor(Math.random() * options.length)];
  try {
    const res = await MakeMove(currentFen.value, chosen.uci);
    const san = res.moveSan || chosen.san || chosen.uci;
    currentFen.value = res.fen;
    currentBoard.value = res;
    legalMoves.value = res.legalMoves || [];
    inCheck.value = Boolean(res.inCheck);
    lastMove.value = { from: chosen.uci.slice(0, 2), to: chosen.uci.slice(2, 4) };
    moveHistory.value.push(san);
    fenHistory.value.push(res.fen);
    statusType.value = 'neutral';
    statusMessage.value = `Opponent played ${san}. Your turn.`;
  } catch (e) {
    console.error('Opponent move failed:', e);
  }
}

async function onBoardMove(uci: string) {
  if (!activeRep.value) return;
  const preMoveFen = currentFen.value;

  try {
    const moveRes = await MakeMove(preMoveFen, uci);
    const moveSan = moveRes.moveSan || (moveRes as any).san || uci;

    if (mode.value === 'builder') {
      await AddRepertoireMove(
        activeRep.value.id,
        preMoveFen,
        uci,
        moveSan,
        moveRes.fen,
        moveComment.value
      );
      activeRep.value = await GetRepertoire(activeRep.value.id);
      await refreshMastery(activeRep.value.id);
      currentFen.value = moveRes.fen;
      currentBoard.value = moveRes;
      legalMoves.value = moveRes.legalMoves || [];
      inCheck.value = Boolean(moveRes.inCheck);
      lastMove.value = { from: uci.slice(0, 2), to: uci.slice(2, 4) };
      moveHistory.value.push(moveSan);
      fenHistory.value.push(moveRes.fen);
      statusType.value = 'correct';
      statusMessage.value = `Recorded ${moveSan} (${uci}) into ${activeRep.value.name}.`;
      moveComment.value = '';
      return;
    }

    // Drill Mode
    sessionTotal.value++;
    const res = await VerifyRepertoireMove(activeRep.value.id, preMoveFen, uci);

    if (res.isRepertoire) {
      sessionCorrect.value++;
      await RecordRepertoirePractice(activeRep.value.id, preMoveFen, uci, true);
      activeRep.value = await GetRepertoire(activeRep.value.id);
      await refreshMastery(activeRep.value.id);

      currentFen.value = moveRes.fen;
      currentBoard.value = moveRes;
      legalMoves.value = moveRes.legalMoves || [];
      inCheck.value = Boolean(moveRes.inCheck);
      lastMove.value = { from: uci.slice(0, 2), to: uci.slice(2, 4) };
      moveHistory.value.push(moveSan);
      fenHistory.value.push(moveRes.fen);
      hintMoves.value = [];

      const note = res.playedMove?.comment ? ` — ${res.playedMove.comment}` : '';
      statusType.value = 'correct';
      statusMessage.value = `Correct: ${moveSan}${note}`;

      if (drillStrategy.value === 'systematic' && activeSystematicLine.value) {
        const line = activeSystematicLine.value;
        // If user played an alternative book move that branches into another line, switch to that matching line if needed
        const currStep = line.steps?.[currentStepIndex.value];
        if (currStep && currStep.uci.toLowerCase() !== uci.toLowerCase()) {
          const matchingLine = masteryReport.value?.lines?.find((l: any) =>
            l.steps?.[currentStepIndex.value]?.coreFen === coreFEN(preMoveFen) &&
            l.steps?.[currentStepIndex.value]?.uci?.toLowerCase() === uci.toLowerCase()
          );
          if (matchingLine) {
            currentLineIndex.value = matchingLine.index;
          }
        }
        currentStepIndex.value++;
        setTimeout(() => playNextSystematicOpponentStep(), 350);
      } else {
        const nextCore = coreFEN(moveRes.fen);
        if (activeRep.value.positions?.[nextCore]?.length > 0) {
          setTimeout(() => handleRandomOpponentMove(), 450);
        } else {
          statusType.value = 'complete';
          statusMessage.value = `Line completed (${moveSan}). Book Mastery: ${masteryPct.value}%.`;
        }
      }
    } else {
      currentLineMistakes.value++;
      await RecordRepertoirePractice(activeRep.value.id, preMoveFen, uci, false);
      activeRep.value = await GetRepertoire(activeRep.value.id);
      await refreshMastery(activeRep.value.id);

      statusType.value = 'wrong';
      hintMoves.value = res.alternatives || [];
      if (hintMoves.value.length > 0) {
        const expectedList = hintMoves.value
          .map((m: any) => (m.comment ? `${m.san} (${m.comment})` : m.san))
          .join(' or ');
        statusMessage.value = `${moveSan} breaks book. Expected: ${expectedList}. Try again.`;
      } else {
        statusMessage.value = `${moveSan} is not in your repertoire for this position.`;
      }
    }
  } catch (e) {
    console.error('Move handling failed:', e);
  }
}

async function stepBack() {
  if (fenHistory.value.length <= 1) return;
  fenHistory.value.pop();
  moveHistory.value.pop();
  if (currentStepIndex.value > 0) {
    currentStepIndex.value--;
  }
  const prevFen = fenHistory.value[fenHistory.value.length - 1];
  await applyBoardStateFromFEN(prevFen);
}

function showHint() {
  if (!activeRep.value) return;
  const core = coreFEN(currentFen.value);
  const moves = activeRep.value.positions?.[core] || [];
  hintMoves.value = moves;
  if (moves.length > 0) {
    statusMessage.value = `Book move: ${moves.map((m: any) => m.san).join(', ')}`;
  } else {
    statusMessage.value = 'No book moves defined for this position.';
  }
}

async function handleImportQuickLine() {
  if (!activeRep.value || !quickLineSAN.value.trim()) return;
  try {
    activeRep.value = await AddRepertoireLineSAN(
      activeRep.value.id,
      quickLineSAN.value.trim(),
      quickLineComment.value.trim()
    );
    await refreshMastery(activeRep.value.id);
    repertoires.value = (await GetRepertoires()) || [];
    statusType.value = 'correct';
    statusMessage.value = `Imported line into ${activeRep.value.name}. Total variations: ${masteryReport.value?.totalLines ?? 0}.`;
    quickLineSAN.value = '';
    quickLineComment.value = '';
    syncEditingComments();
  } catch (e: any) {
    statusType.value = 'wrong';
    statusMessage.value = `Line import error: ${e}`;
  }
}

async function handleSaveMoveComment(uci: string) {
  if (!activeRep.value) return;
  try {
    const comment = editingComments.value[uci] || '';
    activeRep.value = await UpdateRepertoireMoveComment(
      activeRep.value.id,
      currentFen.value,
      uci,
      comment
    );
    await refreshMastery(activeRep.value.id);
    statusType.value = 'correct';
    statusMessage.value = `Updated strategic note for ${uci}.`;
  } catch (e: any) {
    statusType.value = 'wrong';
    statusMessage.value = `Failed to update note: ${e}`;
  }
}

async function handleDeleteMove(uci: string, san: string) {
  if (!activeRep.value) return;
  try {
    activeRep.value = await DeleteRepertoireMove(activeRep.value.id, currentFen.value, uci);
    await refreshMastery(activeRep.value.id);
    repertoires.value = (await GetRepertoires()) || [];
    syncEditingComments();
    statusType.value = 'neutral';
    statusMessage.value = `Removed ${san} (${uci}) from current position.`;
  } catch (e: any) {
    statusType.value = 'wrong';
    statusMessage.value = `Failed to delete move: ${e}`;
  }
}

async function handleResetMastery() {
  if (!activeRep.value) return;
  try {
    masteryReport.value = await ResetRepertoireMastery(activeRep.value.id);
    activeRep.value = await GetRepertoire(activeRep.value.id);
    sessionCorrect.value = 0;
    sessionTotal.value = 0;
    currentLineIndex.value = 0;
    await startLineOrResetBoard(0);
    statusType.value = 'neutral';
    statusMessage.value = 'Mastery progress reset to 0%. Ready for a full 100% verification pass.';
  } catch (e: any) {
    console.error('Reset mastery failed:', e);
  }
}

async function handleCompilePolyglot() {
  if (!activeRep.value) return;
  try {
    const out = await CompileRepertoireToPolyglot(activeRep.value.id);
    polyglotStatus.value = `Compiled to ${out}`;
    statusType.value = 'correct';
    statusMessage.value = `Compiled opening book: ${out}`;
  } catch (e: any) {
    statusType.value = 'wrong';
    statusMessage.value = `Polyglot compile failed: ${e}`;
  }
}

async function createNewRepertoire() {
  if (!newRepName.value.trim()) return;
  const rep = {
    id: '',
    name: newRepName.value.trim(),
    color: newRepColor.value,
    description: newRepDesc.value.trim() || 'Custom user repertoire',
    positions: {},
  };
  await SaveRepertoire(rep as any);
  repertoires.value = (await GetRepertoires()) || [];
  showCreateForm.value = false;
  newRepName.value = '';
  newRepDesc.value = '';
  if (repertoires.value.length > 0) {
    const last = repertoires.value[repertoires.value.length - 1];
    await selectRepertoire(last.id);
    mode.value = 'builder';
  }
}

async function deleteCurrentRepertoire() {
  if (!activeRep.value) return;
  await DeleteRepertoire(activeRep.value.id);
  selectedId.value = '';
  activeRep.value = null;
  await loadRepertoireList();
}

async function runDeviationScan() {
  if (!activeRep.value) return;
  isScanning.value = true;
  try {
    deviationReport.value = await ScanRepertoireDeviations({
      repertoireId: activeRep.value.id,
      source: scanSource.value,
      lichessUsername: scanLichessUser.value.trim(),
      chessComUsername: scanChessComUser.value.trim(),
      playerName: scanPlayerName.value.trim(),
      maxGames: Number(scanMaxGames.value) || 150,
    });
  } catch (e: any) {
    statusType.value = 'wrong';
    statusMessage.value = `Scan failed: ${e}`;
  } finally {
    isScanning.value = false;
  }
}

async function drillDeviationPosition(dev: any) {
  mode.value = 'drill';
  drillStrategy.value = 'random';
  await applyBoardStateFromFEN(dev.fen);
  moveHistory.value = dev.lineBeforeSan ? dev.lineBeforeSan.split(' ') : [];
  fenHistory.value = [STARTING_FEN, dev.fen];
  hintMoves.value = dev.expectedMoves || [];
  statusType.value = 'neutral';
  statusMessage.value = `Deviation Drill: You played ${dev.playedMoveSan} here in ${dev.occurrences} game(s). Play your book move ${dev.primaryBookSan} instead.`;
}

async function sendDeviationToBlunderVault(dev: any) {
  if (!activeRep.value) return;
  try {
    await AddBlunderToVault({
      id: `book-dev-${activeRep.value.id}-${dev.moveNumber}-${dev.playedMoveUci}`,
      fen: dev.fen,
      playerName: scanPlayerName.value || scanLichessUser.value || scanChessComUser.value || 'User',
      userColor: activeRep.value.color === 'b' ? 'black' : 'white',
      playedMove: dev.playedMoveUci,
      bestMove: dev.primaryBookUci,
      deltaCp: Math.max(100, dev.evalLossCp || 120),
      event: `Opening Book Deviation (${activeRep.value.name})`,
      date: new Date().toISOString().slice(0, 10).replace(/-/g, '.'),
      explanation: dev.avoidanceAdvice,
      repetitions: 0,
      intervalDays: 1,
      easeFactor: 2.5,
      nextReview: new Date().toISOString(),
      lastReview: new Date().toISOString(),
      status: 'due',
    } as any);
    vaultSavedKeys.value[dev.key] = true;
  } catch (e) {
    console.error('Failed to add deviation to Blunder Vault:', e);
  }
}

async function addSidelineReplyToBook(side: any) {
  if (!activeRep.value || !side.fullLineWithReply) return;
  try {
    activeRep.value = await AddRepertoireLineSAN(
      activeRep.value.id,
      side.fullLineWithReply,
      `Prepared response vs ${side.opponentMoveSan}`
    );
    await refreshMastery(activeRep.value.id);
    sidelineAddedKeys.value[side.key] = true;
  } catch (e) {
    console.error('Failed to add sideline reply to book:', e);
  }
}

async function inspectSidelineOnBoard(side: any) {
  mode.value = 'builder';
  await applyBoardStateFromFEN(side.afterOpponentFen, side.opponentMoveUci);
  fenHistory.value = [STARTING_FEN, side.afterOpponentFen];
  statusType.value = 'neutral';
  statusMessage.value = `Opponent played ${side.opponentMoveSan}. Play your chosen reply on the board to add it to your book.`;
}

onMounted(async () => {
  await loadUserOnlineDefaults();
  await loadRepertoireList();
});
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('close')">
    <div class="trainer-modal">
      <div class="modal-header">
        <div class="header-title">
          <h2>Opening Book Mastery &amp; Deviation Scanner</h2>
        </div>

        <div class="mode-tabs">
          <button
            class="mode-tab"
            :class="{ active: mode === 'drill' }"
            @click="mode = 'drill'; startLineOrResetBoard()"
          >
            Systematic Trainer (100%)
          </button>
          <button
            class="mode-tab"
            :class="{ active: mode === 'builder' }"
            @click="mode = 'builder'"
          >
            Define &amp; Update Book
          </button>
          <button
            class="mode-tab"
            :class="{ active: mode === 'deviations' }"
            @click="mode = 'deviations'"
          >
            Online Book Breaks
          </button>
        </div>

        <button class="close-btn" @click="emit('close')">&times;</button>
      </div>

      <!-- Top Repertoire Bar & 0-100% Mastery Meter -->
      <div class="repertoire-bar">
        <div class="rep-select-row">
          <label class="rep-label">Opening Book:</label>
          <select
            v-model="selectedId"
            class="rep-select"
            @change="selectRepertoire(selectedId)"
          >
            <option v-for="r in repertoires" :key="r.id" :value="r.id">
              [{{ r.color === 'w' ? 'White' : 'Black' }}] {{ r.name }} ({{ r.numMoves }} moves)
            </option>
          </select>
          <button class="btn-secondary btn-sm" @click="showCreateForm = !showCreateForm">
            + New Book
          </button>
          <button
            v-if="activeRep"
            class="btn-secondary btn-sm"
            title="Compile this repertoire into a Polyglot .bin book and link to your profile"
            @click="handleCompilePolyglot"
          >
            Compile .bin Book
          </button>
          <button
            v-if="activeRep && repertoires.length > 1"
            class="btn-danger-outline btn-sm"
            @click="deleteCurrentRepertoire"
          >
            Delete Book
          </button>
        </div>

        <!-- Create Repertoire Drawer -->
        <div v-if="showCreateForm" class="create-drawer">
          <input
            v-model="newRepName"
            type="text"
            placeholder="Book Name (e.g. My Caro-Kann Repertoire)"
            class="form-input"
          />
          <select v-model="newRepColor" class="form-input color-select">
            <option value="w">White Repertoire</option>
            <option value="b">Black Repertoire</option>
          </select>
          <input
            v-model="newRepDesc"
            type="text"
            placeholder="Strategic notes / description"
            class="form-input"
          />
          <button class="btn-primary btn-sm" @click="createNewRepertoire">Create</button>
        </div>

        <!-- 0-100% Book Mastery Bar -->
        <div v-if="masteryReport" class="mastery-strip">
          <div class="mastery-stats">
            <span class="mastery-title">
              Book Mastery: <strong>{{ masteryPct }}%</strong>
              <span v-if="masteryPct >= 100" class="mastered-pill">100% MASTERED</span>
            </span>
            <span class="mastery-detail">
              Positions: <strong>{{ masteryReport.masteredUserNodes }}/{{ masteryReport.totalUserNodes }}</strong>
              &middot;
              Lines: <strong>{{ masteryReport.masteredLines }}/{{ masteryReport.totalLines }}</strong>
              &middot;
              Untested: <strong>{{ masteryReport.untestedUserNodes }}</strong>
              <template v-if="masteryReport.needsWorkUserNodes > 0">
                &middot; Needs Review: <strong class="text-danger">{{ masteryReport.needsWorkUserNodes }}</strong>
              </template>
            </span>
            <button class="btn-link" @click="handleResetMastery">Reset Progress</button>
          </div>
          <div class="mastery-progress-track">
            <div
              class="mastery-progress-fill"
              :class="{ complete: masteryPct >= 100 }"
              :style="{ width: `${Math.min(100, masteryPct)}%` }"
            ></div>
          </div>
        </div>
      </div>

      <!-- TAB 1 & 2: Board + Systematic Trainer / Builder -->
      <div v-if="mode === 'drill' || mode === 'builder'" class="modal-body">
        <div class="board-column">
          <div class="board-wrapper">
            <Chessboard
              :fen="currentFen"
              :legal-moves="legalMoves"
              :piece-set="props.currentPieceSet"
              :board-theme="props.boardTheme"
              :flipped="flipped"
              :last-move="lastMove"
              :in-check="inCheck"
              @move="onBoardMove"
            />
          </div>

          <div class="status-banner" :class="statusType">
            <span class="status-text">{{ statusMessage }}</span>
          </div>
        </div>

        <div class="controls-column">
          <!-- DRILL MODE CONTROLS -->
          <template v-if="mode === 'drill'">
            <div class="control-card">
              <div class="card-header-row">
                <label class="control-label">Training Method</label>
                <div class="pill-toggle">
                  <button
                    class="pill-btn"
                    :class="{ active: drillStrategy === 'systematic' }"
                    @click="drillStrategy = 'systematic'; startLineOrResetBoard()"
                  >
                    Systematic (100% Coverage)
                  </button>
                  <button
                    class="pill-btn"
                    :class="{ active: drillStrategy === 'random' }"
                    @click="drillStrategy = 'random'; startLineOrResetBoard()"
                  >
                    Random Recall
                  </button>
                </div>
              </div>

              <div class="stats-grid">
                <div class="stat-box">
                  <span class="stat-val">{{ sessionCorrect }} / {{ sessionTotal }}</span>
                  <span class="stat-lbl">Session Score</span>
                </div>
                <div class="stat-box">
                  <span class="stat-val">{{ accuracyPercent }}%</span>
                  <span class="stat-lbl">Session Accuracy</span>
                </div>
                <div class="stat-box">
                  <span class="stat-val">{{ masteryPct }}%</span>
                  <span class="stat-lbl">Total Book Mastery</span>
                </div>
              </div>

              <div class="action-buttons-row">
                <button class="btn-secondary" @click="startLineOrResetBoard()">Restart Line</button>
                <button
                  v-if="drillStrategy === 'systematic'"
                  class="btn-primary"
                  @click="advanceToNextUnmasteredLine"
                >
                  Next Unmastered Line &rarr;
                </button>
                <button class="btn-secondary" @click="stepBack" :disabled="fenHistory.length <= 1">
                  Undo
                </button>
                <button class="btn-secondary" @click="showHint">Hint</button>
                <button class="btn-secondary" @click="flipped = !flipped">Flip</button>
              </div>
            </div>

            <!-- Systematic Variation Checklist -->
            <div v-if="masteryReport?.lines?.length" class="control-card lines-checklist-card">
              <label class="control-label">
                Book Variation Lines ({{ masteryReport.masteredLines }}/{{ masteryReport.totalLines }} Mastered)
              </label>
              <div class="lines-list">
                <div
                  v-for="line in masteryReport.lines"
                  :key="line.index"
                  class="line-item"
                  :class="{
                    active: currentLineIndex === line.index && drillStrategy === 'systematic',
                    mastered: line.isMastered,
                  }"
                  @click="drillStrategy = 'systematic'; startLineOrResetBoard(line.index)"
                >
                  <div class="line-item-top">
                    <span class="line-item-title">
                      #{{ line.index + 1 }} {{ line.name }}
                    </span>
                    <span
                      class="line-badge"
                      :class="line.isMastered ? 'badge-mastered' : line.masteredCount > 0 ? 'badge-progress' : 'badge-untested'"
                    >
                      {{ line.isMastered ? 'Mastered' : `${line.masteredCount}/${line.userMovesCount}` }}
                    </span>
                  </div>
                  <div class="line-item-san">{{ line.sanLine }}</div>
                </div>
              </div>
            </div>
          </template>

          <!-- BUILDER MODE CONTROLS -->
          <template v-else-if="mode === 'builder'">
            <!-- Quick PGN / SAN Line Importer -->
            <div class="control-card">
              <label class="control-label">Import or Add Opening Line (SAN / PGN)</label>
              <textarea
                v-model="quickLineSAN"
                rows="2"
                placeholder="e.g. 1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5"
                class="form-input line-textarea"
              ></textarea>
              <div class="import-row">
                <input
                  v-model="quickLineComment"
                  type="text"
                  placeholder="Line name / strategic note (e.g. Two Knights: Polerio Defense)"
                  class="form-input"
                />
                <button class="btn-primary btn-sm" @click="handleImportQuickLine">
                  Add Line to Book
                </button>
              </div>
            </div>

            <!-- Board Recording Note -->
            <div class="control-card">
              <label class="control-label">Board Move Recording Note</label>
              <input
                v-model="moveComment"
                type="text"
                placeholder="Note for next move played on the board..."
                class="form-input"
              />
              <div class="action-buttons-row" style="margin-top: 8px">
                <button class="btn-secondary btn-sm" @click="startLineOrResetBoard()">
                  Reset to Start Position
                </button>
                <button class="btn-secondary btn-sm" @click="stepBack" :disabled="fenHistory.length <= 1">
                  Step Back
                </button>
                <button class="btn-secondary btn-sm" @click="flipped = !flipped">
                  Flip Board
                </button>
              </div>
            </div>

            <!-- Moves at Current Position Editor -->
            <div class="control-card lines-checklist-card">
              <label class="control-label">
                Book Moves at Current Position ({{ currentCoreMoves.length }})
              </label>
              <div v-if="currentCoreMoves.length === 0" class="empty-hint">
                No moves stored at this position yet. Make a move on the board or import a line above.
              </div>
              <div v-else class="pos-moves-list">
                <div v-for="m in currentCoreMoves" :key="m.uci" class="pos-move-row">
                  <div class="pos-move-header">
                    <span class="pos-move-san">{{ m.san }}</span>
                    <span class="pos-move-meta">
                      UCI: {{ m.uci }} &middot; Practiced: {{ m.correct || 0 }}/{{ m.practice || 0 }}
                      (Streak: {{ m.streak || 0 }})
                    </span>
                    <div class="pos-move-actions">
                      <button class="btn-secondary btn-xs" @click="onBoardMove(m.uci)">
                        Play
                      </button>
                      <button class="btn-danger-outline btn-xs" @click="handleDeleteMove(m.uci, m.san)">
                        Delete
                      </button>
                    </div>
                  </div>
                  <div class="pos-move-comment-row">
                    <input
                      v-model="editingComments[m.uci]"
                      type="text"
                      placeholder="Add strategic note or plan for this move..."
                      class="form-input input-sm"
                    />
                    <button class="btn-secondary btn-xs" @click="handleSaveMoveComment(m.uci)">
                      Save Note
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <!-- Move History Strip -->
          <div class="control-card history-card">
            <label class="control-label">Current Line Moves</label>
            <div class="move-list">
              <span v-if="moveHistory.length === 0" class="empty-hint">Starting position</span>
              <span v-for="(m, idx) in moveHistory" :key="idx" class="move-chip">
                <strong v-if="idx % 2 === 0">{{ Math.floor(idx / 2) + 1 }}.</strong>
                {{ m }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 3: Online & Database Book Deviation Scanner -->
      <div v-else-if="mode === 'deviations'" class="deviations-body">
        <div class="scan-controls-card">
          <div class="scan-fields-grid">
            <div class="field-group">
              <label>Game Source</label>
              <select v-model="scanSource" class="form-input">
                <option value="both">Online Accounts + Local DB</option>
                <option value="online">Online Accounts (Lichess / Chess.com)</option>
                <option value="database">Local SCID Database Only</option>
              </select>
            </div>
            <div class="field-group">
              <label>Lichess Username</label>
              <input
                v-model="scanLichessUser"
                type="text"
                placeholder="Lichess handle"
                class="form-input"
              />
            </div>
            <div class="field-group">
              <label>Chess.com Username</label>
              <input
                v-model="scanChessComUser"
                type="text"
                placeholder="Chess.com handle"
                class="form-input"
              />
            </div>
            <div class="field-group">
              <label>Player Name (DB Filter)</label>
              <input
                v-model="scanPlayerName"
                type="text"
                placeholder="Database player name"
                class="form-input"
              />
            </div>
            <div class="field-group field-sm">
              <label>Max Games</label>
              <input
                v-model.number="scanMaxGames"
                type="number"
                min="10"
                max="1000"
                class="form-input"
              />
            </div>
            <div class="field-group field-action">
              <button
                class="btn-primary scan-btn"
                :disabled="isScanning"
                @click="runDeviationScan"
              >
                {{ isScanning ? 'Scanning Games...' : 'Scan for Book Breaks' }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="!deviationReport && !isScanning" class="empty-scan-state">
          <p>
            Scan your Lichess, Chess.com, or local database games against
            <strong>{{ activeRep?.name }}</strong> to find every game where you broke book, see why the played move got into trouble, and drill the position so you never repeat it.
          </p>
        </div>

        <template v-if="deviationReport">
          <!-- Summary Metrics -->
          <div class="scan-summary-strip">
            <div class="summary-metric">
              <span class="metric-val">{{ deviationReport.totalGamesScanned }}</span>
              <span class="metric-lbl">Games Scanned</span>
            </div>
            <div class="summary-metric">
              <span class="metric-val">{{ deviationReport.gamesEnteredBook }}</span>
              <span class="metric-lbl">Entered Book</span>
            </div>
            <div class="summary-metric">
              <span class="metric-val">{{ deviationReport.compliancePct }}%</span>
              <span class="metric-lbl">Book Compliance</span>
            </div>
            <div class="summary-metric">
              <span class="metric-val text-danger">{{ deviationReport.userDeviatedGames }}</span>
              <span class="metric-lbl">Times You Broke Book</span>
            </div>
            <div class="summary-metric">
              <span class="metric-val">{{ deviationReport.opponentSidelineGames }}</span>
              <span class="metric-lbl">Opponent Sidelines</span>
            </div>
          </div>

          <div class="deviations-columns">
            <!-- Column 1: User Book Deviations (Times User Broke Book) -->
            <div class="deviation-section">
              <h3 class="section-heading">
                Times You Broke Book ({{ deviationReport.userDeviations?.length || 0 }})
              </h3>
              <div v-if="!deviationReport.userDeviations?.length" class="empty-hint">
                Zero user book breaks detected across the scanned games.
              </div>
              <div
                v-for="dev in deviationReport.userDeviations"
                :key="dev.key"
                class="deviation-card"
              >
                <div class="dev-card-header">
                  <span class="dev-move-num">
                    Move {{ dev.moveNumber }}
                    <template v-if="dev.lineBeforeSan">({{ dev.lineBeforeSan }})</template>
                  </span>
                  <span class="dev-freq-badge">
                    {{ dev.occurrences }}x &middot; Score: {{ dev.winRatePct }}% ({{ dev.wins }}W/{{ dev.draws }}D/{{ dev.losses }}L)
                  </span>
                </div>

                <div class="dev-comparison-row">
                  <span class="bad-move-pill">
                    You Played: <strong>{{ dev.playedMoveSan }}</strong>
                  </span>
                  <span class="vs-arrow">&rarr;</span>
                  <span class="book-move-pill">
                    Book Move: <strong>{{ dev.primaryBookSan }}</strong>
                  </span>
                  <span v-if="dev.evalLossCp > 0" class="eval-drop-pill">
                    Eval Drop: -{{ (dev.evalLossCp / 100).toFixed(2) }}
                    <template v-if="dev.punishingReplySan">
                      (allows {{ dev.punishingReplySan }})
                    </template>
                  </span>
                </div>

                <div class="avoidance-box">
                  <strong>How to Avoid:</strong> {{ dev.avoidanceAdvice }}
                </div>

                <div class="dev-actions-row">
                  <button class="btn-primary btn-xs" @click="drillDeviationPosition(dev)">
                    Drill Position on Board
                  </button>
                  <button
                    class="btn-secondary btn-xs"
                    :disabled="vaultSavedKeys[dev.key]"
                    @click="sendDeviationToBlunderVault(dev)"
                  >
                    {{ vaultSavedKeys[dev.key] ? 'Saved to Blunder Vault' : 'Add to Blunder Vault (SM-2)' }}
                  </button>
                </div>
              </div>
            </div>

            <!-- Column 2: Uncovered Opponent Sidelines -->
            <div class="deviation-section">
              <h3 class="section-heading">
                Uncovered Opponent Sidelines ({{ deviationReport.opponentSidelines?.length || 0 }})
              </h3>
              <div v-if="!deviationReport.opponentSidelines?.length" class="empty-hint">
                No uncovered opponent sidelines found.
              </div>
              <div
                v-for="side in deviationReport.opponentSidelines"
                :key="side.key"
                class="deviation-card sideline-card"
              >
                <div class="dev-card-header">
                  <span class="dev-move-num">
                    Move {{ side.moveNumber }}
                    <template v-if="side.lineBeforeSan">({{ side.lineBeforeSan }})</template>
                  </span>
                  <span class="dev-freq-badge">
                    {{ side.occurrences }}x &middot; Score: {{ side.winRatePct }}%
                  </span>
                </div>

                <div class="dev-comparison-row">
                  <span class="sideline-pill">
                    Opponent Played: <strong>{{ side.opponentMoveSan }}</strong>
                  </span>
                  <span v-if="side.recommendedReplySan" class="book-move-pill">
                    Recommended Reply: <strong>{{ side.recommendedReplySan }}</strong>
                  </span>
                </div>

                <div class="avoidance-box">
                  {{ side.remediationAdvice }}
                </div>

                <div class="dev-actions-row">
                  <button
                    v-if="side.fullLineWithReply"
                    class="btn-primary btn-xs"
                    :disabled="sidelineAddedKeys[side.key]"
                    @click="addSidelineReplyToBook(side)"
                  >
                    {{ sidelineAddedKeys[side.key] ? 'Added to Book' : `Add ${side.recommendedReplySan} to Book` }}
                  </button>
                  <button class="btn-secondary btn-xs" @click="inspectSidelineOnBoard(side)">
                    Choose Custom Reply on Board
                  </button>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
   inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.trainer-modal {
  background: var(--bg-secondary, #1e1e2e);
  color: var(--text-primary, #cdd6f4);
  width: 1080px;
  max-width: 96vw;
  max-height: 92vh;
  border-radius: 12px;
  border: 1px solid var(--border-color, #313244);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.6);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  background: var(--bg-tertiary, #181825);
  border-bottom: 1px solid var(--border-color, #313244);
}

.header-title h2 {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 700;
}

.mode-tabs {
  display: flex;
  gap: 6px;
  background: rgba(0, 0, 0, 0.25);
  padding: 4px;
  border-radius: 8px;
}

.mode-tab {
  background: transparent;
  border: none;
  color: var(--text-secondary, #a6adc8);
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
}

.mode-tab.active {
  background: var(--accent-color, #89b4fa);
  color: #11111b;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary, #a6adc8);
  font-size: 1.5rem;
  cursor: pointer;
}

.repertoire-bar {
  padding: 10px 18px;
  background: rgba(0, 0, 0, 0.18);
  border-bottom: 1px solid var(--border-color, #313244);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rep-select-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.rep-label {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-secondary, #a6adc8);
}

.rep-select {
  flex: 1;
  min-width: 240px;
  padding: 6px 10px;
  border-radius: 6px;
  background: var(--bg-primary, #11111b);
  color: var(--text-primary, #cdd6f4);
  border: 1px solid var(--border-color, #45475a);
  font-size: 0.85rem;
}

.create-drawer {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  padding-top: 4px;
}

.color-select {
  max-width: 160px;
}

.mastery-strip {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mastery-stats {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.8rem;
  flex-wrap: wrap;
  gap: 8px;
}

.mastered-pill {
  background: rgba(166, 227, 161, 0.2);
  color: #a6e3a1;
  border: 1px solid #a6e3a1;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 0.72rem;
  margin-left: 6px;
}

.mastery-detail {
  color: var(--text-secondary, #a6adc8);
}

.text-danger {
  color: #f38ba8;
}

.btn-link {
  background: none;
  border: none;
  color: var(--accent-color, #89b4fa);
  font-size: 0.78rem;
  cursor: pointer;
  text-decoration: underline;
}

.mastery-progress-track {
  height: 7px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 4px;
  overflow: hidden;
}

.mastery-progress-fill {
  height: 100%;
  background: var(--accent-color, #89b4fa);
  transition: width 0.25s ease;
}

.mastery-progress-fill.complete {
  background: #a6e3a1;
}

.modal-body {
  display: grid;
  grid-template-columns: 430px 1fr;
  gap: 16px;
  padding: 16px 18px;
  overflow-y: auto;
}

.board-column {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.board-wrapper {
  width: 410px;
  height: 410px;
  border-radius: 8px;
  overflow: hidden;
  border: 2px solid var(--border-color, #313244);
}

.status-banner {
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  border: 1px solid var(--border-color, #313244);
  background: rgba(255, 255, 255, 0.04);
}

.status-banner.correct {
  background: rgba(166, 227, 161, 0.14);
  border-color: #a6e3a1;
  color: #a6e3a1;
}

.status-banner.wrong {
  background: rgba(243, 139, 168, 0.14);
  border-color: #f38ba8;
  color: #f38ba8;
}

.status-banner.complete {
  background: rgba(137, 180, 250, 0.14);
  border-color: #89b4fa;
  color: #89b4fa;
}

.controls-column {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.control-card {
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--border-color, #313244);
  border-radius: 8px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.card-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.control-label {
  font-size: 0.74rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary, #a6adc8);
  font-weight: 700;
}

.pill-toggle {
  display: flex;
  gap: 4px;
  background: rgba(0, 0, 0, 0.3);
  padding: 3px;
  border-radius: 6px;
}

.pill-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary, #a6adc8);
  font-size: 0.75rem;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
}

.pill-btn.active {
  background: var(--accent-color, #89b4fa);
  color: #11111b;
  font-weight: 600;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.stat-box {
  background: rgba(255, 255, 255, 0.03);
  padding: 8px;
  border-radius: 6px;
  text-align: center;
  border: 1px solid var(--border-color, #313244);
}

.stat-val {
  display: block;
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--accent-color, #89b4fa);
}

.stat-lbl {
  font-size: 0.72rem;
  color: var(--text-secondary, #a6adc8);
}

.action-buttons-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.lines-checklist-card {
  max-height: 230px;
  overflow-y: auto;
}

.lines-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.line-item {
  padding: 8px 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color, #313244);
  cursor: pointer;
}

.line-item.active {
  border-color: var(--accent-color, #89b4fa);
  background: rgba(137, 180, 250, 0.1);
}

.line-item.mastered {
  border-left: 3px solid #a6e3a1;
}

.line-item-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.82rem;
  font-weight: 600;
}

.line-item-san {
  font-size: 0.76rem;
  color: var(--text-secondary, #a6adc8);
  font-family: monospace;
  margin-top: 2px;
}

.line-badge {
  font-size: 0.7rem;
  padding: 1px 6px;
  border-radius: 4px;
}

.badge-mastered {
  background: rgba(166, 227, 161, 0.18);
  color: #a6e3a1;
}

.badge-progress {
  background: rgba(249, 226, 175, 0.18);
  color: #f9e2af;
}

.badge-untested {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-secondary, #a6adc8);
}

.line-textarea {
  resize: vertical;
  font-family: monospace;
}

.import-row {
  display: flex;
  gap: 8px;
}

.pos-moves-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.pos-move-row {
  padding: 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color, #313244);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pos-move-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.pos-move-san {
  font-weight: 700;
  color: var(--accent-color, #89b4fa);
}

.pos-move-meta {
  font-size: 0.75rem;
  color: var(--text-secondary, #a6adc8);
}

.pos-move-actions {
  display: flex;
  gap: 4px;
}

.pos-move-comment-row {
  display: flex;
  gap: 6px;
}

.history-card {
  max-height: 90px;
  overflow-y: auto;
}

.move-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 0.82rem;
}

.move-chip {
  background: rgba(255, 255, 255, 0.05);
  padding: 2px 6px;
  border-radius: 4px;
}

/* Deviations Tab */
.deviations-body {
  padding: 14px 18px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.scan-controls-card {
  background: rgba(0, 0, 0, 0.22);
  border: 1px solid var(--border-color, #313244);
  border-radius: 8px;
  padding: 12px;
}

.scan-fields-grid {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr 1fr 90px auto;
  gap: 8px;
  align-items: end;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-group label {
  font-size: 0.72rem;
  color: var(--text-secondary, #a6adc8);
  font-weight: 600;
}

.scan-summary-strip {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 10px;
}

.summary-metric {
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--border-color, #313244);
  border-radius: 8px;
  padding: 10px;
  text-align: center;
}

.metric-val {
  display: block;
  font-size: 1.15rem;
  font-weight: 700;
}

.metric-lbl {
  font-size: 0.74rem;
  color: var(--text-secondary, #a6adc8);
}

.deviations-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.deviation-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.section-heading {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 700;
}

.deviation-card {
  background: rgba(0, 0, 0, 0.22);
  border: 1px solid var(--border-color, #313244);
  border-left: 3px solid #f38ba8;
  border-radius: 8px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sideline-card {
  border-left-color: #f9e2af;
}

.dev-card-header {
  display: flex;
  justify-content: space-between;
  font-size: 0.78rem;
  color: var(--text-secondary, #a6adc8);
}

.dev-comparison-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 0.82rem;
}

.bad-move-pill {
  background: rgba(243, 139, 168, 0.16);
  color: #f38ba8;
  padding: 2px 8px;
  border-radius: 4px;
}

.book-move-pill {
  background: rgba(166, 227, 161, 0.16);
  color: #a6e3a1;
  padding: 2px 8px;
  border-radius: 4px;
}

.sideline-pill {
  background: rgba(249, 226, 175, 0.16);
  color: #f9e2af;
  padding: 2px 8px;
  border-radius: 4px;
}

.eval-drop-pill {
  font-size: 0.75rem;
  color: #fab387;
}

.avoidance-box {
  font-size: 0.8rem;
  line-height: 1.4;
  background: rgba(255, 255, 255, 0.03);
  padding: 8px;
  border-radius: 6px;
}

.dev-actions-row {
  display: flex;
  gap: 8px;
}

.form-input {
  padding: 7px 10px;
  border-radius: 6px;
  background: var(--bg-primary, #11111b);
  color: var(--text-primary, #cdd6f4);
  border: 1px solid var(--border-color, #45475a);
  font-size: 0.82rem;
  width: 100%;
  box-sizing: border-box;
}

.input-sm {
  padding: 4px 8px;
  font-size: 0.78rem;
}

.btn-primary {
  background: var(--accent-color, #89b4fa);
  color: #11111b;
  border: none;
  padding: 7px 12px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  font-size: 0.82rem;
}

.btn-secondary {
  background: var(--bg-tertiary, #313244);
  color: var(--text-primary, #cdd6f4);
  border: 1px solid var(--border-color, #45475a);
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 0.8rem;
  cursor: pointer;
}

.btn-danger-outline {
  background: transparent;
  color: #f38ba8;
  border: 1px solid #f38ba8;
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 0.8rem;
  cursor: pointer;
}

.btn-sm {
  padding: 5px 10px;
  font-size: 0.78rem;
}

.btn-xs {
  padding: 4px 8px;
  font-size: 0.74rem;
}

.empty-hint,
.empty-scan-state {
  color: var(--text-secondary, #6c7086);
  font-size: 0.82rem;
}
</style>
