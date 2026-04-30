<script setup lang="ts">
import type { Recommendation } from '~/lib/api'
import { api } from '~/lib/api'
import { Wallet, Crosshair, Palette, Zap, Users, Calendar, ArrowUp, Check, Cpu } from 'lucide-vue-next'

const props = defineProps<{ recommendation: Recommendation }>()

const applying = ref(false)
const applied = ref(props.recommendation.is_applied)

const priorityConfig = {
  HIGH:   { class: 'text-red-400 bg-red-500/10 border-red-500/20' },
  MEDIUM: { class: 'text-amber-400 bg-amber-500/10 border-amber-500/20' },
  LOW:    { class: 'text-blue-400 bg-blue-500/10 border-blue-500/20' },
}

const categoryIcons: Record<string, any> = {
  BUDGET: Wallet, TARGETING: Crosshair, CREATIVE: Palette,
  BIDDING: Zap, AUDIENCE: Users, SCHEDULE: Calendar,
}

async function apply() {
  applying.value = true
  try {
    await api.campaigns.applyRecommendation(props.recommendation.id)
    applied.value = true
  } finally {
    applying.value = false
  }
}
</script>

<template>
  <div class="card border-bg-border/50 shadow-lg shadow-black/10 hover:border-blue-muted/50 transition-all duration-200" :class="{ 'opacity-60': applied }">
    <div class="flex items-start justify-between gap-3 mb-3">
      <div class="flex items-center gap-2">
        <span
          class="text-xs font-bold px-2 py-0.5 rounded border shrink-0"
          :class="priorityConfig[recommendation.priority]?.class"
        >
          {{ recommendation.priority }}
        </span>
        <div class="flex items-center gap-1.5">
          <component :is="categoryIcons[recommendation.category]" class="w-3.5 h-3.5 text-muted" />
          <span class="text-secondary text-xs font-medium uppercase tracking-wide">
            {{ recommendation.category }}
          </span>
        </div>
      </div>

      <button
        v-if="!applied"
        :disabled="applying"
        class="flex items-center gap-1 text-xs font-medium text-emerald-400 hover:text-emerald-300 bg-emerald-500/10 hover:bg-emerald-500/20 px-2.5 py-1 rounded-lg border border-emerald-500/20 transition-all shrink-0 disabled:opacity-50"
        @click.prevent="apply"
      >
        <Check class="w-3 h-3" />
        {{ applying ? 'Applying...' : 'Apply' }}
      </button>
      <span v-else class="text-xs text-emerald-400 font-medium flex items-center gap-1">
        <Check class="w-3 h-3" /> Applied
      </span>
    </div>

    <p class="text-primary text-sm font-medium mb-1.5 leading-relaxed">
      {{ recommendation.action }}
    </p>

    <div class="flex items-center gap-1.5 mb-2">
      <ArrowUp class="w-3 h-3 text-emerald-400" />
      <span class="text-emerald-400 text-xs font-medium">{{ recommendation.expected_impact }}</span>
    </div>

    <p class="text-muted text-xs leading-relaxed border-t border-bg-border/50 pt-2 mt-2">
      {{ recommendation.rationale }}
    </p>

    <div class="mt-2 flex justify-end">
      <span class="text-muted text-xs bg-bg-elevated/50 px-2 py-0.5 rounded-full border border-bg-border/30 flex items-center gap-1">
        <Cpu class="w-3 h-3" />
        {{ recommendation.model_used }}
      </span>
    </div>
  </div>
</template>
