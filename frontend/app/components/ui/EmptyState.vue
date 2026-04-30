<script setup lang="ts">
defineProps<{
  icon?: any
  title: string
  description?: string
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center py-16 text-center">
    <div v-if="icon" class="w-14 h-14 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-4">
      <component :is="icon" class="w-7 h-7 text-muted" />
    </div>
    <p class="text-primary font-medium mb-1">{{ title }}</p>
    <p v-if="description" class="text-muted text-sm max-w-xs">{{ description }}</p>
  </div>
</template>
