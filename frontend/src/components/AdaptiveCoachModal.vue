<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import Chessboard from './Chessboard.vue';
import {
  GetAdaptiveCoachDashboard,
  GenerateAdaptiveCoachingSession,
  RecordAdaptiveCoachAttempt,
  GetMainUserOnlineDefaults,
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
}>();

const STARTING_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1';

const activeTab = ref<'drill' | 'progress'>('drill');

// Scan configuration
const scanSource = ref<'both' | 'online' | 'database'>('both');
const lichessUsername = ref<string>('');
const chessComUsername = ref<string>('');
const playerName = ref<string>('');
const focusTheme = ref<string>('auto');
const maxGames = ref<number>(120);
const batchSize = ref<number>(12);
const isScanning = ref<boolean>(false);

// Dashboard & Active Drill state
const dashboard = ref<any | null>(null);
const currentDrillIndex = ref<number>(0);
const currentStepIdx = ref<number>(0);
const hadMistakeOnCurrent = ref<boolean>(false);
const drillStartTimeMs = ref<number>(Date.now());
const showVisualCues = ref<boolean>(true);
const showTakeaway = ref<boolean>(false);
const reinforcementNotice = ref<string>('');

// Board state
const currentFen = ref<string>(STARTING_FEN);
const legalMoves = ref<string[]>([]);
const inCheck = ref<boolean>(false);
const lastMove = ref<{ from: string; to: string } | null>(null);
const flipped = ref<boolean>(false);
const statusType = ref<'neutral' | 'correct' | 'wrong' | 'complete'>('neutral');
const statusMessage = ref<string>('Scan your online or database games to generate a customized coaching session.');

const activeQueue = computed(() => dashboard.value?.activeQueue || []);

const activeDrill = computed(() => {
  if (!activeQueue.value.length) return null;
  const idx = Math.max(0, Math.min(currentDrillIndex.value, activeQueue.value.length - 1));
  return activeQueue.value[idx] || null;
});

function parseUciMove(uci: string): { from: string; to: string } | null {
  if (!uci || uci.length < 4) return null;
  return { from: uci.slice(0, 2), to: uci.slice(2, 4) };
}

async function setBoardFen(fen: string, moveUci: string = '') {
  currentFen.value = fen;
  lastMove.value = parseUciMove(moveUci);
  try {
    const state = await ParseFEN(fen);
    if (state) {
      legalMoves.value = state.legalMoves || [];
      inCheck.value = Boolean(state.inCheck);
    }
  } catch (e) {
    legalMoves.value = [];
    inCheck.value = false;
  }
}

async function loadDefaultsAndDashboard() {
  try {
    const defaults = await GetMainUserOnlineDefaults();
    if (defaults) {
      if (defaults.lichessUsername) lichessUsername.value = defaults.lichessUsername;
      if (defaults.chessComUsername) chessComUsername.value = defaults.chessComUsername;
      if (defaults.displayName) playerName.value = defaults.displayName;
    }
  } catch (e) {
    // Ignore in test environments
  }

  try {
    dashboard.value = await GetAdaptiveCoachDashboard();
    if (activeQueue.value.length > 0) {
      const firstUnattempted = activeQueue.value.findIndex((d: any) => !d.attempted);
      await loadDrillAtIndex(firstUnattempted >= 0 ? firstUnattempted : 0);
    } else {
      await setBoardFen(STARTING_FEN);
    }
  } catch (e) {
    console.error('Failed to load adaptive coach dashboard:', e);
  }
}

async function handleGenerateSession(overrideTheme?: string) {
  if (overrideTheme) {
    focusTheme.value = overrideTheme;
  }
  isScanning.value = true;
  reinforcementNotice.value = '';
  try {
    dashboard.value = await GenerateAdaptiveCoachingSession({
      source: scanSource.value,
      lichessUsername: lichessUsername.value.trim(),
      chessComUsername: chessComUsername.value.trim(),
      playerName: playerName.value.trim(),
      focusTheme: focusTheme.value,
      maxGames: Number(maxGames.value) || 120,
      batchSize: Number(batchSize.value) || 12,
    });
    activeTab.value = 'drill';
    if (activeQueue.value.length > 0) {
      await loadDrillAtIndex(0);
    }
  } catch (e: any) {
    statusType.value = 'wrong';
    statusMessage.value = `Failed to generate coaching session: ${e}`;
  } finally {
    isScanning.value = false;
  }
}

async function loadDrillAtIndex(idx: number) {
  if (!activeQueue.value.length) return;
  const clamped = Math.max(0, Math.min(idx, activeQueue.value.length - 1));
  currentDrillIndex.value = clamped;
  currentStepIdx.value = 0;
  hadMistakeOnCurrent.value = false;
  showTakeaway.value = false;
  reinforcementNotice.value = '';
  drillStartTimeMs.value = Date.now();

  const d = activeQueue.value[clamped];
  await setBoardFen(d.fen);
  flipped.value = d.turn === 'b';
  statusType.value = 'neutral';
  statusMessage.value = `${d.diagnosisHeadline} — Find the best move for ${d.turn === 'b' ? 'Black' : 'White'}.`;
}

async function onBoardMove(uci: string) {
  const d = activeDrill.value;
  if (!d || !d.solutionUci || !d.solutionUci.length) return;

  const expectedUCI = d.solutionUci[currentStepIdx.value];
  if (!expectedUCI) return;

  try {
    const moveRes = await MakeMove(currentFen.value, uci);
    const playedSan = moveRes.moveSan || uci;

    if (uci.toLowerCase() === expectedUCI.toLowerCase()) {
      currentFen.value = moveRes.fen;
      legalMoves.value = moveRes.legalMoves || [];
      inCheck.value = Boolean(moveRes.inCheck);
      lastMove.value = parseUciMove(moveRes.moveUci || uci);
      currentStepIdx.value++;

      if (currentStepIdx.value >= d.solutionUci.length) {
        // Completed the drill!
        const elapsedSec = Math.max(1, Math.round((Date.now() - drillStartTimeMs.value) / 1000));
        const solvedCleanly = !hadMistakeOnCurrent.value;
        showTakeaway.value = true;
        statusType.value = 'complete';
        statusMessage.value = `Solved (${playedSan}). ${d.postSolveTakeaway}`;

        if (!d.attempted || solvedCleanly) {
          const res = await RecordAdaptiveCoachAttempt(d.id, solvedCleanly, elapsedSec);
          if (res?.dashboard) {
            dashboard.value = res.dashboard;
          }
        }
      } else {
        // Play opponent response in multi-move tactical sequence
        statusType.value = 'correct';
        statusMessage.value = `Good move (${playedSan}). Calculating opponent reply...`;
        const oppUCI = d.solutionUci[currentStepIdx.value];
        setTimeout(async () => {
          try {
            const oppRes = await MakeMove(currentFen.value, oppUCI);
            currentFen.value = oppRes.fen;
            legalMoves.value = oppRes.legalMoves || [];
            inCheck.value = Boolean(oppRes.inCheck);
            lastMove.value = parseUciMove(oppRes.moveUci || oppUCI);
            currentStepIdx.value++;
            statusMessage.value = `Opponent played ${oppRes.moveSan || oppUCI}. Find the follow-up move.`;
          } catch (err) {
            console.error('Opponent reply failed:', err);
          }
        }, 300);
      }
    } else {
      // Wrong move! Record miss on first mistake so the coach immediately pushes this theme harder
      const isFirstMissOnDrill = !hadMistakeOnCurrent.value;
      hadMistakeOnCurrent.value = true;
      showVisualCues.value = true;
      showTakeaway.value = true;
      statusType.value = 'wrong';
      const expectedSAN = d.solutionSan?.[currentStepIdx.value] || expectedUCI;
      statusMessage.value = `${playedSan} misses the tactic. Study the Coaching Protocol on the right and play ${expectedSAN}.`;

      if (isFirstMissOnDrill) {
        const elapsedSec = Math.max(1, Math.round((Date.now() - drillStartTimeMs.value) / 1000));
        const res = await RecordAdaptiveCoachAttempt(d.id, false, elapsedSec);
        if (res?.dashboard) {
          dashboard.value = res.dashboard;
        }
        if (res?.reinforcementAdded && res?.addedDrill) {
          reinforcementNotice.value = `Adaptive Push Triggered: Because you missed this ${d.themeName} position, its priority weight increased to ${res.updatedTheme?.priorityWeight} and an extra ${d.themeName} drill was added to your queue.`;
        }
      }
    }
  } catch (e) {
    console.error('Move validation failed:', e);
  }
}

async function nextDrill() {
  if (currentDrillIndex.value + 1 < activeQueue.value.length) {
    await loadDrillAtIndex(currentDrillIndex.value + 1);
  } else if (activeQueue.value.length > 0) {
    await loadDrillAtIndex(0);
  }
}

async function prevDrill() {
  if (currentDrillIndex.value > 0) {
    await loadDrillAtIndex(currentDrillIndex.value - 1);
  }
}

function revealSolutionHint() {
  const d = activeDrill.value;
  if (!d) return;
  showVisualCues.value = true;
  const expectedSAN = d.solutionSan?.[currentStepIdx.value] || d.solutionUci?.[currentStepIdx.value] || '';
  statusMessage.value = `Coach Hint: Look at ${expectedSAN} (${d.themeName}).`;
}

onMounted(async () => {
  await loadDefaultsAndDashboard();
});
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('close')">
    <div class="coach-modal">
      <!-- Header -->
      <div class="modal-header">
        <div class="header-title">
          <h2>Adaptive Weakness Coach &amp; Online Game Trainer</h2>
        </div>

        <div class="mode-tabs">
          <button
            class="mode-tab"
            :class="{ active: activeTab === 'drill' }"
            @click="activeTab = 'drill'"
          >
            Interactive Coaching Session ({{ activeQueue.length }})
          </button>
          <button
            class="mode-tab"
            :class="{ active: activeTab === 'progress' }"
            @click="activeTab = 'progress'"
          >
            Weakness Profile &amp; Progress History
          </button>
        </div>

        <button class="close-btn" @click="emit('close')">&times;</button>
      </div>

      <!-- Scan & Custom Curriculum Generator Bar -->
      <div class="scanner-bar">
        <div class="scanner-inputs-row">
          <div class="input-col">
            <label>Source</label>
            <select v-model="scanSource" class="form-input">
              <option value="both">Online + Local DB</option>
              <option value="online">Online Accounts (Lichess / Chess.com)</option>
              <option value="database">Local SCID Database</option>
            </select>
          </div>
          <div class="input-col">
            <label>Lichess Handle</label>
            <input v-model="lichessUsername" type="text" placeholder="Lichess user" class="form-input" />
          </div>
          <div class="input-col">
            <label>Chess.com Handle</label>
            <input v-model="chessComUsername" type="text" placeholder="Chess.com user" class="form-input" />
          </div>
          <div class="input-col">
            <label>DB Player Name</label>
            <input v-model="playerName" type="text" placeholder="Player name" class="form-input" />
          </div>
          <div class="input-col">
            <label>Target Weakness Focus</label>
            <select v-model="focusTheme" class="form-input">
              <option value="auto">Auto (Push Top Online Weaknesses)</option>
              <option value="hangingPiece">Hanging Pieces &amp; Board Vision</option>
              <option value="fork">Forks / Double Attacks</option>
              <option value="pin">Pins &amp; Alignment</option>
              <option value="skewer">Skewers</option>
              <option value="discoveredAttack">Discovered Attacks</option>
              <option value="backRankMate">Back-Rank &amp; Mating Patterns</option>
              <option value="deflection">Deflection &amp; Overloaded Defenders</option>
              <option value="sacrifice">Tactical Sacrifices</option>
              <option value="endgame">Endgame Tactics</option>
            </select>
          </div>
          <div class="input-col action-col">
            <button
              class="btn-primary generate-btn"
              :disabled="isScanning"
              @click="handleGenerateSession()"
            >
              {{ isScanning ? 'Analyzing Games...' : 'Scan & Build Training' }}
            </button>
          </div>
        </div>

        <div v-if="dashboard" class="coach-summary-strip">
          <span class="summary-text">{{ dashboard.coachSummaryText }}</span>
          <div class="summary-badges">
            <span class="stat-pill">
              Overall Mastery: <strong>{{ dashboard.overallMasteryPct }}%</strong>
            </span>
            <span class="stat-pill">
              Drills Solved: <strong>{{ dashboard.totalDrillsSolved }}</strong>
            </span>
            <span v-if="dashboard.topWeaknesses?.length" class="stat-pill priority-pill">
              #1 Priority Push: <strong>{{ dashboard.topWeaknesses[0].themeName }}</strong>
              (Weight {{ dashboard.topWeaknesses[0].priorityWeight }})
            </span>
          </div>
        </div>
      </div>

      <!-- TAB 1: Interactive High-Instructional Coaching Drill -->
      <div v-if="activeTab === 'drill'" class="modal-body">
        <div class="board-column">
          <div class="board-wrapper">
            <Chessboard
              :fen="currentFen"
              :legal-moves="legalMoves"
              :flipped="flipped"
              :piece-set="props.currentPieceSet"
              :board-theme="props.boardTheme"
              :last-move="lastMove"
              :in-check="inCheck"
              @move="onBoardMove"
            />
          </div>

          <div class="status-banner" :class="statusType">
            <span>{{ statusMessage }}</span>
          </div>

          <div v-if="reinforcementNotice" class="reinforcement-alert">
            {{ reinforcementNotice }}
          </div>

          <div class="board-controls-row">
            <button class="btn-secondary" :disabled="currentDrillIndex <= 0" @click="prevDrill">
              &larr; Prev Drill
            </button>
            <button class="btn-secondary" @click="loadDrillAtIndex(currentDrillIndex)">
              Retry Position
            </button>
            <button class="btn-secondary" @click="revealSolutionHint">
              Coach Hint
            </button>
            <button class="btn-secondary" @click="flipped = !flipped">
              Flip
            </button>
            <button class="btn-primary" :disabled="!activeQueue.length" @click="nextDrill">
              Next Drill &rarr;
            </button>
          </div>
        </div>

        <div class="coaching-column">
          <template v-if="activeDrill">
            <div class="instruction-card">
              <div class="drill-meta-header">
                <span
                  class="source-badge"
                  :class="activeDrill.sourceType === 'online_game_mistake' ? 'badge-game' : 'badge-reinforce'"
                >
                  {{ activeDrill.sourceType === 'online_game_mistake' ? 'From Your Game' : 'Targeted Weakness Push' }}
                </span>
                <span class="theme-badge">{{ activeDrill.themeName }}</span>
                <span class="tier-badge">{{ activeDrill.difficultyTier }} ({{ activeDrill.estimatedRating }})</span>
              </div>

              <h3 class="drill-headline">{{ activeDrill.diagnosisHeadline }}</h3>

              <div class="coaching-section">
                <div class="section-label">1. Diagnosis: What Happened</div>
                <p class="section-body">{{ activeDrill.whatWentWrong }}</p>
              </div>

              <div v-if="showVisualCues && activeDrill.visualCues?.length" class="coaching-section">
                <div class="section-label">2. Board Vision Cues to Spot</div>
                <ul class="cues-list">
                  <li v-for="(cue, cIdx) in activeDrill.visualCues" :key="cIdx">{{ cue }}</li>
                </ul>
              </div>

              <div class="coaching-section">
                <div class="section-label">3. High-Instruction Thinking Protocol</div>
                <p class="section-body protocol-box">{{ activeDrill.stepByStepCoach }}</p>
              </div>

              <div v-if="showTakeaway" class="coaching-section takeaway-section">
                <div class="section-label">4. Key Lesson &amp; Solution Breakdown</div>
                <p class="section-body">{{ activeDrill.postSolveTakeaway }}</p>
              </div>
            </div>

            <!-- Active Session Queue -->
            <div class="instruction-card queue-card">
              <div class="section-label">
                Customized Drill Queue ({{ currentDrillIndex + 1 }} / {{ activeQueue.length }})
              </div>
              <div class="queue-list">
                <div
                  v-for="(item, idx) in activeQueue"
                  :key="item.id"
                  class="queue-item"
                  :class="{
                    active: idx === currentDrillIndex,
                    solved: item.attempted && item.solvedFirstTry,
                    missed: item.attempted && !item.solvedFirstTry,
                  }"
                  @click="loadDrillAtIndex(idx)"
                >
                  <span class="queue-idx">#{{ idx + 1 }}</span>
                  <span class="queue-theme">{{ item.themeName }}</span>
                  <span class="queue-source">
                    {{ item.sourceType === 'online_game_mistake' ? `${item.white} vs ${item.black}` : 'Reinforcement' }}
                  </span>
                  <span class="queue-status">
                    {{ !item.attempted ? 'Pending' : item.solvedFirstTry ? 'Solved' : 'Needs Review' }}
                  </span>
                </div>
              </div>
            </div>
          </template>

          <div v-else class="empty-coach-state">
            <p>
              Click <strong>Scan &amp; Build Training</strong> above to analyze your online Lichess/Chess.com or local database games and generate a customized coaching session.
            </p>
          </div>
        </div>
      </div>

      <!-- TAB 2: Weakness Profile & Progress History -->
      <div v-else-if="activeTab === 'progress'" class="progress-body">
        <div class="progress-section">
          <h3 class="section-heading">Diagnosed Tactical Profile &amp; Adaptive Push Weights</h3>
          <div class="weakness-table-wrap">
            <table class="weakness-table">
              <thead>
                <tr>
                  <th>Tactical Motif</th>
                  <th>Game Misses</th>
                  <th>Drills (Solved / Total)</th>
                  <th>Streak</th>
                  <th>Mastery</th>
                  <th>Adaptive Push Weight</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="st in dashboard?.allThemeStats || []" :key="st.themeKey">
                  <td>
                    <strong>{{ st.themeName }}</strong>
                    <div class="habit-sub">{{ st.coachingHabit }}</div>
                  </td>
                  <td>
                    <span :class="{ 'text-danger': st.gameMissCount > 0 }">
                      {{ st.gameMissCount }}
                    </span>
                  </td>
                  <td>{{ st.drillsSolved }} / {{ st.drillsAttempted }} ({{ st.accuracyPct }}%)</td>
                  <td>{{ st.currentStreak }} (Best {{ st.bestStreak }})</td>
                  <td>
                    <div class="mini-mastery-bar">
                      <div class="mini-mastery-fill" :style="{ width: `${Math.min(100, st.masteryScore)}%` }"></div>
                    </div>
                    <span class="mini-pct">{{ st.masteryScore }}%</span>
                  </td>
                  <td>
                    <span class="weight-badge" :class="{ high: st.priorityWeight >= 140 }">
                      {{ st.priorityWeight }}
                    </span>
                  </td>
                  <td>
                    <button class="btn-primary btn-xs" @click="handleGenerateSession(st.themeKey)">
                      Drill Harder
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="progress-section">
          <h3 class="section-heading">Completed Coaching Sessions ({{ dashboard?.sessionHistory?.length || 0 }})</h3>
          <div v-if="!dashboard?.sessionHistory?.length" class="empty-hint">
            Complete your current drill queue to log your first session into permanent training history.
          </div>
          <div v-else class="history-grid">
            <div v-for="sess in dashboard.sessionHistory" :key="sess.sessionId" class="history-card">
              <div class="history-top">
                <strong>{{ sess.completedAt ? sess.completedAt.slice(0, 10) : sess.startedAt.slice(0, 10) }}</strong>
                <span>Accuracy: {{ sess.sessionAccuracy }}% ({{ sess.firstTrySolved }}/{{ sess.totalDrills }})</span>
              </div>
              <div class="history-detail">
                Focus: {{ (sess.focusThemes || []).join(', ') }} &middot;
                Extra Reinforcements Added: {{ sess.reinforcedOnMiss }} &middot;
                Mastery: {{ sess.overallMasteryPct }}%
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.78);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.coach-modal {
  background: var(--bg-secondary, #1e1e2e);
  color: var(--text-primary, #cdd6f4);
  width: 1120px;
  max-width: 96vw;
  max-height: 92vh;
  border-radius: 12px;
  border: 1px solid var(--border-color, #313244);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.65);
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

.scanner-bar {
  padding: 10px 18px;
  background: rgba(0, 0, 0, 0.2);
  border-bottom: 1px solid var(--border-color, #313244);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.scanner-inputs-row {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr 0.9fr 0.9fr 1.3fr auto;
  gap: 8px;
  align-items: end;
}

.input-col {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.input-col label {
  font-size: 0.72rem;
  color: var(--text-secondary, #a6adc8);
  font-weight: 600;
}

.coach-summary-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  font-size: 0.8rem;
  padding-top: 4px;
}

.summary-text {
  color: var(--text-secondary, #a6adc8);
}

.summary-badges {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.stat-pill {
  background: rgba(255, 255, 255, 0.06);
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 0.76rem;
}

.priority-pill {
  background: rgba(243, 139, 168, 0.16);
  color: #f38ba8;
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
  font-size: 0.84rem;
  font-weight: 600;
  border: 1px solid var(--border-color, #313244);
  background: rgba(255, 255, 255, 0.04);
}

.status-banner.correct,
.status-banner.complete {
  background: rgba(166, 227, 161, 0.14);
  border-color: #a6e3a1;
  color: #a6e3a1;
}

.status-banner.wrong {
  background: rgba(243, 139, 168, 0.14);
  border-color: #f38ba8;
  color: #f38ba8;
}

.reinforcement-alert {
  background: rgba(249, 226, 175, 0.15);
  border: 1px solid #f9e2af;
  color: #f9e2af;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 0.78rem;
  font-weight: 600;
}

.board-controls-row {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.coaching-column {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.instruction-card {
  background: rgba(0, 0, 0, 0.22);
  border: 1px solid var(--border-color, #313244);
  border-radius: 8px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.drill-meta-header {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.source-badge,
.theme-badge,
.tier-badge {
  font-size: 0.72rem;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
}

.badge-game {
  background: rgba(243, 139, 168, 0.18);
  color: #f38ba8;
}

.badge-reinforce {
  background: rgba(137, 180, 250, 0.18);
  color: #89b4fa;
}

.theme-badge {
  background: rgba(166, 227, 161, 0.16);
  color: #a6e3a1;
}

.tier-badge {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-secondary, #a6adc8);
}

.drill-headline {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 700;
}

.coaching-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.section-label {
  font-size: 0.73rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--accent-color, #89b4fa);
  font-weight: 700;
}

.section-body {
  margin: 0;
  font-size: 0.82rem;
  line-height: 1.45;
}

.protocol-box {
  background: rgba(255, 255, 255, 0.04);
  padding: 8px 10px;
  border-radius: 6px;
  border-left: 3px solid var(--accent-color, #89b4fa);
}

.takeaway-section {
  background: rgba(166, 227, 161, 0.08);
  padding: 8px 10px;
  border-radius: 6px;
  border-left: 3px solid #a6e3a1;
}

.cues-list {
  margin: 0;
  padding-left: 18px;
  font-size: 0.8rem;
  line-height: 1.4;
}

.queue-card {
  max-height: 195px;
  overflow-y: auto;
}

.queue-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.queue-item {
  display: grid;
  grid-template-columns: 36px 140px 1fr auto;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color, #313244);
  font-size: 0.78rem;
  cursor: pointer;
}

.queue-item.active {
  border-color: var(--accent-color, #89b4fa);
  background: rgba(137, 180, 250, 0.1);
}

.queue-item.solved {
  border-left: 3px solid #a6e3a1;
}

.queue-item.missed {
  border-left: 3px solid #f38ba8;
}

.progress-body {
  padding: 16px 18px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.weakness-table-wrap {
  overflow-x: auto;
}

.weakness-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.82rem;
}

.weakness-table th,
.weakness-table td {
  padding: 8px 10px;
  text-align: left;
  border-bottom: 1px solid var(--border-color, #313244);
}

.habit-sub {
  font-size: 0.73rem;
  color: var(--text-secondary, #a6adc8);
  margin-top: 2px;
}

.mini-mastery-bar {
  width: 90px;
  height: 6px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
  overflow: hidden;
  display: inline-block;
  vertical-align: middle;
  margin-right: 6px;
}

.mini-mastery-fill {
  height: 100%;
  background: #a6e3a1;
}

.weight-badge {
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  font-weight: 700;
}

.weight-badge.high {
  background: rgba(243, 139, 168, 0.2);
  color: #f38ba8;
}

.history-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.history-card {
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--border-color, #313244);
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 0.8rem;
}

.history-top {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.history-detail {
  color: var(--text-secondary, #a6adc8);
  font-size: 0.75rem;
}

.form-input {
  padding: 6px 9px;
  border-radius: 6px;
  background: var(--bg-primary, #11111b);
  color: var(--text-primary, #cdd6f4);
  border: 1px solid var(--border-color, #45475a);
  font-size: 0.8rem;
  width: 100%;
  box-sizing: border-box;
}

.btn-primary {
  background: var(--accent-color, #89b4fa);
  color: #11111b;
  border: none;
  padding: 7px 12px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  font-size: 0.8rem;
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

.btn-xs {
  padding: 4px 8px;
  font-size: 0.74rem;
}

.text-danger {
  color: #f38ba8;
  font-weight: 700;
}

.empty-hint,
.empty-coach-state {
  color: var(--text-secondary, #a6adc8);
  font-size: 0.84rem;
}
</style>
