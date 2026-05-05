<script setup lang="ts">
import {
  Users, Search, RefreshCw, Globe, Heart, ListChecks, Copy, Smartphone, Zap, ExternalLink,
  Loader2,
} from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import UiButton from '~/components/ui/UiButton.vue'

interface Audience {
  id: string
  name: string
  subtype: string
  subtype_label: string
  description: string
  count_low: number
  count_high: number
  delivery_status_code: number
  delivery_status_text: string
  operation_status: string
  time_created: number
  time_updated: number
  account_meta_id: string
  account_name: string
  bm_name: string
}

const api = useApi()
const audiences = ref<Audience[]>([])
const loading = ref(true)
const refreshing = ref(false)
const cachedUntil = ref<string | null>(null)
const subtypeFilter = ref<string>('all')
const searchQuery = ref('')
const accountFilter = ref<string>('all')

async function load(force = false) {
  if (force) refreshing.value = true
  else loading.value = true
  try {
    const url = force ? `/publicos?_=${Date.now()}` : '/publicos'
    const res = await api.get<{ data: { rows: Audience[]; cached_until: string } }>(url)
    audiences.value = res.data.rows || []
    cachedUntil.value = res.data.cached_until
  } finally {
    loading.value = false
    refreshing.value = false
  }
}
onMounted(() => load(false))

const subtypeIcons: Record<string, any> = {
  CUSTOM: ListChecks,
  WEBSITE: Globe,
  ENGAGEMENT: Heart,
  VIDEO: Heart,
  LOOKALIKE: Copy,
  APP: Smartphone,
  LEAD_AD: Zap,
  IG_BUSINESS: Heart,
}

const subtypeTabs = computed(() => {
  const counts: Record<string, number> = {}
  for (const a of audiences.value) {
    counts[a.subtype] = (counts[a.subtype] || 0) + 1
  }
  const order = ['LOOKALIKE', 'CUSTOM', 'ENGAGEMENT', 'WEBSITE', 'IG_BUSINESS', 'VIDEO', 'LEAD_AD', 'APP']
  const tabs = [{ value: 'all', label: 'Tudo', count: audiences.value.length }]
  for (const k of order) {
    if (counts[k]) tabs.push({ value: k, label: subtypeShort(k), count: counts[k] })
  }
  // Restos não mapeados
  for (const k of Object.keys(counts).sort()) {
    if (!order.includes(k)) tabs.push({ value: k, label: k, count: counts[k] })
  }
  return tabs
})

const accounts = computed(() => {
  const seen = new Map<string, string>()
  for (const a of audiences.value) {
    if (a.account_meta_id && !seen.has(a.account_meta_id)) {
      seen.set(a.account_meta_id, a.account_name || a.account_meta_id)
    }
  }
  return Array.from(seen.entries()).map(([id, name]) => ({ id, name })).sort((a, b) => a.name.localeCompare(b.name))
})

const filtered = computed(() => {
  return audiences.value.filter((a) => {
    if (subtypeFilter.value !== 'all' && a.subtype !== subtypeFilter.value) return false
    if (accountFilter.value !== 'all' && a.account_meta_id !== accountFilter.value) return false
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      return a.name.toLowerCase().includes(q) || a.description.toLowerCase().includes(q)
    }
    return true
  })
})

function subtypeShort(s: string) {
  return ({
    CUSTOM: 'Listas',
    WEBSITE: 'Site',
    ENGAGEMENT: 'Engajamento',
    VIDEO: 'Vídeo',
    LOOKALIKE: 'Sósias',
    APP: 'App',
    LEAD_AD: 'Lead Ads',
    IG_BUSINESS: 'Instagram',
  } as Record<string, string>)[s] || s
}

function formatCount(low: number, high: number) {
  if (low === 0 && high === 0) return 'Calculando...'
  if (low === high) return formatNum(low)
  return `${formatNum(low)} – ${formatNum(high)}`
}
function formatNum(v: number) {
  if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`
  if (v >= 1_000) return `${(v / 1_000).toFixed(0)}k`
  return String(v)
}
function formatDate(ts: number) {
  if (!ts) return '—'
  return new Date(ts * 1000).toLocaleDateString('pt-BR')
}
function deliveryBadge(code: number): 'success' | 'warning' | 'danger' | 'neutral' {
  if (code === 200) return 'success'
  if (code >= 400) return 'danger'
  return 'warning'
}
function metaManagerLink(a: Audience) {
  const acct = a.account_meta_id.replace(/^act_/, '')
  return `https://business.facebook.com/adsmanager/audiences?act=${acct}&selected_audience_ids=${a.id}`
}
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-ink">Públicos</h1>
        <p class="text-sm text-ink-muted">
          Listas e sósias salvas em todas as suas contas. Use no wizard de campanha pra mirar quem importa.
        </p>
      </div>
      <UiButton variant="ghost" size="sm" :loading="refreshing" @click="load(true)">
        <RefreshCw v-if="!refreshing" class="h-4 w-4" /> Atualizar
      </UiButton>
    </div>

    <!-- Resumo agregado -->
    <div v-if="!loading && audiences.length" class="grid gap-3 sm:grid-cols-3">
      <UiCard>
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <Users class="h-3.5 w-3.5" /> Total
        </div>
        <p class="mt-1 text-2xl font-semibold tabular-nums text-ink">{{ audiences.length }}</p>
        <p class="mt-1 text-xs text-ink-faint">{{ accounts.length }} {{ accounts.length === 1 ? 'conta' : 'contas' }} com públicos</p>
      </UiCard>
      <UiCard>
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <Copy class="h-3.5 w-3.5" /> Sósias (Lookalikes)
        </div>
        <p class="mt-1 text-2xl font-semibold tabular-nums text-ink">
          {{ audiences.filter((a) => a.subtype === 'LOOKALIKE').length }}
        </p>
        <p class="mt-1 text-xs text-ink-faint">públicos parecidos com seus melhores clientes</p>
      </UiCard>
      <UiCard>
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <Heart class="h-3.5 w-3.5" /> Engajamento
        </div>
        <p class="mt-1 text-2xl font-semibold tabular-nums text-ink">
          {{ audiences.filter((a) => a.subtype === 'ENGAGEMENT' || a.subtype === 'IG_BUSINESS' || a.subtype === 'VIDEO').length }}
        </p>
        <p class="mt-1 text-xs text-ink-faint">quem já interagiu com você</p>
      </UiCard>
    </div>

    <!-- Tabs subtype -->
    <div class="flex flex-wrap gap-2">
      <button
        v-for="t in subtypeTabs"
        :key="t.value"
        type="button"
        :class="[
          'rounded-full px-3 py-1.5 text-sm transition',
          subtypeFilter === t.value
            ? 'bg-accent text-white'
            : 'border border-border bg-bg text-ink-muted hover:text-ink',
        ]"
        @click="subtypeFilter = t.value"
      >
        {{ t.label }} <span class="ml-1 text-xs opacity-70">{{ t.count }}</span>
      </button>
    </div>

    <!-- Conta + busca -->
    <div class="flex flex-wrap items-center gap-3">
      <select
        v-model="accountFilter"
        class="rounded-lg border border-border bg-bg px-3 py-2 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none"
      >
        <option value="all">Todas as contas</option>
        <option v-for="a in accounts" :key="a.id" :value="a.id">{{ a.name }}</option>
      </select>
      <div class="flex-1 max-w-sm relative">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-faint" />
        <input
          v-model="searchQuery"
          placeholder="Buscar pelo nome..."
          class="w-full rounded-lg border border-border bg-bg pl-9 pr-3 py-2 text-sm text-ink placeholder:text-ink-faint focus:border-accent focus:shadow-focus focus:outline-none"
        />
      </div>
      <p class="ml-auto text-xs text-ink-muted">{{ filtered.length }} resultados</p>
    </div>

    <!-- Skeleton -->
    <div v-if="loading" class="space-y-3">
      <UiSkeleton v-for="i in 5" :key="i" class="h-20" />
    </div>

    <UiCard v-else-if="!audiences.length" class="!p-8 text-center">
      <Users class="mx-auto h-10 w-10 text-ink-faint" />
      <p class="mt-3 font-medium text-ink">Nenhum público encontrado</p>
      <p class="mt-1 text-sm text-ink-muted">
        Cria públicos no Gerenciador da Meta — depois aparecem aqui automaticamente.
      </p>
    </UiCard>

    <UiCard v-else-if="!filtered.length" class="!p-6">
      <p class="text-sm text-ink-muted">Nenhum público com esses filtros.</p>
    </UiCard>

    <!-- Lista -->
    <div v-else class="space-y-2">
      <UiCard v-for="a in filtered" :key="a.id" class="!p-4">
        <div class="flex items-start gap-4">
          <div
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full"
            :class="a.subtype === 'LOOKALIKE' ? 'bg-accent-soft text-accent' : 'bg-bg-muted text-ink-muted'"
          >
            <component :is="subtypeIcons[a.subtype] || Users" class="h-5 w-5" />
          </div>

          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="font-medium text-ink truncate">{{ a.name }}</h3>
              <UiBadge variant="neutral">{{ a.subtype_label || a.subtype }}</UiBadge>
              <UiBadge
                v-if="a.delivery_status_code !== 200"
                :variant="deliveryBadge(a.delivery_status_code)"
              >
                {{ a.delivery_status_text }}
              </UiBadge>
            </div>
            <p v-if="a.description" class="mt-1 text-xs text-ink-muted truncate">{{ a.description }}</p>
            <p class="mt-1 text-xs text-ink-faint flex flex-wrap gap-x-3 gap-y-0.5">
              <NuxtLink :to="`/contas/${a.account_meta_id}`" class="hover:text-ink hover:underline">
                {{ a.account_name }}
              </NuxtLink>
              <span v-if="a.time_created">· criado em {{ formatDate(a.time_created) }}</span>
            </p>
          </div>

          <div class="text-right shrink-0">
            <p class="text-xs text-ink-faint">Tamanho estimado</p>
            <p class="text-base font-semibold tabular-nums text-ink">{{ formatCount(a.count_low, a.count_high) }}</p>
            <p v-if="a.count_high > 0" class="text-[10px] text-ink-faint">pessoas</p>
          </div>

          <a
            :href="metaManagerLink(a)"
            target="_blank"
            rel="noopener"
            class="shrink-0 rounded-lg p-2 text-ink-faint hover:bg-bg-muted hover:text-ink"
            title="Abrir no Gerenciador da Meta"
          >
            <ExternalLink class="h-4 w-4" />
          </a>
        </div>
      </UiCard>
    </div>

    <p v-if="cachedUntil && !loading" class="text-center text-xs text-ink-faint">
      Dados em cache por {{ Math.max(0, Math.floor((new Date(cachedUntil).getTime() - Date.now()) / 60000)) }} min ·
      use o botão Atualizar pra forçar uma busca direto na Meta.
    </p>
  </div>
</template>
