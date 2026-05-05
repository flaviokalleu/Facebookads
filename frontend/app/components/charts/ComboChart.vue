<script setup lang="ts">
interface Props {
  bars: number[]
  line: number[]
  labels: string[]
  height?: number
  barColor?: string
  lineColor?: string
  barLabel?: string
  lineLabel?: string
  formatBar?: (v: number) => string
  formatLine?: (v: number) => string
}
const props = withDefaults(defineProps<Props>(), {
  height: 220,
  barColor: '#1877F2',
  lineColor: '#42B72A',
  barLabel: 'Investimento',
  lineLabel: 'Contatos',
  formatBar: (v) => `R$ ${v.toFixed(2)}`,
  formatLine: (v) => String(Math.round(v)),
})

const hover = ref<number | null>(null)

const maxBar = computed(() => Math.max(...props.bars, 1))
const maxLine = computed(() => Math.max(...props.line, 1))

const linePoints = computed(() => {
  const n = props.line.length
  if (!n) return ''
  const w = 100
  const stepX = n > 1 ? w / (n - 1) : 0
  return props.line
    .map((v, i) => {
      const x = i * stepX
      const y = 100 - (v / maxLine.value) * 100
      return `${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')
})

const lineDots = computed(() =>
  props.line.map((v, i) => {
    const n = props.line.length
    const stepX = n > 1 ? 100 / (n - 1) : 0
    return {
      cx: i * stepX,
      cy: 100 - (v / maxLine.value) * 100,
    }
  }),
)
</script>

<template>
  <div>
    <div class="mb-2 flex flex-wrap items-center gap-4 text-xs">
      <div class="flex items-center gap-1.5">
        <span class="inline-block h-3 w-3 rounded-sm" :style="{ backgroundColor: barColor }" />
        <span class="text-ink-muted">{{ barLabel }}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="inline-block h-0.5 w-3" :style="{ backgroundColor: lineColor }" />
        <span class="text-ink-muted">{{ lineLabel }}</span>
      </div>
    </div>

    <div class="relative" :style="{ height: `${height}px` }" @mouseleave="hover = null">
      <!-- Bars -->
      <div class="absolute inset-0 flex items-end gap-1">
        <div
          v-for="(v, i) in bars"
          :key="i"
          class="group relative flex-1 cursor-default rounded-t-sm transition"
          :class="[hover === i ? 'opacity-100' : 'opacity-90']"
          :style="{
            height: `${Math.max((v / maxBar) * 100, 1)}%`,
            backgroundColor: barColor,
          }"
          @mouseenter="hover = i"
        />
      </div>

      <!-- Line overlay (SVG) -->
      <svg
        class="pointer-events-none absolute inset-0 h-full w-full"
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
      >
        <polyline
          v-if="line.length"
          :points="linePoints"
          :stroke="lineColor"
          stroke-width="0.6"
          stroke-linecap="round"
          stroke-linejoin="round"
          fill="none"
          vector-effect="non-scaling-stroke"
        />
      </svg>
      <svg
        class="pointer-events-none absolute inset-0 h-full w-full overflow-visible"
        :viewBox="`0 0 100 100`"
        preserveAspectRatio="none"
      >
        <circle
          v-for="(p, i) in lineDots"
          :key="i"
          :cx="p.cx"
          :cy="p.cy"
          r="0.8"
          :fill="lineColor"
          vector-effect="non-scaling-stroke"
        />
      </svg>

      <!-- Tooltip -->
      <div
        v-if="hover !== null && hover < bars.length"
        class="pointer-events-none absolute z-10 rounded-lg bg-ink px-3 py-2 text-xs text-white shadow-lg"
        :style="{
          left: `${(hover / Math.max(bars.length - 1, 1)) * 100}%`,
          bottom: '100%',
          transform: 'translateX(-50%)',
          marginBottom: '4px',
        }"
      >
        <p class="font-medium">{{ labels[hover] }}</p>
        <p class="mt-0.5">
          <span :style="{ color: barColor }">●</span> {{ formatBar(bars[hover]) }}
        </p>
        <p>
          <span :style="{ color: lineColor }">●</span> {{ formatLine(line[hover]) }} {{ lineLabel.toLowerCase() }}
        </p>
      </div>
    </div>

    <div class="mt-2 flex w-full text-[10px] text-ink-faint">
      <span v-for="(l, i) in labels" :key="i" class="flex-1 text-center">
        <template v-if="i === 0 || i === labels.length - 1 || i === Math.floor(labels.length / 2)">
          {{ l }}
        </template>
      </span>
    </div>
  </div>
</template>
