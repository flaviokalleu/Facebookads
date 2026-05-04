<script setup lang="ts">
interface Props {
  values: number[]
  labels?: string[]
  height?: number
  color?: string
}
const props = withDefaults(defineProps<Props>(), {
  height: 140,
  color: '#1877F2',
})

const max = computed(() => Math.max(...props.values, 1))
</script>

<template>
  <div class="flex w-full items-end gap-1" :style="{ height: `${height}px` }">
    <div
      v-for="(v, i) in values"
      :key="i"
      class="flex-1 rounded-t-sm transition hover:opacity-80"
      :style="{
        height: `${Math.max((v / max) * 100, 2)}%`,
        backgroundColor: color,
      }"
      :title="labels?.[i] ? `${labels[i]}: ${v}` : String(v)"
    />
  </div>
  <div v-if="labels?.length" class="mt-2 flex w-full text-[10px] text-ink-faint">
    <span v-for="(l, i) in labels" :key="i" class="flex-1 text-center">
      <template v-if="i === 0 || i === labels.length - 1 || i === Math.floor(labels.length / 2)">
        {{ l }}
      </template>
    </span>
  </div>
</template>
