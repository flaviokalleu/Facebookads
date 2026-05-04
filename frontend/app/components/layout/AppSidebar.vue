<script setup lang="ts">
import {
  LayoutDashboard,
  Building2,
  Megaphone,
  Users,
  MessageSquare,
  Settings,
  ScrollText,
  History,
  Key,
} from 'lucide-vue-next'

const route = useRoute()

const links = [
  { to: '/dashboard', label: 'Painel', icon: LayoutDashboard },
  { to: '/imoveis', label: 'Imóveis', icon: Building2 },
  { to: '/campanhas', label: 'Anúncios', icon: Megaphone },
  { to: '/publicos', label: 'Públicos', icon: Users },
  { to: '/ia/chat', label: 'IA', icon: MessageSquare },
  { to: '/ia/regras', label: 'Regras', icon: ScrollText },
  { to: '/ia/historico', label: 'Histórico', icon: History },
  { to: '/ajustes/api-keys', label: 'Chaves de IA', icon: Key },
  { to: '/ajustes/token', label: 'Ajustes', icon: Settings },
]

function isActive(to: string) {
  if (to === '/dashboard') return route.path === '/dashboard'
  return route.path.startsWith(to)
}
</script>

<template>
  <aside class="fixed left-0 top-0 flex h-screen w-60 flex-col bg-accent-deep text-white/90">
    <div class="flex h-16 items-center px-6 text-lg font-semibold tracking-tight text-white">
      Gestor IA
    </div>
    <nav class="flex-1 space-y-1 px-3 pb-6">
      <NuxtLink
        v-for="l in links"
        :key="l.to"
        :to="l.to"
        :class="[
          'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition',
          isActive(l.to)
            ? 'bg-white/15 text-white'
            : 'text-white/80 hover:bg-white/10 hover:text-white',
        ]"
      >
        <component :is="l.icon" class="h-4 w-4" />
        <span>{{ l.label }}</span>
      </NuxtLink>
    </nav>
  </aside>
</template>
