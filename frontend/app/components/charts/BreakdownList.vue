<script setup lang="ts">
interface Item {
  label: string
  value: number
  spend?: number
  cpl?: number
  meta?: string
}
interface Props {
  items: Item[]
  emptyText?: string
  showCPL?: boolean
}
const props = withDefaults(defineProps<Props>(), {
  emptyText: 'Sem dados.',
  showCPL: true,
})

const max = computed(() => Math.max(...props.items.map((x) => x.value), 1))
const total = computed(() => props.items.reduce((s, x) => s + x.value, 0))

const brl = (v: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v || 0)
</script>

<template>
  <div>
    <p v-if="!items.length" class="text-sm text-ink-muted">{{ emptyText }}</p>
    <ul v-else class="space-y-2.5">
      <li v-for="(it, i) in items" :key="i" class="text-xs">
        <div class="flex items-center justify-between gap-2">
          <span class="truncate font-medium text-ink">{{ it.label }}</span>
          <span class="shrink-0 tabular-nums text-ink-muted">
            <span class="text-ink">{{ it.value }}</span>
            <span v-if="total > 0" class="ml-1 text-ink-faint">
              ({{ Math.round((it.value / total) * 100) }}%)
            </span>
          </span>
        </div>
        <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-bg-muted">
          <div
            class="h-full rounded-full bg-accent transition-[width]"
            :style="{ width: `${(it.value / max) * 100}%` }"
          />
        </div>
        <div v-if="showCPL && it.cpl !== undefined && it.cpl > 0" class="mt-0.5 flex justify-between text-[10px] text-ink-faint">
          <span v-if="it.spend !== undefined">{{ brl(it.spend) }} investido</span>
          <span>{{ brl(it.cpl) }} por contato</span>
        </div>
      </li>
    </ul>
  </div>
</template>
