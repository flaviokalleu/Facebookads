<script setup lang="ts">
import type { CampaignWithMetrics } from '~/lib/api'
import { HealthBadge } from '#components'
import { Eye, MousePointerClick, TrendingUp, DollarSign } from 'lucide-vue-next'

const props = defineProps<{ campaign: CampaignWithMetrics }>()

function fmt(n: number, prefix = '') {
  if (n >= 1_000_000) return `${prefix}${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000)     return `${prefix}${(n / 1_000).toFixed(1)}K`
  return `${prefix}${n.toFixed(0)}`
}

function fmtCurrency(n: number) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(n)
}
</script>

<template>
  <NuxtLink
    :to="`/campaigns/${campaign.id}`"
    class="card border-bg-border/50 shadow-lg shadow-black/10 hover:border-blue-muted/50 hover:shadow-xl hover:shadow-blue-default/5 transition-all duration-200 block group"
  >
    <div class="flex items-start justify-between gap-2 mb-3">
      <div class="min-w-0 flex-1">
        <p class="text-primary text-sm font-semibold truncate group-hover:text-blue-bright transition-colors">{{ campaign.name }}</p>
        <p class="text-muted text-xs mt-0.5">{{ campaign.objective }}</p>
      </div>
      <HealthBadge :status="campaign.health_status" class="shrink-0" />
    </div>

    <div class="grid grid-cols-3 gap-3">
      <div class="bg-bg-elevated/30 rounded-lg p-2">
        <div class="flex items-center gap-1 text-muted mb-0.5">
          <DollarSign class="w-3 h-3" />
          <span class="text-2xs">Spend</span>
        </div>
        <p class="text-primary text-sm font-bold font-mono">{{ fmtCurrency(campaign.spend_30d ?? 0) }}</p>
      </div>
      <div class="bg-bg-elevated/30 rounded-lg p-2">
        <div class="flex items-center gap-1 text-muted mb-0.5">
          <MousePointerClick class="w-3 h-3" />
          <span class="text-2xs">CTR</span>
        </div>
        <p class="text-primary text-sm font-bold font-mono">{{ ((campaign.avg_ctr_7d ?? 0) * 100).toFixed(2) }}%</p>
      </div>
      <div class="bg-bg-elevated/30 rounded-lg p-2">
        <div class="flex items-center gap-1 text-muted mb-0.5">
          <TrendingUp class="w-3 h-3" />
          <span class="text-2xs">ROAS</span>
        </div>
        <p class="text-primary text-sm font-bold font-mono">{{ (campaign.avg_roas_7d ?? 0).toFixed(2) }}x</p>
      </div>
    </div>
  </NuxtLink>
</template>
