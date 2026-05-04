<script setup lang="ts">
defineProps<{ step: number; total: number }>()
</script>

<template>
  <div class="mx-auto max-w-xl px-6 pt-8">
    <div class="mb-2 flex items-center justify-between text-xs text-ink-muted">
      <span>Passo {{ step }} de {{ total }}</span>
      <span>{{ Math.round((step / total) * 100) }}%</span>
    </div>
    <div class="h-1 overflow-hidden rounded-full bg-bg-muted">
      <div
        class="h-full rounded-full bg-accent transition-all"
        :style="{ width: `${(step / total) * 100}%` }"
      />
    </div>
  </div>
</template>
