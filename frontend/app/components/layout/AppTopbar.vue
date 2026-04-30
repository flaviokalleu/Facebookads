<script setup lang="ts">
import { useAuthStore } from '~/stores/useAuthStore'
import { Menu, RefreshCw } from 'lucide-vue-next'

const props = defineProps<{
  title: string
  subtitle?: string
}>()

const emit = defineEmits<{
  'toggle-sidebar': []
}>()

const auth = useAuthStore()
</script>

<template>
  <header class="flex items-center justify-between py-4 px-4 md:px-6 bg-bg-base/80 backdrop-blur sticky top-0 z-40 border-b border-bg-border/50">
    <div class="flex items-center gap-3">
      <button
        class="lg:hidden p-2 rounded-lg hover:bg-bg-elevated text-secondary hover:text-primary transition-colors"
        @click="emit('toggle-sidebar')"
      >
        <Menu class="w-5 h-5" />
      </button>
      <div>
        <h1 class="text-primary text-lg font-bold">{{ title }}</h1>
        <p v-if="subtitle" class="text-muted text-xs">{{ subtitle }}</p>
      </div>
    </div>

    <div class="flex items-center gap-3">
      <button
        class="flex items-center gap-1.5 text-xs font-medium text-blue-bright hover:text-blue-glow px-3 py-1.5 rounded-xl bg-blue-default/10 hover:bg-blue-default/20 border border-blue-default/20 transition-all"
      >
        <RefreshCw class="w-3.5 h-3.5" />
        Sync
      </button>
      <div v-if="auth.user" class="flex items-center gap-2">
        <span class="w-7 h-7 rounded-full bg-gradient-to-br from-blue-default to-blue-glow flex items-center justify-center text-xs font-bold text-white shadow-sm">
          {{ auth.user.name.charAt(0).toUpperCase() }}
        </span>
      </div>
    </div>
  </header>
</template>
