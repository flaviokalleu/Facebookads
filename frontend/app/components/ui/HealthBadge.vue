<script setup lang="ts">
const props = defineProps<{
  status: 'SCALING' | 'HEALTHY' | 'AT_RISK' | 'UNDERPERFORMING'
}>()

const config = {
  SCALING:         { label: 'Scaling',         class: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30' },
  HEALTHY:         { label: 'Healthy',          class: 'bg-blue-500/15 text-blue-400 border-blue-500/30' },
  AT_RISK:         { label: 'At Risk',          class: 'bg-amber-500/15 text-amber-400 border-amber-500/30 animate-pulse-slow' },
  UNDERPERFORMING: { label: 'Underperforming',  class: 'bg-red-500/15 text-red-400 border-red-500/30' },
}

const current = computed(() => config[props.status] ?? config.HEALTHY)
</script>

<template>
  <span
    class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border"
    :class="current.class"
  >
    {{ current.label }}
  </span>
</template>
