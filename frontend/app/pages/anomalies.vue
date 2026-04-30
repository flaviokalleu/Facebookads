<script setup lang="ts">
import { useAnomalyStore } from '~/stores/useAnomalyStore'
import { AlertTriangle, Filter } from 'lucide-vue-next'

const store = useAnomalyStore()
onMounted(() => store.fetchAll())

const SEVERITY_OPTIONS = [
  { label: 'All',    value: 'all' },
  { label: 'High',   value: 'HIGH' },
  { label: 'Medium', value: 'MEDIUM' },
  { label: 'Low',    value: 'LOW' },
]
</script>

<template>
  <div class="space-y-5">
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div class="flex items-center gap-2">
        <AlertTriangle class="w-5 h-5 text-red-400" />
        <div>
          <h1 class="text-primary text-xl font-bold">Anomalies</h1>
          <p class="text-muted text-sm">{{ store.anomalies.length }} detected</p>
        </div>
      </div>
      <div class="flex gap-2">
        <span v-if="store.highCount > 0" class="text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded-full px-3 py-1.5 flex items-center gap-1">
          <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse" />
          {{ store.highCount }} critical
        </span>
      </div>
    </div>

    <!-- Severity filter -->
    <div class="flex gap-2 flex-wrap">
      <Filter class="w-4 h-4 text-muted self-center" />
      <button
        v-for="opt in SEVERITY_OPTIONS"
        :key="opt.value"
        class="text-xs px-3 py-1.5 rounded-full border transition-all"
        :class="store.severityFilter === opt.value
          ? 'bg-blue-default/20 text-blue-bright border-blue-default/40'
          : 'text-muted border-bg-border/50 hover:text-secondary hover:border-bg-border'"
        @click="store.severityFilter = opt.value"
      >
        {{ opt.label }}
      </button>
    </div>

    <template v-if="store.loading">
      <SkeletonCard v-for="i in 4" :key="i" />
    </template>
    <div v-else-if="store.filtered.length === 0" class="flex flex-col items-center justify-center py-16 text-center">
      <div class="w-16 h-16 rounded-2xl bg-emerald-500/10 flex items-center justify-center mb-4">
        <svg class="w-8 h-8 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
      </div>
      <p class="text-primary font-medium mb-1">No anomalies</p>
      <p class="text-muted text-sm max-w-xs">Everything looks normal for the selected filter.</p>
    </div>
    <div v-else class="space-y-2">
      <AnomalyCard v-for="a in store.filtered" :key="a.id" :anomaly="a" />
    </div>
  </div>
</template>
