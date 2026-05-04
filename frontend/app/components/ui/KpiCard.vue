<script setup lang="ts">
import { Info } from 'lucide-vue-next'
import UiTooltip from './UiTooltip.vue'

defineProps<{
  label: string
  value: string
  deltaPct?: number
  hint?: string
}>()
</script>

<template>
  <div class="rounded-xl border border-border bg-bg p-6 transition hover:shadow-sm">
    <div class="flex items-center justify-between">
      <p class="text-sm text-ink-muted">{{ label }}</p>
      <UiTooltip v-if="hint" :text="hint">
        <Info class="h-4 w-4 text-ink-faint" />
      </UiTooltip>
    </div>
    <p class="mt-2 text-3xl font-semibold tracking-tight text-ink">{{ value }}</p>
    <p
      v-if="deltaPct !== undefined"
      :class="['mt-1 text-sm', deltaPct >= 0 ? 'text-success' : 'text-danger']"
    >
      {{ deltaPct >= 0 ? '+' : '' }}{{ deltaPct }}% vs ontem
    </p>
  </div>
</template>
