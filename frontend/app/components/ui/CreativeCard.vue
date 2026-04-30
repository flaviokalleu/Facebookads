<script setup lang="ts">
import { Lightbulb, Eye, MousePointerClick } from 'lucide-vue-next'

defineProps<{
  creative: {
    id: string
    campaign_name: string
    headline?: string
    fatigue_score: number
    ctr: number
    impressions: number
    recommendation?: string
  }
}>()

function fatigueColor(score: number) {
  if (score >= 75) return 'text-red-400'
  if (score >= 50) return 'text-amber-400'
  return 'text-emerald-400'
}

function fatigueLabel(score: number) {
  if (score >= 75) return 'High Fatigue'
  if (score >= 50) return 'Moderate'
  return 'Healthy'
}
</script>

<template>
  <div class="card border-bg-border/50 shadow-lg shadow-black/10 hover:border-blue-muted/50 transition-all duration-200">
    <div class="flex items-start justify-between gap-2 mb-3">
      <div class="min-w-0">
        <p class="text-primary text-xs font-semibold truncate">{{ creative.headline ?? 'Untitled Creative' }}</p>
        <p class="text-muted text-xs truncate">{{ creative.campaign_name }}</p>
      </div>
      <span class="text-xs font-bold shrink-0 px-2 py-0.5 rounded-full border" :class="`${fatigueColor(creative.fatigue_score)} bg-${creative.fatigue_score >= 75 ? 'red' : creative.fatigue_score >= 50 ? 'amber' : 'emerald'}-500/10 border-${creative.fatigue_score >= 75 ? 'red' : creative.fatigue_score >= 50 ? 'amber' : 'emerald'}-500/20`">
        {{ fatigueLabel(creative.fatigue_score) }}
      </span>
    </div>

    <!-- Fatigue bar -->
    <div class="mb-3">
      <div class="flex justify-between text-xs text-muted mb-1">
        <span>Fatigue</span>
        <span :class="fatigueColor(creative.fatigue_score)">{{ creative.fatigue_score }}%</span>
      </div>
      <div class="h-1.5 bg-bg-elevated rounded-full overflow-hidden">
        <div
          class="h-full rounded-full transition-all duration-500"
          :class="creative.fatigue_score >= 75 ? 'bg-red-500' : creative.fatigue_score >= 50 ? 'bg-amber-500' : 'bg-emerald-500'"
          :style="{ width: `${creative.fatigue_score}%` }"
        />
      </div>
    </div>

    <div class="flex gap-4 text-xs mb-3">
      <div class="flex items-center gap-1">
        <MousePointerClick class="w-3 h-3 text-muted" />
        <span class="text-muted">CTR </span>
        <span class="text-primary font-mono font-medium">{{ creative.ctr.toFixed(2) }}%</span>
      </div>
      <div class="flex items-center gap-1">
        <Eye class="w-3 h-3 text-muted" />
        <span class="text-muted">Impr. </span>
        <span class="text-primary font-mono font-medium">
          {{ creative.impressions >= 1000 ? `${(creative.impressions / 1000).toFixed(1)}K` : creative.impressions }}
        </span>
      </div>
    </div>

    <p v-if="creative.recommendation" class="text-muted text-xs border-t border-bg-border/50 pt-2 mt-2 flex items-start gap-1.5">
      <Lightbulb class="w-3 h-3 text-amber-400 mt-0.5 shrink-0" />
      {{ creative.recommendation }}
    </p>
  </div>
</template>
