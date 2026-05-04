<script setup lang="ts">
import AppSidebar from '~/components/layout/AppSidebar.vue'
import AppTopbar from '~/components/layout/AppTopbar.vue'
</script>

<template>
  <div class="min-h-screen bg-bg-subtle">
    <AppSidebar />
    <div class="ml-60 flex min-h-screen flex-col">
      <AppTopbar />
      <main class="flex-1 p-8">
        <slot />
      </main>
    </div>
  </div>
</template>
