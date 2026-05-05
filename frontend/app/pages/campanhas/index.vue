<script setup lang="ts">
import {
  Search, Calendar, ArrowDownUp, ExternalLink,
  Megaphone, Target, MessageCircle, Activity,
} from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import DateRangePicker from '~/components/ui/DateRangePicker.vue'

interface CampaignRow {
  id: string
  meta_campaign_id: string
  name: string
  status: string
  objective: string
  daily_budget: number
  lifetime_budget: number
  health_status: string
  account_meta_id: string
  account_name: string
  bm_name: string
  meta_start_time?: string
  meta_stop_time?: string
  days_running: number
  spend: number
  impressions: number
  clicks: number
  leads: number
  ctr: number
  cpl: number
  avg_frequency: number
}

const api = useApi()
const days = ref(7)
const items = ref<CampaignRow[]>([])
const loading = ref(true)
const statusFilter = ref<'all' | 'ACTIVE' | 'PAUSED' | 'ARCHIVED' | 'DELETED'>('ACTIVE')
const searchQuery = ref('')
const sortBy = ref<'spend' | 'leads' | 'cpl' | 'name'>('spend')

async function load() {
  loading.value = true
  try {
    const params = new URLSearchParams()
    params.set('days', String(days.value))
    if (statusFilter.value !== 'all') params.set('status', statusFilter.value)
    if (searchQuery.value.trim()) params.set('q', searchQuery.value.trim())
    const res = await api.get<{ data: CampaignRow[] }>(`/campanhas?${params.toString()}`)
    items.value = res.data || []
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch([days, statusFilter], load)

let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(searchQuery, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(load, 300)
})

const sorted = computed(() => {
  const arr = [...items.value]
  if (sortBy.value === 'spend') arr.sort((a, b) => b.spend - a.spend)
  else if (sortBy.value === 'leads') arr.sort((a, b) => b.leads - a.leads)
  else if (sortBy.value === 'cpl') arr.sort((a, b) => {
    if (a.leads === 0 && b.leads === 0) return 0
    if (a.leads === 0) return 1
    if (b.leads === 0) return -1
    return a.cpl - b.cpl
  })
  else arr.sort((a, b) => a.name.localeCompare(b.name))
  return arr
})

const totals = computed(() => {
  const t = { spend: 0, leads: 0, impressions: 0, clicks: 0 }
  for (const it of items.value) {
    t.spend += it.spend; t.leads += it.leads
    t.impressions += it.impressions; t.clicks += it.clicks
  }
  return { ...t, avgCPL: t.leads > 0 ? t.spend / t.leads : 0 }
})

const brl = (v: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v || 0)
const num = (v: number) => new Intl.NumberFormat('pt-BR').format(v || 0)
const pct = (v: number) => `${((v || 0) * 100).toFixed(2)}%`

function statusBadge(s: string): 'success' | 'warning' | 'neutral' | 'danger' {
  return s === 'ACTIVE' ? 'success' : s === 'PAUSED' ? 'neutral' : s === 'DELETED' ? 'danger' : 'warning'
}
function statusLabel(s: string) {
  return ({ ACTIVE: 'No ar', PAUSED: 'Pausada', ARCHIVED: 'Arquivada', DELETED: 'Removida' } as Record<string, string>)[s] || s
}
function objectiveLabel(o: string) {
  return ({
    OUTCOME_LEADS: 'Captar contatos', OUTCOME_TRAFFIC: 'Levar ao site',
    OUTCOME_AWARENESS: 'Reconhecimento', OUTCOME_ENGAGEMENT: 'Engajar',
    OUTCOME_SALES: 'Vendas', OUTCOME_APP_PROMOTION: 'App',
    LEAD_GENERATION: 'Captar contatos', CONVERSIONS: 'Conversões', LINK_CLICKS: 'Cliques',
  } as Record<string, string>)[o] || o
}
function healthDot(h: string) {
  return h === 'HEALTHY' ? 'bg-success' : h === 'AT_RISK' ? 'bg-warning' : h === 'CRITICAL' ? 'bg-danger' : 'bg-ink-faint'
}
function ctrColor(v: number) { return v < 0.01 ? 'text-danger' : v < 0.02 ? 'text-warning' : 'text-ink' }
function freqColor(v: number) { return v >= 3.5 ? 'text-danger' : v >= 2.5 ? 'text-warning' : 'text-ink-muted' }

function metaManagerLink(c: CampaignRow) {
  const acct = c.account_meta_id.replace(/^act_/, '')
  return `https://adsmanager.facebook.com/adsmanager/manage/ads?act=${acct}&selected_campaign_ids=${c.meta_campaign_id}`
}
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-ink">Anúncios</h1>
        <p class="text-sm text-ink-muted">Todas as campanhas, em todas as suas contas, num lugar só.</p>
      </div>
      <DateRangePicker v-model="days" />
    </div>

    <!-- Resumo agregado -->
    <div v-if="!loading && items.length" class="grid gap-3 sm:grid-cols-4">
      <div class="rounded-xl border border-border bg-bg p-4">
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <Megaphone class="h-3.5 w-3.5" /> {{ items.length }} {{ items.length === 1 ? 'campanha' : 'campanhas' }}
        </div>
        <p class="mt-1 text-xl font-semibold tabular-nums text-ink">
          {{ items.filter((c) => c.status === 'ACTIVE').length }} no ar
        </p>
      </div>
      <div class="rounded-xl border border-border bg-bg p-4">
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <Target class="h-3.5 w-3.5" /> Investimento
        </div>
        <p class="mt-1 text-xl font-semibold tabular-nums text-ink">{{ brl(totals.spend) }}</p>
      </div>
      <div class="rounded-xl border border-border bg-bg p-4">
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <MessageCircle class="h-3.5 w-3.5" /> Contatos
        </div>
        <p class="mt-1 text-xl font-semibold tabular-nums text-ink">{{ num(totals.leads) }}</p>
      </div>
      <div class="rounded-xl border border-border bg-bg p-4">
        <div class="flex items-center gap-2 text-xs text-ink-muted">
          <Activity class="h-3.5 w-3.5" /> Custo médio
        </div>
        <p class="mt-1 text-xl font-semibold tabular-nums text-ink">
          {{ totals.leads > 0 ? brl(totals.avgCPL) : '—' }}
        </p>
      </div>
    </div>

    <!-- Filtros -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="inline-flex items-center gap-1 rounded-full bg-bg-muted p-1">
        <button
          v-for="opt in [
            { v: 'ACTIVE',  l: 'No ar' },
            { v: 'PAUSED',  l: 'Pausadas' },
            { v: 'all',     l: 'Tudo' },
          ]"
          :key="opt.v"
          type="button"
          :class="[
            'rounded-full px-3 py-1 text-xs font-medium transition',
            statusFilter === opt.v ? 'bg-bg text-ink shadow-sm' : 'text-ink-muted hover:text-ink',
          ]"
          @click="statusFilter = opt.v as any"
        >{{ opt.l }}</button>
      </div>

      <div class="flex-1 max-w-sm relative">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-faint" />
        <input
          v-model="searchQuery"
          placeholder="Buscar por nome..."
          class="w-full rounded-lg border border-border bg-bg pl-9 pr-3 py-2 text-sm text-ink placeholder:text-ink-faint focus:border-accent focus:shadow-focus focus:outline-none"
        />
      </div>

      <div class="ml-auto flex items-center gap-2">
        <ArrowDownUp class="h-3.5 w-3.5 text-ink-faint" />
        <select
          v-model="sortBy"
          class="rounded-lg border border-border bg-bg px-2 py-1.5 text-xs text-ink focus:border-accent focus:shadow-focus focus:outline-none"
        >
          <option value="spend">Mais investido</option>
          <option value="leads">Mais contatos</option>
          <option value="cpl">Menor custo</option>
          <option value="name">Nome (A–Z)</option>
        </select>
      </div>
    </div>

    <!-- Skeleton -->
    <div v-if="loading" class="space-y-3">
      <UiSkeleton v-for="i in 4" :key="i" class="h-24" />
    </div>

    <!-- Vazio -->
    <UiCard v-else-if="!items.length" class="!p-8 text-center">
      <Megaphone class="mx-auto h-10 w-10 text-ink-faint" />
      <p class="mt-3 font-medium text-ink">
        {{ searchQuery ? 'Nenhuma campanha encontrada' : 'Nenhuma campanha sincronizada' }}
      </p>
      <p class="mt-1 text-sm text-ink-muted">
        {{ searchQuery ? 'Tente outro termo ou troque o status.' : 'A próxima sincronização chega em até 30min.' }}
      </p>
    </UiCard>

    <!-- Lista -->
    <div v-else class="space-y-2">
      <NuxtLink
        v-for="c in sorted"
        :key="c.id"
        :to="`/campanhas/${c.id}`"
        class="block transition hover:[&>*]:shadow-sm"
      >
      <UiCard class="!p-4">
        <div class="flex items-start gap-4">
          <span :class="['mt-2 inline-block h-2 w-2 shrink-0 rounded-full', healthDot(c.health_status)]" />

          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="font-medium text-ink truncate">{{ c.name }}</h3>
              <UiBadge :variant="statusBadge(c.status)">{{ statusLabel(c.status) }}</UiBadge>
              <UiBadge variant="neutral">{{ objectiveLabel(c.objective) }}</UiBadge>
            </div>
            <p class="mt-1 text-xs text-ink-muted truncate">
              <NuxtLink :to="`/contas/${c.account_meta_id}`" class="hover:text-ink hover:underline">
                {{ c.account_name }}
              </NuxtLink>
              <span v-if="c.bm_name"> · {{ c.bm_name }}</span>
            </p>
            <p class="mt-1 text-xs text-ink-faint flex flex-wrap gap-x-3 gap-y-0.5">
              <span v-if="c.days_running > 0" class="flex items-center gap-1">
                <Calendar class="h-3 w-3" />
                {{ c.days_running }} {{ c.days_running === 1 ? 'dia' : 'dias' }}
                <span v-if="c.meta_start_time">
                  (desde {{ new Date(c.meta_start_time).toLocaleDateString('pt-BR') }})
                </span>
              </span>
              <span v-if="c.daily_budget > 0">· {{ brl(c.daily_budget) }}/dia</span>
              <span v-if="c.meta_stop_time" class="text-warning">
                · termina {{ new Date(c.meta_stop_time).toLocaleDateString('pt-BR') }}
              </span>
            </p>
          </div>

          <div class="hidden lg:grid grid-cols-4 gap-x-6 gap-y-1 text-right shrink-0">
            <div>
              <p class="text-xs text-ink-faint">Investido</p>
              <p class="text-sm font-semibold tabular-nums text-ink">{{ brl(c.spend) }}</p>
            </div>
            <div>
              <p class="text-xs text-ink-faint">Contatos</p>
              <p class="text-sm font-semibold tabular-nums text-ink">{{ c.leads }}</p>
            </div>
            <div>
              <p class="text-xs text-ink-faint">Custo</p>
              <p class="text-sm font-semibold tabular-nums text-ink">{{ c.leads > 0 ? brl(c.cpl) : '—' }}</p>
            </div>
            <div>
              <p class="text-xs text-ink-faint">CTR / Freq</p>
              <p class="text-sm tabular-nums">
                <span :class="ctrColor(c.ctr)">{{ pct(c.ctr) }}</span>
                <span class="text-ink-faint mx-1">·</span>
                <span :class="freqColor(c.avg_frequency)">{{ c.avg_frequency ? c.avg_frequency.toFixed(1) + '×' : '—' }}</span>
              </p>
            </div>
          </div>

          <a
            :href="metaManagerLink(c)"
            target="_blank"
            rel="noopener"
            class="shrink-0 rounded-lg p-2 text-ink-faint hover:bg-bg-muted hover:text-ink"
            title="Abrir no Gerenciador da Meta"
            @click.stop
          >
            <ExternalLink class="h-4 w-4" />
          </a>
        </div>

        <!-- Métricas mobile -->
        <div class="mt-3 grid grid-cols-4 gap-2 lg:hidden">
          <div class="text-center">
            <p class="text-[10px] text-ink-faint uppercase tracking-wide">Gasto</p>
            <p class="text-sm font-semibold tabular-nums text-ink">{{ brl(c.spend) }}</p>
          </div>
          <div class="text-center">
            <p class="text-[10px] text-ink-faint uppercase tracking-wide">Contatos</p>
            <p class="text-sm font-semibold tabular-nums text-ink">{{ c.leads }}</p>
          </div>
          <div class="text-center">
            <p class="text-[10px] text-ink-faint uppercase tracking-wide">Custo cada</p>
            <p class="text-sm font-semibold tabular-nums text-ink">{{ c.leads > 0 ? brl(c.cpl) : '—' }}</p>
          </div>
          <div class="text-center">
            <p class="text-[10px] text-ink-faint uppercase tracking-wide">Freq</p>
            <p :class="['text-sm font-semibold tabular-nums', freqColor(c.avg_frequency)]">
              {{ c.avg_frequency ? c.avg_frequency.toFixed(1) + '×' : '—' }}
            </p>
          </div>
        </div>
      </UiCard>
      </NuxtLink>
    </div>
  </div>
</template>
