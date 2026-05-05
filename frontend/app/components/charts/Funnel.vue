<script setup lang="ts">
interface Step {
  label: string
  value: number
  hint?: string
}
interface Props { steps: Step[] }
const props = defineProps<Props>()

const max = computed(() => Math.max(...props.steps.map((s) => s.value), 1))
const num = (v: number) => new Intl.NumberFormat('pt-BR').format(v || 0)
</script>

<template>
  <div class="space-y-2">
    <div
      v-for="(s, i) in steps"
      :key="i"
      class="rounded-lg bg-bg-muted p-3"
    >
      <div class="flex items-baseline justify-between gap-2">
        <p class="text-xs font-medium text-ink">{{ s.label }}</p>
        <p class="text-xl font-semibold tabular-nums text-ink">{{ num(s.value) }}</p>
      </div>
      <div class="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-bg">
        <div
          class="h-full rounded-full bg-accent transition-[width]"
          :style="{ width: `${(s.value / max) * 100}%` }"
        />
      </div>
      <div class="mt-1 flex items-center justify-between text-[10px] text-ink-faint">
        <span>{{ s.hint || '&nbsp;' }}</span>
        <span v-if="i > 0 && steps[i - 1].value > 0" class="tabular-nums">
          {{ ((s.value / steps[i - 1].value) * 100).toFixed(1) }}% do passo anterior
        </span>
      </div>
    </div>
  </div>
</template>
