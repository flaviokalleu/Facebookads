<script setup lang="ts">
import { Search } from 'lucide-vue-next'

const route = useRoute()

const labels: Record<string, string> = {
  dashboard: 'Painel',
  imoveis: 'Imóveis',
  campanhas: 'Anúncios',
  publicos: 'Públicos',
  ia: 'IA',
  chat: 'Conversa',
  regras: 'Regras',
  historico: 'Histórico',
  ajustes: 'Ajustes',
  token: 'Acesso Meta',
  nova: 'Novo',
}

const crumbs = computed(() =>
  route.path.split('/').filter(Boolean).map(seg => labels[seg] || seg),
)
</script>

<template>
  <header class="sticky top-0 z-10 flex h-16 items-center gap-4 border-b border-border bg-bg px-8">
    <nav class="flex items-center gap-2 text-sm text-ink-muted">
      <template v-for="(c, i) in crumbs" :key="i">
        <span v-if="i > 0" class="text-ink-faint">/</span>
        <span :class="i === crumbs.length - 1 ? 'text-ink' : ''">{{ c }}</span>
      </template>
    </nav>
    <div class="ml-auto flex items-center gap-2">
      <div class="flex h-9 w-72 items-center gap-2 rounded-lg border border-border bg-bg-muted px-3 text-sm text-ink-faint">
        <Search class="h-4 w-4" />
        <span>Buscar (em breve)</span>
      </div>
    </div>
  </header>
</template>
