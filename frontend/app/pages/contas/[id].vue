<script setup lang="ts">
import {
  ArrowLeft, Sparkles, AlertCircle, CheckCircle2, AlertTriangle,
  TrendingUp, TrendingDown, Minus, Calendar, Clock, Trophy,
  Loader2, RefreshCw, ChevronDown, ChevronUp, MapPin, Layout,
  Users as UsersIcon,
} from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import DateRangePicker from '~/components/ui/DateRangePicker.vue'
import ComboChart from '~/components/charts/ComboChart.vue'
import HourHeatmap from '~/components/charts/HourHeatmap.vue'
import BreakdownList from '~/components/charts/BreakdownList.vue'
import AgeGenderChart from '~/components/charts/AgeGenderChart.vue'
import Funnel from '~/components/charts/Funnel.vue'
import { useBreakdowns, type BreakdownRow } from '~/composables/useBreakdowns'

interface Account {
  meta_id: string; name: string; currency: string; access_kind: string
  account_status: number; balance: number; amount_spent: number; spend_cap: number
  bm_name: string | null
}
interface Kpis {
  spend_7d: number; spend_prev_7d: number
  impressions_7d: number; clicks_7d: number
  leads_7d: number; leads_prev_7d: number
  avg_cpl_7d: number; avg_cpl_prev_7d: number
  avg_ctr_7d: number
  active_campaigns: number; paused_campaigns: number
  days_balance_left?: number
  best_day?: string; best_day_leads: number; best_day_cpl: number
}
interface CampaignRow {
  id: string; meta_campaign_id: string; name: string; status: string; objective: string
  daily_budget: number; health_status: string
  spend_7d: number; impressions_7d: number; clicks_7d: number; leads_7d: number
  ctr_7d: number; cpl_7d: number; avg_frequency_7d: number
  first_insight_date?: string
  meta_created_time?: string; meta_start_time?: string; meta_stop_time?: string
  days_running: number
}
interface DailyPoint { date: string; spend: number; impressions: number; clicks: number; leads: number; cpl: number }
interface Highlight { kind: 'good' | 'warn' | 'bad'; title: string; detail: string }
interface NextAction { priority: 'high' | 'medium' | 'low'; action: string }
interface Analysis {
  summary: string
  highlights?: Highlight[]
  next_actions?: NextAction[]
  model_used?: string
  created_at?: string
}

const route = useRoute()
const api = useApi()
const breakdowns = useBreakdowns()
const accountId = computed(() => String(route.params.id))

const days = ref(7)
const account = ref<Account | null>(null)
const kpis = ref<Kpis | null>(null)
const campaigns = ref<CampaignRow[]>([])
const daily = ref<DailyPoint[]>([])
const analysis = ref<Analysis | null>(null)
const analysisLoading = ref(false)
const loading = ref(true)
const errorMsg = ref<string | null>(null)
const analysisError = ref<string | null>(null)
const showDetails = ref(false)
const lastLoadedAt = ref<Date | null>(null)

// Breakdowns
const bdRegion    = ref<BreakdownRow[]>([])
const bdHour      = ref<BreakdownRow[]>([])
const bdAgeGender = ref<BreakdownRow[]>([])
const bdPlacement = ref<BreakdownRow[]>([])
const bdDevice    = ref<BreakdownRow[]>([])
const bdLoading   = ref(false)

async function loadCore() {
  loading.value = true; errorMsg.value = null
  try {
    const [detail, list, dly, an] = await Promise.all([
      api.get<{ data: { account: Account; kpis: Kpis } }>(`/contas/${accountId.value}?days=${days.value}`),
      api.get<{ data: CampaignRow[] }>(`/contas/${accountId.value}/campanhas?days=${days.value}`),
      api.get<{ data: DailyPoint[] }>(`/contas/${accountId.value}/insights/daily?days=${Math.max(days.value, 7)}`),
      api.get<{ data: any }>(`/contas/${accountId.value}/analysis`).catch(() => ({ data: null })),
    ])
    account.value = detail.data.account
    kpis.value = detail.data.kpis
    campaigns.value = list.data || []
    daily.value = dly.data || []
    if (an?.data) {
      analysis.value = {
        summary: an.data.summary,
        highlights: an.data.highlights?.highlights || an.data.highlights || [],
        next_actions: an.data.highlights?.next_actions || an.data.next_actions || [],
        model_used: an.data.model_used,
        created_at: an.data.created_at,
      }
    }
    lastLoadedAt.value = new Date()
  } catch (e: any) {
    errorMsg.value = e?.data?.error?.message || e?.message || 'Não foi possível carregar.'
  } finally {
    loading.value = false
  }
}

async function loadBreakdowns() {
  bdLoading.value = true
  const [r, h, ag, pl, dv] = await Promise.all([
    breakdowns.fetch(accountId.value, 'region',     days.value),
    breakdowns.fetch(accountId.value, 'hour',       days.value),
    breakdowns.fetch(accountId.value, 'age_gender', days.value),
    breakdowns.fetch(accountId.value, 'placement',  days.value),
    breakdowns.fetch(accountId.value, 'device',     days.value),
  ])
  bdRegion.value    = r
  bdHour.value      = h
  bdAgeGender.value = ag
  bdPlacement.value = pl
  bdDevice.value    = dv
  bdLoading.value = false
}

async function loadAll() {
  await Promise.all([loadCore(), loadBreakdowns()])
}

async function runAnalysis() {
  analysisLoading.value = true; analysisError.value = null
  try {
    const res = await api.post<{ data: Analysis }>(`/contas/${accountId.value}/analyze`)
    analysis.value = res.data
  } catch (e: any) {
    analysisError.value = e?.data?.error?.message || e?.message || 'Falha ao analisar.'
  } finally {
    analysisLoading.value = false
  }
}

watch(days, async () => { await loadAll() })

onMounted(async () => {
  await loadAll()
  if (!analysis.value && !errorMsg.value) runAnalysis()
})

// ── Helpers ────────────────────────────────────────────────────────────────
const brl = (v: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v || 0)
const num = (v: number) => new Intl.NumberFormat('pt-BR').format(v || 0)
const pct = (v: number) => `${((v || 0) * 100).toFixed(1)}%`

function deltaPct(now: number, prev: number): number | null {
  if (prev === 0) return now > 0 ? 100 : null
  return ((now - prev) / prev) * 100
}

const spendDelta = computed(() => kpis.value ? deltaPct(kpis.value.spend_7d, kpis.value.spend_prev_7d) : null)
const leadsDelta = computed(() => kpis.value ? deltaPct(kpis.value.leads_7d, kpis.value.leads_prev_7d) : null)
const cplDelta   = computed(() => kpis.value && kpis.value.avg_cpl_prev_7d > 0
  ? deltaPct(kpis.value.avg_cpl_7d, kpis.value.avg_cpl_prev_7d) : null)

const dailyLabels  = computed(() => daily.value.map((d) => {
  const [, m, dd] = d.date.split('-'); return `${dd}/${m}`
}))
const spendSeries  = computed(() => daily.value.map((d) => d.spend))
const leadsSeries  = computed(() => daily.value.map((d) => d.leads))

// Quantos dias da janela escolhida realmente têm dados (para hint).
const daysWithData = computed(() => daily.value.filter((d) => d.spend > 0 || d.leads > 0 || d.impressions > 0).length)
const windowExceedsData = computed(() => daysWithData.value > 0 && days.value > daysWithData.value)
const filteredSpend = computed(() => kpis.value?.spend_7d ?? 0)
// Se a janela é maior que o histórico, mostrar um aviso explicativo.
const dataHint = computed(() => {
  if (!kpis.value) return null
  if (kpis.value.spend_7d === 0 && kpis.value.leads_7d === 0) return null
  if (windowExceedsData.value) {
    return `Sua conta tem ${daysWithData.value} ${daysWithData.value === 1 ? 'dia' : 'dias'} de histórico. Janelas maiores mostram o mesmo total.`
  }
  return null
})
const lastSyncText = computed(() => {
  if (!lastLoadedAt.value) return ''
  const min = Math.floor((Date.now() - lastLoadedAt.value.getTime()) / 60000)
  if (min < 1) return 'agora há pouco'
  return `há ${min} min`
})

// Status hero
const status = computed<{ tone: 'good' | 'warn' | 'bad'; light: string; bg: string; text: string; headline: string; body: string }>(() => {
  if (!kpis.value || !account.value) return { tone: 'warn', light: 'bg-warning', bg: 'bg-warning-soft border-warning/20', text: 'text-warning', headline: '', body: '' }
  const k = kpis.value
  const a = account.value

  if (k.leads_7d === 0 && k.spend_7d === 0) {
    return {
      tone: 'warn', light: 'bg-ink-faint',
      bg: 'bg-bg-muted border-border', text: 'text-ink-muted',
      headline: 'Nenhum anúncio rodou nos últimos dias',
      body: a.balance > 10
        ? 'A conta tem saldo, mas nada está veiculando. Verifique se as campanhas estão ativas.'
        : 'A conta está sem saldo e sem entrega. Recarregue para começar.',
    }
  }
  let tone: 'good' | 'warn' | 'bad' = 'good'
  const reasons: string[] = []
  if (k.leads_7d === 0 && k.spend_7d > 0) { tone = 'bad'; reasons.push('gastou sem trazer contatos') }
  else if (k.days_balance_left !== undefined && k.days_balance_left < 3) { tone = 'warn'; reasons.push(`saldo dura só mais ${Math.floor(k.days_balance_left)} dia(s)`) }
  else if (k.avg_ctr_7d > 0 && k.avg_ctr_7d < 0.01) { tone = 'warn'; reasons.push('taxa de clique baixa') }
  else if (cplDelta.value !== null && cplDelta.value > 30) { tone = 'warn'; reasons.push('custo subindo rápido') }

  const headline = tone === 'good'
    ? `Tudo indo bem — ${k.leads_7d} ${k.leads_7d === 1 ? 'cliente chamou' : 'clientes chamaram'} a R$ ${k.avg_cpl_7d.toFixed(2)} cada nos últimos ${days.value} dias`
    : tone === 'warn'
      ? `Atenção — ${reasons.join(', ')}`
      : `Algo errado — ${reasons.join(', ')}`
  const body = tone === 'good'
    ? `Você investiu ${brl(k.spend_7d)} e gerou ${k.leads_7d} contatos. Custo por contato em ${brl(k.avg_cpl_7d)}.`
    : tone === 'warn' ? 'Vale revisar antes que piore. Veja as sugestões abaixo.' : 'Recomendamos pausar ou reformular a campanha.'
  return tone === 'good'
    ? { tone, light: 'bg-success', bg: 'bg-success-soft border-success/20', text: 'text-success', headline, body }
    : tone === 'warn'
      ? { tone, light: 'bg-warning', bg: 'bg-warning-soft border-warning/20', text: 'text-warning', headline, body }
      : { tone, light: 'bg-danger', bg: 'bg-danger-soft border-danger/20', text: 'text-danger', headline, body }
})

const bestDayText = computed(() => {
  if (!kpis.value?.best_day) return null
  const [, m, dd] = kpis.value.best_day.split('-')
  return { label: `${dd}/${m}`, leads: kpis.value.best_day_leads, cpl: kpis.value.best_day_cpl }
})

function statusBadge(s: string): 'success' | 'warning' | 'neutral' | 'danger' {
  return s === 'ACTIVE' ? 'success' : s === 'PAUSED' ? 'neutral' : s === 'DELETED' ? 'danger' : 'warning'
}
function statusLabel(s: string) {
  return ({ ACTIVE: 'No ar', PAUSED: 'Pausada', ARCHIVED: 'Arquivada', DELETED: 'Removida' } as Record<string, string>)[s] || s
}
function objectiveLabel(o: string) {
  return ({
    OUTCOME_LEADS: 'Captar contatos', OUTCOME_TRAFFIC: 'Levar ao site',
    OUTCOME_AWARENESS: 'Mostrar a marca', OUTCOME_ENGAGEMENT: 'Engajar',
    OUTCOME_SALES: 'Vendas', OUTCOME_APP_PROMOTION: 'App',
    LEAD_GENERATION: 'Captar contatos', CONVERSIONS: 'Conversões', LINK_CLICKS: 'Cliques',
  } as Record<string, string>)[o] || o
}
function placementLabel(p: string) {
  return ({
    facebook: 'Facebook', instagram: 'Instagram', audience_network: 'Audience Network', messenger: 'Messenger',
  } as Record<string, string>)[p] || p
}
function deviceLabel(d: string) {
  return ({
    desktop: 'Computador', mobile_app: 'Celular (app)', mobile_web: 'Celular (web)',
    iphone: 'iPhone', ipad: 'iPad', android_smartphone: 'Android', android_tablet: 'Tablet Android',
  } as Record<string, string>)[d] || d
}

function campaignStory(c: CampaignRow): { tone: 'good' | 'warn' | 'bad'; line: string } {
  if (c.status !== 'ACTIVE') return { tone: 'warn', line: 'Está pausada — não está veiculando.' }
  if (c.days_running < 7) return { tone: 'warn', line: `Em fase de aprendizagem — ${7 - c.days_running} dia(s) pra estabilizar. A IA não vai mexer ainda.` }
  if (c.leads_7d === 0) return { tone: 'bad', line: 'Sem contatos nos últimos 7 dias. Vale revisar.' }
  if (c.avg_frequency_7d >= 3.5) return { tone: 'warn', line: `Frequência alta (${c.avg_frequency_7d.toFixed(1)}× por pessoa). Tempo de trocar criativo.` }
  if (c.ctr_7d > 0 && c.ctr_7d < 0.01) return { tone: 'warn', line: 'Taxa de clique baixa — anúncio pouco atraente.' }
  return { tone: 'good', line: `${c.leads_7d} contatos por ${brl(c.cpl_7d)} cada. Performance saudável.` }
}

const topActions = computed<NextAction[]>(() => {
  if (analysis.value?.next_actions?.length) return analysis.value.next_actions.slice(0, 3)
  const out: NextAction[] = []
  if (!kpis.value || !account.value) return out
  if (kpis.value.days_balance_left !== undefined && kpis.value.days_balance_left < 3) {
    out.push({ priority: 'high', action: `Recarregar saldo — dá pra mais ${Math.floor(kpis.value.days_balance_left)} dia(s) no ritmo atual.` })
  }
  for (const c of campaigns.value) {
    if (c.days_running < 7 && c.status === 'ACTIVE') {
      out.push({ priority: 'low', action: `Esperar ${7 - c.days_running} dia(s) na campanha "${c.name}" — ainda em aprendizagem.` })
      break
    }
  }
  if (out.length === 0) out.push({ priority: 'low', action: 'Continue acompanhando. A IA vai avisar se algo mudar.' })
  return out
})

function priorityStyle(p: string) {
  return p === 'high'
    ? { ring: 'ring-danger/30 bg-danger-soft', dot: 'bg-danger', label: 'Urgente', text: 'text-danger' }
    : p === 'medium'
      ? { ring: 'ring-warning/30 bg-warning-soft', dot: 'bg-warning', label: 'Importante', text: 'text-warning' }
      : { ring: 'ring-border bg-bg', dot: 'bg-ink-faint', label: 'Quando der', text: 'text-ink-muted' }
}
function highlightStyle(kind: string) {
  return ({
    good: { icon: CheckCircle2, color: 'text-success' },
    warn: { icon: AlertTriangle, color: 'text-warning' },
    bad:  { icon: AlertCircle,    color: 'text-danger' },
  } as Record<string, { icon: any; color: string }>)[kind] || { icon: AlertCircle, color: 'text-ink-muted' }
}

// ── Breakdown view-models ───────────────────────────────────────────────────
// Region: Meta não agrega leads aqui — usar cliques como proxy.
const regionUsesClicks = computed(() => bdRegion.value.every((r) => r.leads === 0))
const regionItems = computed(() => bdRegion.value
  .filter((r) => r.dim.region)
  .sort((a, b) => (b.leads - a.leads) || (b.clicks - a.clicks) || (b.spend - a.spend))
  .slice(0, 10)
  .map((r) => {
    const useClicks = regionUsesClicks.value
    const v = useClicks ? r.clicks : r.leads
    return {
      label: r.dim.region,
      value: v,
      spend: r.spend,
      cpl: useClicks ? (r.clicks > 0 ? r.spend / r.clicks : 0) : r.cpl,
    }
  }))

// Hour: idem — usar cliques.
const hourUsesClicks = computed(() => bdHour.value.every((r) => r.leads === 0))
const hourValues = computed(() => {
  const arr = new Array(24).fill(0)
  for (const r of bdHour.value) {
    const h = parseInt(r.dim.hourly_stats_aggregated_by_advertiser_time_zone || '0', 10)
    if (Number.isNaN(h) || h < 0 || h >= 24) continue
    arr[h] = hourUsesClicks.value ? r.clicks : r.leads
  }
  return arr
})

const ageGenderRows = computed(() => bdAgeGender.value.map((r) => ({
  age: r.dim.age || 'unknown',
  gender: r.dim.gender || 'unknown',
  value: r.leads,
})))

const placementItems = computed(() => bdPlacement.value
  .filter((r) => r.dim.publisher_platform)
  .sort((a, b) => b.leads - a.leads)
  .map((r) => ({ label: placementLabel(r.dim.publisher_platform), value: r.leads, spend: r.spend, cpl: r.cpl })))

const deviceItems = computed(() => bdDevice.value
  .filter((r) => r.dim.impression_device)
  .sort((a, b) => b.leads - a.leads)
  .map((r) => ({ label: deviceLabel(r.dim.impression_device), value: r.leads, spend: r.spend, cpl: r.cpl })))

// Funnel
const funnelSteps = computed(() => {
  if (!kpis.value) return []
  const k = kpis.value
  return [
    { label: 'Pessoas alcançadas (impressões)', value: k.impressions_7d, hint: 'Quantas vezes seu anúncio apareceu' },
    { label: 'Cliques no anúncio',              value: k.clicks_7d,      hint: 'Quem se interessou e clicou' },
    { label: 'Contatos / leads',                value: k.leads_7d,       hint: 'Quem te chamou no WhatsApp ou virou lead' },
  ]
})
</script>

<template>
  <div class="space-y-5">
    <NuxtLink to="/dashboard" class="inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
      <ArrowLeft class="h-4 w-4" /> Voltar para o painel
    </NuxtLink>

    <div v-if="loading" class="space-y-4">
      <UiSkeleton class="h-32" />
      <UiSkeleton class="h-24" />
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiSkeleton v-for="i in 4" :key="i" class="h-28" />
      </div>
    </div>

    <div v-else-if="errorMsg" class="rounded-xl bg-danger-soft p-4 text-sm text-danger">{{ errorMsg }}</div>

    <template v-else-if="account && kpis">
      <!-- Header + DateRangePicker -->
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p class="text-xs uppercase tracking-wider text-ink-faint">{{ account.bm_name || 'Conta pessoal' }}</p>
          <h1 class="mt-0.5 text-2xl font-semibold tracking-tight text-ink">{{ account.name }}</h1>
          <p class="mt-0.5 flex items-center gap-2 text-xs text-ink-muted">
            <span class="font-mono">{{ account.meta_id }}</span>
            <span class="inline-flex items-center gap-1">
              <span class="relative flex h-2 w-2">
                <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-50"></span>
                <span class="relative inline-flex h-2 w-2 rounded-full bg-success"></span>
              </span>
              atualizado {{ lastSyncText }}
            </span>
          </p>
        </div>
        <DateRangePicker v-model="days" />
      </div>

      <!-- STATUS HERO -->
      <div :class="['rounded-2xl border p-6', status.bg]">
        <div class="flex items-start gap-4">
          <span :class="['mt-2 inline-block h-3 w-3 shrink-0 rounded-full', status.light]" />
          <div class="flex-1 min-w-0">
            <h2 class="text-lg font-semibold leading-tight text-ink">{{ status.headline }}</h2>
            <p class="mt-1 text-sm text-ink-muted">{{ status.body }}</p>
          </div>
        </div>
      </div>

      <!-- Hint quando filtro maior que histórico disponível -->
      <div
        v-if="dataHint"
        class="flex items-start gap-2 rounded-lg border border-border bg-bg-muted px-3 py-2 text-xs text-ink-muted"
      >
        <AlertCircle class="h-4 w-4 shrink-0 text-ink-faint" />
        <span>{{ dataHint }}</span>
      </div>

      <!-- O QUE FAZER AGORA -->
      <UiCard>
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <Sparkles class="h-5 w-5 text-accent" />
            <h2 class="text-base font-semibold text-ink">O que fazer agora</h2>
          </div>
          <UiButton variant="ghost" size="sm" :loading="analysisLoading" @click="runAnalysis">
            <RefreshCw v-if="!analysisLoading" class="h-4 w-4" /> Atualizar
          </UiButton>
        </div>
        <div v-if="analysisLoading && !analysis" class="mt-4 flex items-center gap-2 text-sm text-ink-muted">
          <Loader2 class="h-4 w-4 animate-spin text-accent" /> A IA está pensando...
        </div>
        <ul v-else class="mt-4 space-y-2">
          <li
            v-for="(a, i) in topActions"
            :key="i"
            :class="['flex items-start gap-3 rounded-lg p-3 ring-1', priorityStyle(a.priority).ring]"
          >
            <span :class="['mt-1.5 inline-block h-2 w-2 shrink-0 rounded-full', priorityStyle(a.priority).dot]" />
            <div class="flex-1 min-w-0">
              <p :class="['text-xs font-semibold uppercase tracking-wide', priorityStyle(a.priority).text]">
                {{ priorityStyle(a.priority).label }}
              </p>
              <p class="mt-0.5 text-sm text-ink">{{ a.action }}</p>
            </div>
          </li>
        </ul>
      </UiCard>

      <!-- KPI grid -->
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard>
          <p class="text-xs text-ink-muted">Quantos clientes me chamaram?</p>
          <p class="mt-2 text-3xl font-semibold tabular-nums text-ink">{{ num(kpis.leads_7d) }}</p>
          <p class="mt-1 text-xs text-ink-muted">nos últimos {{ days }} dias</p>
          <div v-if="leadsDelta !== null" class="mt-2 flex items-center gap-1 text-xs">
            <TrendingUp v-if="leadsDelta > 0" class="h-3.5 w-3.5 text-success" />
            <TrendingDown v-else-if="leadsDelta < 0" class="h-3.5 w-3.5 text-danger" />
            <Minus v-else class="h-3.5 w-3.5 text-ink-faint" />
            <span :class="leadsDelta > 0 ? 'text-success' : leadsDelta < 0 ? 'text-danger' : 'text-ink-faint'">
              {{ leadsDelta > 0 ? '+' : '' }}{{ leadsDelta.toFixed(0) }}% vs período anterior
            </span>
          </div>
        </UiCard>
        <UiCard>
          <p class="text-xs text-ink-muted">Quanto custou cada cliente?</p>
          <p class="mt-2 text-3xl font-semibold tabular-nums text-ink">
            {{ kpis.leads_7d ? brl(kpis.avg_cpl_7d) : '—' }}
          </p>
          <p class="mt-1 text-xs text-ink-muted">média do período</p>
          <div v-if="cplDelta !== null" class="mt-2 flex items-center gap-1 text-xs">
            <TrendingDown v-if="cplDelta < 0" class="h-3.5 w-3.5 text-success" />
            <TrendingUp v-else-if="cplDelta > 0" class="h-3.5 w-3.5 text-danger" />
            <Minus v-else class="h-3.5 w-3.5 text-ink-faint" />
            <span :class="cplDelta < 0 ? 'text-success' : cplDelta > 0 ? 'text-danger' : 'text-ink-faint'">
              {{ cplDelta > 0 ? '+' : '' }}{{ cplDelta.toFixed(0) }}% vs período anterior
            </span>
          </div>
        </UiCard>
        <UiCard>
          <p class="text-xs text-ink-muted">Quanto investi?</p>
          <p class="mt-2 text-3xl font-semibold tabular-nums text-ink">{{ brl(kpis.spend_7d) }}</p>
          <p class="mt-1 text-xs text-ink-muted">nos últimos {{ days }} dias</p>
          <div v-if="spendDelta !== null" class="mt-2 flex items-center gap-1 text-xs">
            <TrendingUp v-if="spendDelta > 0" class="h-3.5 w-3.5 text-ink-muted" />
            <TrendingDown v-else-if="spendDelta < 0" class="h-3.5 w-3.5 text-ink-muted" />
            <Minus v-else class="h-3.5 w-3.5 text-ink-faint" />
            <span class="text-ink-muted">{{ spendDelta > 0 ? '+' : '' }}{{ spendDelta.toFixed(0) }}% vs período anterior</span>
          </div>
        </UiCard>
        <UiCard>
          <p class="text-xs text-ink-muted">Saldo na conta</p>
          <p class="mt-2 text-3xl font-semibold tabular-nums text-ink">{{ brl(account.balance) }}</p>
          <p
            v-if="kpis.days_balance_left !== undefined"
            :class="['mt-1 text-xs', kpis.days_balance_left < 3 ? 'text-danger font-medium' : 'text-ink-muted']"
          >
            <Clock class="inline h-3 w-3 mr-0.5" />
            dá pra mais ~{{ Math.max(0, Math.floor(kpis.days_balance_left)) }} {{ Math.floor(kpis.days_balance_left) === 1 ? 'dia' : 'dias' }} no ritmo atual
          </p>
          <p v-else class="mt-1 text-xs text-ink-muted">total já gasto: {{ brl(account.amount_spent) }}</p>
        </UiCard>
      </div>

      <!-- Combo chart + Funnel -->
      <div class="grid gap-4 lg:grid-cols-3">
        <UiCard class="lg:col-span-2">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-ink">Investimento × contatos por dia</h2>
              <p class="text-sm text-ink-muted">Últimos {{ Math.max(days, 7) }} dias</p>
            </div>
            <div v-if="bestDayText" class="flex items-center gap-2 rounded-lg bg-success-soft px-3 py-2 text-xs text-success">
              <Trophy class="h-4 w-4" />
              <span>
                Melhor dia: <strong>{{ bestDayText.label }}</strong> —
                {{ bestDayText.leads }} contatos a {{ brl(bestDayText.cpl) }} cada
              </span>
            </div>
          </div>
          <div class="mt-4">
            <ComboChart
              :bars="spendSeries"
              :line="leadsSeries"
              :labels="dailyLabels"
              :height="220"
              bar-color="#1877F2"
              line-color="#42B72A"
              bar-label="Investimento"
              line-label="Contatos"
            />
          </div>
        </UiCard>

        <UiCard>
          <h2 class="text-base font-semibold text-ink">Caminho do cliente</h2>
          <p class="text-sm text-ink-muted">Da impressão ao contato</p>
          <div class="mt-4">
            <Funnel :steps="funnelSteps" />
          </div>
        </UiCard>
      </div>

      <!-- BREAKDOWNS — 2x2 -->
      <div>
        <div class="mb-3 flex items-center justify-between">
          <h2 class="text-base font-semibold text-ink">Quem está chamando — e de onde</h2>
          <p v-if="bdLoading" class="text-xs text-ink-muted flex items-center gap-1">
            <Loader2 class="h-3 w-3 animate-spin" /> Carregando...
          </p>
        </div>
        <div class="grid gap-4 lg:grid-cols-2">
          <UiCard>
            <div class="flex items-center gap-2">
              <MapPin class="h-4 w-4 text-accent" />
              <h3 class="font-semibold text-ink">De onde vêm?</h3>
            </div>
            <p class="mt-1 text-xs text-ink-muted">
              Top 10 regiões com mais {{ regionUsesClicks ? 'cliques' : 'contatos' }}
            </p>
            <div class="mt-4">
              <BreakdownList :items="regionItems" empty-text="Aguardando dados regionais." />
            </div>
          </UiCard>

          <UiCard>
            <div class="flex items-center gap-2">
              <Clock class="h-4 w-4 text-accent" />
              <h3 class="font-semibold text-ink">Que horas o público está ativo?</h3>
            </div>
            <p class="mt-1 text-xs text-ink-muted">
              {{ hourUsesClicks ? 'Cliques' : 'Contatos' }} por hora do dia
            </p>
            <div class="mt-4">
              <HourHeatmap :values="hourValues" :label="hourUsesClicks ? 'cliques' : 'contatos'" />
            </div>
          </UiCard>

          <UiCard>
            <div class="flex items-center gap-2">
              <UsersIcon class="h-4 w-4 text-accent" />
              <h3 class="font-semibold text-ink">Quem chama?</h3>
            </div>
            <p class="mt-1 text-xs text-ink-muted">Idade e gênero dos contatos</p>
            <div class="mt-4">
              <AgeGenderChart :rows="ageGenderRows" empty-text="Aguardando dados demográficos." />
            </div>
          </UiCard>

          <UiCard>
            <div class="flex items-center gap-2">
              <Layout class="h-4 w-4 text-accent" />
              <h3 class="font-semibold text-ink">Onde aparece o anúncio?</h3>
            </div>
            <p class="mt-1 text-xs text-ink-muted">Plataforma e dispositivo</p>
            <div class="mt-4 grid gap-4 sm:grid-cols-2">
              <BreakdownList :items="placementItems" empty-text="Sem dados." />
              <BreakdownList :items="deviceItems" empty-text="Sem dados." />
            </div>
          </UiCard>
        </div>
      </div>

      <!-- Campanhas -->
      <div>
        <div class="mb-3 flex items-center justify-between">
          <h2 class="text-base font-semibold text-ink">Suas campanhas</h2>
          <p class="text-xs text-ink-muted">{{ campaigns.length }} no total · {{ kpis.active_campaigns }} no ar</p>
        </div>
        <UiCard v-if="!campaigns.length">
          <p class="text-sm text-ink-muted">Nenhuma campanha sincronizada ainda.</p>
        </UiCard>
        <div v-else class="space-y-3">
          <UiCard v-for="c in campaigns" :key="c.id" class="!p-4">
            <div class="flex items-start justify-between gap-4">
              <div class="flex items-start gap-3 min-w-0 flex-1">
                <span
                  class="mt-1.5 inline-block h-2 w-2 shrink-0 rounded-full"
                  :class="campaignStory(c).tone === 'good' ? 'bg-success' : campaignStory(c).tone === 'warn' ? 'bg-warning' : 'bg-danger'"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="font-medium text-ink">{{ c.name }}</h3>
                    <UiBadge :variant="statusBadge(c.status)">{{ statusLabel(c.status) }}</UiBadge>
                  </div>
                  <p class="mt-1 text-sm text-ink-muted">{{ campaignStory(c).line }}</p>
                  <p class="mt-2 text-xs text-ink-faint flex flex-wrap items-center gap-x-3 gap-y-1">
                    <span>
                      <Calendar class="inline h-3 w-3" />
                      {{ c.days_running }} {{ c.days_running === 1 ? 'dia rodando' : 'dias rodando' }}
                      <span v-if="c.meta_start_time" class="text-ink-faint">
                        (desde {{ new Date(c.meta_start_time).toLocaleDateString('pt-BR') }})
                      </span>
                    </span>
                    <span>· {{ objectiveLabel(c.objective) }}</span>
                    <span v-if="c.daily_budget > 0">· {{ brl(c.daily_budget) }}/dia</span>
                    <span v-if="c.meta_stop_time" class="text-warning">
                      · termina {{ new Date(c.meta_stop_time).toLocaleDateString('pt-BR') }}
                    </span>
                  </p>
                </div>
              </div>
              <div class="grid grid-cols-2 gap-x-6 gap-y-1 text-right">
                <div>
                  <p class="text-xs text-ink-faint">Investido</p>
                  <p class="text-sm font-semibold tabular-nums text-ink">{{ brl(c.spend_7d) }}</p>
                </div>
                <div>
                  <p class="text-xs text-ink-faint">Contatos</p>
                  <p class="text-sm font-semibold tabular-nums text-ink">{{ c.leads_7d }}</p>
                </div>
                <div>
                  <p class="text-xs text-ink-faint">Custo cada</p>
                  <p class="text-sm font-semibold tabular-nums text-ink">{{ c.leads_7d ? brl(c.cpl_7d) : '—' }}</p>
                </div>
                <div>
                  <p class="text-xs text-ink-faint">Frequência</p>
                  <p class="text-sm font-semibold tabular-nums text-ink">{{ c.avg_frequency_7d ? `${c.avg_frequency_7d.toFixed(1)}×` : '—' }}</p>
                </div>
              </div>
            </div>
          </UiCard>
        </div>
      </div>

      <!-- Diagnóstico completo da IA -->
      <UiCard v-if="analysis || analysisError">
        <button type="button" class="flex w-full items-center justify-between text-left" @click="showDetails = !showDetails">
          <div class="flex items-center gap-2">
            <Sparkles class="h-4 w-4 text-accent" />
            <span class="font-medium text-ink">{{ showDetails ? 'Esconder' : 'Ver' }} diagnóstico completo da IA</span>
          </div>
          <component :is="showDetails ? ChevronUp : ChevronDown" class="h-4 w-4 text-ink-muted" />
        </button>
        <div v-if="showDetails" class="mt-4 space-y-4">
          <div v-if="analysisError" class="rounded-lg bg-danger-soft p-3 text-sm text-danger">
            {{ analysisError }}
            <span v-if="analysisError.includes('api key')">— vá em <NuxtLink to="/ajustes/api-keys" class="underline">Ajustes → Chaves de IA</NuxtLink></span>
          </div>
          <p v-if="analysis?.summary" class="text-sm leading-relaxed text-ink">{{ analysis.summary }}</p>
          <div v-if="analysis?.highlights?.length" class="grid gap-2 sm:grid-cols-2">
            <div v-for="(h, i) in analysis.highlights" :key="i" class="flex items-start gap-2 rounded-lg border border-border p-3">
              <component :is="highlightStyle(h.kind).icon" :class="['mt-0.5 h-4 w-4 shrink-0', highlightStyle(h.kind).color]" />
              <div>
                <p class="text-sm font-medium text-ink">{{ h.title }}</p>
                <p class="mt-0.5 text-xs text-ink-muted">{{ h.detail }}</p>
              </div>
            </div>
          </div>
          <p v-if="analysis?.created_at" class="text-xs text-ink-faint">
            Análise gerada em {{ new Date(analysis.created_at).toLocaleString('pt-BR') }}
          </p>
        </div>
      </UiCard>
    </template>
  </div>
</template>
