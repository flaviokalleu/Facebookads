<script setup lang="ts">
interface Props {
  values: number[]
  height?: number
  color?: string
  fill?: boolean
}
const props = withDefaults(defineProps<Props>(), {
  height: 56,
  color: '#1877F2',
  fill: true,
})

const points = computed(() => {
  const vs = props.values
  if (!vs.length) return ''
  const max = Math.max(...vs, 1)
  const min = Math.min(...vs, 0)
  const range = Math.max(max - min, 1)
  const w = 100
  const h = props.height
  const stepX = vs.length > 1 ? w / (vs.length - 1) : 0
  return vs
    .map((v, i) => {
      const x = i * stepX
      const y = h - ((v - min) / range) * h
      return `${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')
})

const fillPoints = computed(() => {
  if (!props.values.length) return ''
  const w = 100
  const h = props.height
  return `0,${h} ${points.value} ${w},${h}`
})
</script>

<template>
  <svg
    class="block w-full"
    :height="height"
    :viewBox="`0 0 100 ${height}`"
    preserveAspectRatio="none"
  >
    <polygon v-if="fill && values.length" :points="fillPoints" :fill="color" fill-opacity="0.12" />
    <polyline
      v-if="values.length"
      :points="points"
      :stroke="color"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
      fill="none"
      vector-effect="non-scaling-stroke"
    />
    <text
      v-if="!values.length"
      x="50"
      :y="height / 2"
      text-anchor="middle"
      class="fill-ink-faint text-[6px]"
    >
      sem dados
    </text>
  </svg>
</template>
