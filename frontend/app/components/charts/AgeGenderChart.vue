<script setup lang="ts">
interface Row {
  age: string
  gender: string
  value: number
}
interface Props {
  rows: Row[]
  emptyText?: string
}
const props = withDefaults(defineProps<Props>(), { emptyText: 'Sem dados.' })

// Pivot to: per age bucket, sum by gender
const ages = computed(() => {
  const order = ['13-17', '18-24', '25-34', '35-44', '45-54', '55-64', '65+']
  const map = new Map<string, { male: number; female: number; unknown: number }>()
  for (const r of props.rows) {
    const a = r.age || 'unknown'
    if (!map.has(a)) map.set(a, { male: 0, female: 0, unknown: 0 })
    const slot = map.get(a)!
    const g = (r.gender || '').toLowerCase()
    if (g === 'male')      slot.male += r.value
    else if (g === 'female') slot.female += r.value
    else                   slot.unknown += r.value
  }
  const entries = Array.from(map.entries())
  entries.sort((a, b) => {
    const ai = order.indexOf(a[0]); const bi = order.indexOf(b[0])
    if (ai === -1 && bi === -1) return a[0].localeCompare(b[0])
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })
  return entries
})

const maxTotal = computed(() => {
  let m = 1
  for (const [, slot] of ages.value) {
    const t = slot.male + slot.female + slot.unknown
    if (t > m) m = t
  }
  return m
})

function malePct(slot: { male: number; female: number; unknown: number }) {
  const t = slot.male + slot.female + slot.unknown
  return t === 0 ? 0 : (slot.male / t) * 100
}
function femalePct(slot: { male: number; female: number; unknown: number }) {
  const t = slot.male + slot.female + slot.unknown
  return t === 0 ? 0 : (slot.female / t) * 100
}
</script>

<template>
  <div>
    <p v-if="!ages.length" class="text-sm text-ink-muted">{{ emptyText }}</p>
    <div v-else class="space-y-2">
      <div v-for="([age, slot]) in ages" :key="age" class="text-xs">
        <div class="flex items-center justify-between">
          <span class="font-medium text-ink">{{ age }}</span>
          <span class="tabular-nums text-ink-muted">{{ slot.male + slot.female + slot.unknown }}</span>
        </div>
        <div
          class="mt-1 h-2 w-full overflow-hidden rounded-full bg-bg-muted"
          :title="`${slot.male} homens · ${slot.female} mulheres`"
        >
          <div
            class="h-full inline-block transition-[width]"
            :style="{
              width: `${((slot.male + slot.female + slot.unknown) / maxTotal) * 100}%`,
              background: `linear-gradient(to right, #1877F2 0%, #1877F2 ${malePct(slot)}%, #E91E63 ${malePct(slot)}%, #E91E63 ${malePct(slot) + femalePct(slot)}%, #8A8D91 ${malePct(slot) + femalePct(slot)}%, #8A8D91 100%)`,
            }"
          />
        </div>
      </div>
      <div class="mt-3 flex flex-wrap gap-3 text-[10px]">
        <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm bg-accent"></span>Homens</span>
        <span class="flex items-center gap-1"><span class="h-2 w-2 rounded-sm" style="background:#E91E63"></span>Mulheres</span>
        <span class="flex items-center gap-1 text-ink-faint"><span class="h-2 w-2 rounded-sm bg-ink-faint"></span>Não informado</span>
      </div>
    </div>
  </div>
</template>
