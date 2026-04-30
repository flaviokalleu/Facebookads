<script setup lang="ts">
import type { Anomaly } from '~/lib/api'
import { AlertTriangle, AlertCircle, Info } from 'lucide-vue-next'

defineProps<{ anomaly: Anomaly }>()

const severityConfig = {
  HIGH:   { color: 'border-red-500',   icon: AlertTriangle, textClass: 'text-red-400', bgClass: 'bg-red-500/10' },
  MEDIUM: { color: 'border-amber-500', icon: AlertCircle,   textClass: 'text-amber-400', bgClass: 'bg-amber-500/10' },
  LOW:    { color: 'border-blue-500',  icon: Info,          textClass: 'text-blue-400', bgClass: 'bg-blue-500/10' },
}

const typeLabel: Record<string, string> = {
  CPC_SPIKE:           'CPC Spike',
  CTR_DROP:            'CTR Drop',
  CREATIVE_FATIGUE:    'Creative Fatigue',
  BUDGET_WASTE:        'Budget Waste',
  AUDIENCE_SATURATION: 'Audience Saturation',
  ROAS_COLLAPSE:       'ROAS Collapse',
  DELIVERY_STALL:      'Delivery Stall',
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const hours = Math.floor(diff / 3_600_000)
  if (hours < 1) return 'Just now'
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}
</script>

<template>
  <div
    class="bg-bg-surface rounded-card p-4 border-l-4 border border-bg-border/50 hover:border-bg-border transition-all"
    :class="severityConfig[anomaly.severity]?.color ?? 'border-l-blue-500'"
  >
    <div class="flex items-start justify-between gap-2">
      <div class="flex items-center gap-2">
        <component :is="severityConfig[anomaly.severity]?.icon" class="w-4 h-4" :class="severityConfig[anomaly.severity]?.textClass" />
        <span class="text-xs font-bold tracking-wide px-1.5 py-0.5 rounded" :class="[severityConfig[anomaly.severity]?.textClass, severityConfig[anomaly.severity]?.bgClass]">
          {{ anomaly.severity }}
        </span>
        <span class="text-primary text-sm font-semibold">
          {{ typeLabel[anomaly.type] ?? anomaly.type }}
        </span>
      </div>
      <span class="text-muted text-xs shrink-0">{{ timeAgo(anomaly.detected_at) }}</span>
    </div>

    <p class="text-secondary text-sm mt-2 leading-relaxed">{{ anomaly.description }}</p>
  </div>
</template>
