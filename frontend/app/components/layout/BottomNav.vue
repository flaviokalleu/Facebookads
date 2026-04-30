<script setup lang="ts">
import { LayoutDashboard, Megaphone, AlertTriangle, Wallet, Palette } from 'lucide-vue-next'

const route = useRoute()

const items = [
  { label: 'Overview',  path: '/dashboard',      icon: LayoutDashboard },
  { label: 'Campaigns', path: '/campaigns',       icon: Megaphone },
  { label: 'Anomalies', path: '/anomalies',       icon: AlertTriangle },
  { label: 'Budget',    path: '/budget',          icon: Wallet },
  { label: 'Creative',  path: '/creatives',       icon: Palette },
]

function isActive(path: string) {
  return route.path.startsWith(path)
}
</script>

<template>
  <nav class="fixed bottom-0 inset-x-0 z-30 bg-bg-surface/95 backdrop-blur border-t border-bg-border/50 flex safe-area-pb">
    <NuxtLink
      v-for="item in items"
      :key="item.path"
      :to="item.path"
      class="flex-1 flex flex-col items-center gap-1 py-2 text-xs transition-colors"
      :class="isActive(item.path) ? 'text-blue-bright' : 'text-muted hover:text-secondary'"
    >
      <component :is="item.icon" class="w-5 h-5" />
      <span>{{ item.label }}</span>
    </NuxtLink>
  </nav>
</template>
