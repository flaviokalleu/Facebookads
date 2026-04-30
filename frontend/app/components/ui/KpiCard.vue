<script setup lang="ts">
import { TrendingUp, TrendingDown } from 'lucide-vue-next'

const props = defineProps<{
  title: string
  value: string
  delta?: number
  deltaLabel?: string
  icon?: any
  loading?: boolean
}>()

const isPositive = computed(() => (props.delta ?? 0) >= 0)
const deltaText = computed(() =>
  props.delta !== undefined
    ? `${isPositive.value ? '+' : ''}${props.delta.toFixed(1)}%`
    : null
)
</script>

<template>
  <div v-if="loading" class="card border-bg-border/50 shadow-lg shadow-black/10">
    <div class="skeleton h-4 w-24 rounded-lg mb-3" />
    <div class="skeleton h-8 w-32 rounded-lg mb-2" />
    <div class="skeleton h-3 w-20 rounded-lg" />
  </div>

  <div v-else class="card border-bg-border/50 shadow-lg shadow-black/10 group hover:border-blue-muted/50 transition-all duration-200 hover:shadow-xl hover:shadow-blue-default/5">
    <div class="flex items-start justify-between mb-2">
      <span class="text-secondary text-sm font-medium">{{ title }}</span>
      <component :is="icon" v-if="icon" class="w-5 h-5 text-blue-glow/60 group-hover:text-blue-glow transition-colors" />
    </div>

    <div class="font-mono text-primary text-2xl font-bold tracking-tight mb-1">
      {{ value }}
    </div>

    <div v-if="deltaText" class="flex items-center gap-1.5 text-xs">
      <component
        :is="isPositive ? TrendingUp : TrendingDown"
        class="w-3.5 h-3.5"
        :class="isPositive ? 'text-emerald-400' : 'text-red-400'"
      />
      <span :class="isPositive ? 'text-emerald-400' : 'text-red-400'" class="font-medium">
        {{ deltaText }}
      </span>
      <span v-if="deltaLabel" class="text-muted">{{ deltaLabel }}</span>
    </div>
  </div>
</template>
