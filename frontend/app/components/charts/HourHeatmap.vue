<script setup lang="ts">
interface Props {
  // 24-length array, value per hour (0..23)
  values: number[]
  label?: string
}
const props = defineProps<Props>()

const max = computed(() => Math.max(...props.values, 1))

function intensity(v: number) {
  const r = v / max.value
  // Map 0..1 to opacity 0.06..1.0
  return 0.06 + r * 0.94
}

function hourLabel(h: number) {
  return `${String(h).padStart(2, '0')}h`
}

const peakHour = computed(() => {
  let idx = 0
  let best = -1
  for (let i = 0; i < props.values.length; i++) {
    if (props.values[i] > best) { best = props.values[i]; idx = i }
  }
  return { hour: idx, value: best }
})
</script>

<template>
  <div>
    <div class="grid grid-cols-12 gap-1 sm:grid-cols-24">
      <div
        v-for="(v, h) in values"
        :key="h"
        class="group relative aspect-square rounded-sm"
        :style="{ backgroundColor: '#1877F2', opacity: intensity(v) }"
        :title="`${hourLabel(h)}: ${v} ${label || ''}`"
      >
        <span
          class="pointer-events-none absolute -top-7 left-1/2 z-10 hidden -translate-x-1/2 whitespace-nowrap rounded bg-ink px-2 py-1 text-[10px] text-white group-hover:block"
        >
          {{ hourLabel(h) }} · {{ v }} {{ label || '' }}
        </span>
      </div>
    </div>
    <div class="mt-2 flex justify-between text-[10px] text-ink-faint">
      <span>00h</span>
      <span>06h</span>
      <span>12h</span>
      <span>18h</span>
      <span>23h</span>
    </div>
    <p v-if="peakHour.value > 0" class="mt-3 text-xs text-ink-muted">
      Pico às <strong class="text-ink">{{ hourLabel(peakHour.hour) }}</strong> com {{ peakHour.value }} {{ label || '' }}
    </p>
  </div>
</template>

<style scoped>
@media (min-width: 640px) {
  .sm\:grid-cols-24 {
    grid-template-columns: repeat(24, minmax(0, 1fr));
  }
}
</style>
