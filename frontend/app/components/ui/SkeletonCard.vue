<script setup lang="ts">
defineProps<{ lines?: number }>()
</script>

<template>
  <div class="card space-y-3">
    <div class="skeleton h-4 w-1/3" />
    <div class="skeleton h-7 w-1/2" />
    <template v-for="i in (lines ?? 2)" :key="i">
      <div class="skeleton h-3 w-full" />
    </template>
  </div>
</template>
