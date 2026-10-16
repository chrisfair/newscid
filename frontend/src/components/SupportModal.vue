<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Heart, X, ExternalLink, CheckCircle, ShieldCheck, Sparkles } from '@lucide/vue'
import {
  GetSupportStatus,
  RecordUserSupport,
  DismissSupportNag,
  OpenSupportLink,
} from '../../wailsjs/go/main/App'

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'updated', supported: boolean): void
}>()

interface SupportState {
  supported: boolean
  isAuthor: boolean
  authorId?: string
  deviceSignature?: string
  supporterTier?: string
  supportedAt?: number
  dismissCount: number
  lastDismissedAt?: number
  stripeUrl: string
  minAmountUsd: number
  shouldShowNag: boolean
}

const status = ref<SupportState>({
  supported: false,
  isAuthor: false,
  authorId: '',
  deviceSignature: '',
  supporterTier: '',
  supportedAt: 0,
  dismissCount: 0,
  lastDismissedAt: 0,
  stripeUrl: 'https://buy.stripe.com/cNidR2e262cEeiOdpCefC01',
  minAmountUsd: 5,
  shouldShowNag: true,
})

const selectedAmount = ref<number>(5)
const customAmount = ref<number>(5)
const useCustomAmount = ref<boolean>(false)
const referenceInput = ref<string>('')
const stripeOpened = ref<boolean>(false)
const confirmMessage = ref<string>('')

const presetAmounts = [
  { amount: 5, label: '$5', desc: 'Knight' },
  { amount: 10, label: '$10', desc: 'Rook' },
  { amount: 35, label: '$35', desc: 'Queen' },
  { amount: 100, label: '$100', desc: 'Princess' },
  { amount: 350, label: '$350', desc: 'King' },
]

const effectiveAmount = computed(() => {
  if (useCustomAmount.value) {
    return Math.max(status.value.minAmountUsd || 5, Number(customAmount.value) || 5)
  }
  return Math.max(status.value.minAmountUsd || 5, selectedAmount.value)
})

function selectPreset(amt: number) {
  useCustomAmount.value = false
  selectedAmount.value = amt
}

function enableCustom() {
  useCustomAmount.value = true
  if (customAmount.value < 5) {
    customAmount.value = 5
  }
}

onMounted(async () => {
  try {
    const res = await GetSupportStatus()
    if (res) {
      status.value = res
    }
  } catch (err) {
    console.error('Failed to load support status:', err)
  }
})

async function handleOpenStripe() {
  stripeOpened.value = true
  try {
    const targetUrl = await OpenSupportLink(effectiveAmount.value)
    const refreshed = await GetSupportStatus()
    if (refreshed) {
      status.value = refreshed
    } else {
      status.value.supported = true
      status.value.shouldShowNag = false
    }
    confirmMessage.value = 'Thank you for supporting NewScid! The startup prompt is now permanently disabled.'
    emit('updated', true)
    return targetUrl
  } catch {
    if (typeof window !== 'undefined' && status.value.stripeUrl) {
      window.open(status.value.stripeUrl, '_blank', 'noopener,noreferrer')
    }
  }
}

async function handleConfirmSupport() {
  try {
    const tierLabel = `$${effectiveAmount.value} Support`
    const updated = await RecordUserSupport(tierLabel, referenceInput.value)
    if (updated) {
      status.value = updated
    } else {
      status.value.supported = true
      status.value.shouldShowNag = false
    }
    confirmMessage.value = 'Thank you for supporting NewScid! The startup prompt is now permanently disabled.'
    emit('updated', true)
  } catch (err) {
    console.error('Failed to record support:', err)
  }
}

async function handleDismiss() {
  try {
    if (!status.value.supported && !status.value.isAuthor) {
      await DismissSupportNag()
    }
  } catch (err) {
    console.error('Failed to dismiss support prompt:', err)
  } finally {
    emit('close')
  }
}
</script>

<template>
  <div
    class="fixed inset-0 bg-black/85 backdrop-blur-sm flex items-center justify-center z-70 p-4 animate-fade-in select-none"
    @click.self="handleDismiss"
  >
    <div class="bg-zinc-900 border border-zinc-700 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2.5">
          <div class="w-7 h-7 rounded-lg bg-rose-500/15 border border-rose-500/30 flex items-center justify-center">
            <Heart class="w-4 h-4 text-rose-400 fill-rose-400/30" />
          </div>
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Support NewScid</h2>
            <p class="text-[11px] text-zinc-400">Voluntary Author Support &bull; Non-Obligatory</p>
          </div>
        </div>
        <button
          data-testid="support-close-btn"
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors cursor-pointer"
          title="Dismiss"
          @click="handleDismiss"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Body -->
      <div class="p-5 space-y-4 text-xs text-zinc-300">
        <!-- Author or Active Supporter Banner -->
        <div
          v-if="status.isAuthor"
          data-testid="author-verified-banner"
          class="p-3.5 rounded-xl bg-emerald-950/70 border border-emerald-500/40 flex items-start gap-3 text-emerald-200"
        >
          <ShieldCheck class="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
          <div class="space-y-1">
            <div class="font-bold text-emerald-300">Author Machine Verified — Startup Prompt Disabled</div>
            <p class="text-[11px] text-emerald-200/80">
              Authenticated via linked Chess.com &amp; Lichess.org IDs
              <span class="font-mono text-emerald-300">({{ status.authorId }})</span>.
            </p>
          </div>
        </div>

        <div
          v-else-if="status.supported"
          data-testid="supporter-active-banner"
          class="p-3.5 rounded-xl bg-emerald-950/70 border border-emerald-500/40 flex items-start gap-3 text-emerald-200"
        >
          <CheckCircle class="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
          <div class="space-y-1">
            <div class="font-bold text-emerald-300">Thank You for Supporting NewScid!</div>
            <p class="text-[11px] text-emerald-200/80">
              Supporter status active ({{ status.supporterTier || '$5+ Support' }}). The startup reminder is permanently disabled on this computer.
            </p>
          </div>
        </div>

        <!-- Pitch / Explanation -->
        <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5 space-y-2 leading-relaxed">
          <p class="text-zinc-200 font-medium">
            Enjoying NewScid? Consider supporting the author with a voluntary contribution of <span class="text-emerald-400 font-bold">$5 or higher</span>.
          </p>
          <p class="text-zinc-400 text-[11px]">
            Your support directly funds continued development, database tools, and training features. Supporting is completely optional—you can dismiss this window at any time and use all features without restriction. Supporters never see the startup reminder again.
          </p>
        </div>

        <!-- Amount Selector ($5 or higher) -->
        <div class="space-y-2">
          <label class="block text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
            Choose Support Amount ($5 Minimum)
          </label>
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="preset in presetAmounts"
              :key="preset.amount"
              type="button"
              :data-testid="`tier-${preset.amount}`"
              class="py-2.5 px-2 rounded-xl border text-center transition-all cursor-pointer flex flex-col items-center justify-center gap-0.5"
              :class="!useCustomAmount && selectedAmount === preset.amount
                ? 'bg-emerald-600/20 border-emerald-500 text-white shadow-sm'
                : 'bg-zinc-950 border-zinc-800 text-zinc-400 hover:border-zinc-700 hover:text-zinc-200'"
              @click="selectPreset(preset.amount)"
            >
              <span class="text-sm font-bold text-zinc-100">{{ preset.label }}</span>
              <span class="text-[10px] text-zinc-400">{{ preset.desc }}</span>
            </button>

            <button
              type="button"
              data-testid="tier-custom"
              class="py-2.5 px-2 rounded-xl border text-center transition-all cursor-pointer flex flex-col items-center justify-center gap-0.5"
              :class="useCustomAmount
                ? 'bg-emerald-600/20 border-emerald-500 text-white shadow-sm'
                : 'bg-zinc-950 border-zinc-800 text-zinc-400 hover:border-zinc-700 hover:text-zinc-200'"
              @click="enableCustom"
            >
              <span class="text-sm font-bold text-zinc-100">Custom</span>
              <span class="text-[10px] text-zinc-400">$5 or more</span>
            </button>
          </div>

          <div v-if="useCustomAmount" class="pt-1 flex items-center gap-2">
            <span class="text-xs font-bold text-zinc-400">USD $</span>
            <input
              v-model.number="customAmount"
              data-testid="custom-amount-input"
              type="number"
              min="5"
              step="1"
              class="w-32 bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
            />
            <span class="text-[11px] text-zinc-500">(Minimum $5)</span>
          </div>
        </div>

        <!-- Primary Stripe Checkout Link Button -->
        <div class="pt-1">
          <button
            type="button"
            data-testid="stripe-support-btn"
            class="w-full py-2.5 px-4 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs flex items-center justify-center gap-2 shadow-lg transition-colors cursor-pointer"
            @click="handleOpenStripe"
          >
            <Heart class="w-4 h-4 fill-current" />
            <span>Support via Stripe (${{ effectiveAmount }}+)</span>
            <ExternalLink class="w-3.5 h-3.5 opacity-80" />
          </button>
        </div>

        <!-- Supporter / Author Verification to disable Nag Screen -->
        <div
          v-if="!status.supported"
          class="bg-zinc-950/90 border border-zinc-800 rounded-xl p-3.5 space-y-2.5"
        >
          <div class="flex items-center justify-between">
            <span class="text-[11px] font-semibold text-zinc-300 flex items-center gap-1.5">
              <Sparkles class="w-3.5 h-3.5 text-amber-400" />
              <span>Already Donated? Disable Startup Reminder</span>
            </span>
          </div>
          <div class="flex gap-2">
            <input
              v-model="referenceInput"
              data-testid="support-reference-input"
              type="text"
              placeholder="Optional: Stripe receipt email or note (no account needed)"
              class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
            />
            <button
              type="button"
              data-testid="confirm-support-btn"
              class="px-3.5 py-1.5 rounded-lg bg-zinc-800 hover:bg-emerald-700 text-zinc-200 hover:text-white border border-zinc-700 text-xs font-semibold transition-colors cursor-pointer whitespace-nowrap"
              @click="handleConfirmSupport"
            >
              I've Supported
            </button>
          </div>
        </div>

        <p v-if="confirmMessage" data-testid="confirm-support-msg" class="text-emerald-400 text-xs font-semibold text-center">
          {{ confirmMessage }}
        </p>

        <!-- Footer / Non-obligatory Dismiss -->
        <div class="flex items-center justify-between pt-2 border-t border-zinc-800/80">
          <span class="text-[11px] text-zinc-500">
            100% non-obligatory &bull; All features unlocked
          </span>
          <button
            type="button"
            data-testid="dismiss-support-btn"
            class="px-4 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-white text-xs font-medium transition-colors cursor-pointer"
            @click="handleDismiss"
          >
            {{ status.supported ? 'Close' : 'Maybe Later' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
