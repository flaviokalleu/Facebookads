<script setup lang="ts">
import { useBudgetStore } from '~/stores/useBudgetStore'
import { Wallet, TrendingUp, TrendingDown, Check, Cpu } from 'lucide-vue-next'
import { api } from '~/lib/api'

const store = useBudgetStore()
onMounted(() => store.fetchAll())

function fmtCurrency(n: number) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(n)
}

const applyingId = ref<string | null>(null)

async function applySuggestion(id: string) {
  applyingId.value = id
  try {
    await api.campaigns.applyBudgetSuggestion(id)
    await store.fetchAll()
  } finally {
    applyingId.value = null
  }
}
</script>

<template>
  <div class="space-y-5">
    <div class="flex items-center gap-2">
      <Wallet class="w-5 h-5 text-emerald-400" />
      <div>
        <h1 class="text-primary text-xl font-bold">Budget Advisor</h1>
        <p class="text-muted text-sm">AI-powered budget reallocation suggestions</p>
      </div>
    </div>

    <!-- Summary KPIs -->
    <div class="grid grid-cols-2 gap-3" v-if="!store.loading">
      <div class="card border-bg-border/50 shadow-lg shadow-black/10">
        <div class="flex items-center gap-1.5 text-muted mb-1">
          <TrendingUp class="w-3.5 h-3.5 text-emerald-400" />
          <p class="text-xs">Suggested Increases</p>
        </div>
        <p class="text-emerald-400 text-xl font-bold font-mono">+{{ fmtCurrency(store.totalIncrease) }}</p>
      </div>
      <div class="card border-bg-border/50 shadow-lg shadow-black/10">
        <div class="flex items-center gap-1.5 text-muted mb-1">
          <TrendingDown class="w-3.5 h-3.5 text-red-400" />
          <p class="text-xs">Suggested Decreases</p>
        </div>
        <p class="text-red-400 text-xl font-bold font-mono">-{{ fmtCurrency(store.totalDecrease) }}</p>
      </div>
    </div>

    <template v-if="store.loading">
      <SkeletonCard v-for="i in 4" :key="i" />
    </template>
    <div v-else-if="store.suggestions.length === 0" class="flex flex-col items-center justify-center py-16 text-center">
      <div class="w-16 h-16 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-4">
        <Wallet class="w-8 h-8 text-muted" />
      </div>
      <p class="text-primary font-medium mb-1">No budget suggestions</p>
      <p class="text-muted text-sm max-w-xs">Your budgets look optimal. Check back after running AI analysis.</p>
    </div>
    <div v-else class="space-y-3">
      <div
        v-for="s in store.suggestions"
        :key="s.id"
        class="card border-bg-border/50 shadow-lg shadow-black/10 hover:border-blue-muted/50 transition-all"
        :class="{ 'opacity-50': s.is_applied }"
      >
        <div class="flex items-start justify-between gap-3 mb-2">
          <div class="min-w-0">
            <p class="text-primary text-sm font-semibold truncate">{{ s.campaign_name }}</p>
            <p class="text-muted text-xs mt-0.5">
              Current: {{ fmtCurrency(s.current_budget) }} →
              Suggested: {{ fmtCurrency(s.suggested_budget) }}
            </p>
          </div>
          <div class="flex items-center gap-3 shrink-0">
            <div class="text-right">
              <span
                class="text-sm font-bold"
                :class="(s.suggested_change ?? 0) > 0 ? 'text-emerald-400' : 'text-red-400'"
              >
                {{ (s.suggested_change ?? 0) > 0 ? '+' : '' }}{{ (s.suggested_change ?? 0).toFixed(1) }}%
              </span>
              <p class="text-muted text-xs">change</p>
            </div>
            <button
              v-if="!s.is_applied"
              :disabled="applyingId === s.id"
              class="flex items-center gap-1 text-xs font-medium text-emerald-400 hover:text-emerald-300 bg-emerald-500/10 hover:bg-emerald-500/20 px-2.5 py-1.5 rounded-lg border border-emerald-500/20 transition-all disabled:opacity-50"
              @click="applySuggestion(s.id)"
            >
              <Check class="w-3 h-3" />
              {{ applyingId === s.id ? 'Applying...' : 'Apply' }}
            </button>
            <span v-else class="text-xs text-emerald-400 font-medium flex items-center gap-1">
              <Check class="w-3 h-3" /> Applied
            </span>
          </div>
        </div>

        <p class="text-secondary text-xs leading-relaxed">{{ s.change_reason || s.rationale }}</p>

        <div class="mt-2 flex items-center justify-between">
          <span class="text-xs text-muted">
            Expected impact: <span class="text-emerald-400">{{ s.expected_roas_improvement || s.expected_impact }}</span>
          </span>
          <span class="text-xs text-muted font-mono bg-bg-elevated/50 px-2 py-0.5 rounded-full border border-bg-border/30 flex items-center gap-1">
            <Cpu class="w-3 h-3" />
            {{ s.model_used }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
