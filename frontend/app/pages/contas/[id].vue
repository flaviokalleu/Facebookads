<script setup lang="ts">
import {
  ArrowLeft, Wallet, Bot, Sparkles, AlertCircle, CheckCircle2, AlertTriangle,
  TrendingUp, MessageCircle, Activity, Target, Loader2, RefreshCw,
} from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import Sparkline from '~/components/charts/Sparkline.vue'
import BarChart from '~/components/charts/BarChart.vue'

interface Account {
  meta_id: string
  name: string
  currency: string
  access_kind: string
  account_status: number
  balance: number
  amount_spent: number
  spend_cap: number
  bm_name: string | null
}

interface Kpis {
  spend_7d: number
  impressions_7d: number
  clicks_7d: number
  leads_7d: number
  avg_cpl_7d: number
  avg_ctr_7d: number
  active_campaigns: number
  paused_campaigns: number
}

interface CampaignRow {
  id: string
  meta_campaign_id: string
  name: string
  status: string
  objective: string
  daily_budget: number
  health_status: string
  spend_7d: number
  impressions_7d: number
  clicks_7d: number
  leads_7d: number
  ctr_7d: number
  cpl_7d: number
  avg_frequency_7d: number
  last_insight_date?: string
}

interface DailyPoint {
  date: string
  spend: number
  impressions: number
  clicks: number
  leads: number
  cpl: number
}

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
const accountId = computed(() => String(route.params.id))

const account = ref<Account | null>(null)
const kpis = ref<Kpis | null>(null)
const campaigns = ref<CampaignRow[]>([])
const daily = ref<DailyPoint[]>([])
const analysis = ref<Analysis | null>(null)
const analysisLoading = ref(false)
const loading = ref(true)
const errorMsg = ref<string | null>(null)
const analysisError = ref<string | null>(null)

async function load() {
  loading.value = true
  errorMsg.value = null
  try {
    const [detail, list, dly, an] = await Promise.all([
      api.get<{ data: { account: Account; kpis: Kpis } }>(`/contas/${accountId.value}`),
      api.get<{ data: CampaignRow[] }>(`/contas/${accountId.value}/campanhas`),
      api.get<{ data: DailyPoint[] }>(`/contas/${accountId.value}/insights/daily?days=14`),
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
  } catch (e: any) {
    errorMsg.value = e?.data?.error?.message || e?.message || 'Não foi possível carregar.'
  } finally {
    loading.value = false
  }
}

async function runAnalysis() {
  analysisLoading.value = true
  analysisError.value = null
  try {
    const res = await api.post<{ data: Analysis }>(`/contas/${accountId.value}/analyze`)
    analysis.value = res.data
  } catch (e: any) {
    analysisError.value = e?.data?.error?.message || e?.message || 'Falha ao analisar.'
  } finally {
    analysisLoading.value = false
  }
}

onMounted(async () => {
  await load()
  // Auto-rodar análise se ainda não tem nenhuma cacheada.
  if (!analysis.value && !errorMsg.value) {
    runAnalysis()
  }
})

const brl = (v: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v || 0)
const num = (v: number) =>
  new Intl.NumberFormat('pt-BR').format(v || 0)
const pct = (v: number) => `${((v || 0) * 100).toFixed(2)}%`

const spendSeries  = computed(() => daily.value.map((d) => d.spend))
const leadsSeries  = computed(() => daily.value.map((d) => d.leads))
const cplSeries    = computed(() => daily.value.map((d) => d.cpl))
const ctrSeries    = computed(() => daily.value.map((d) => d.impressions ? d.clicks / d.impressions : 0))
const dayLabels    = computed(() => daily.value.map((d) => {
  const [, m, dd] = d.date.split('-')
  return `${dd}/${m}`
}))

const totalCampaignSpend = computed(() => campaigns.value.reduce((s, c) => s + c.spend_7d, 0))

function statusBadge(status: string): 'success' | 'warning' | 'neutral' | 'danger' {
  return status === 'ACTIVE' ? 'success' : status === 'PAUSED' ? 'neutral' : status === 'DELETED' ? 'danger' : 'warning'
}
function statusLabel(status: string) {
  return ({ ACTIVE: 'Ativa', PAUSED: 'Pausada', ARCHIVED: 'Arquivada', DELETED: 'Removida' } as Record<string, string>)[status] || status
}
function objectiveLabel(o: string) {
  return ({
    OUTCOME_LEADS: 'Captar contatos', OUTCOME_TRAFFIC: 'Tráfego',
    OUTCOME_AWARENESS: 'Reconhecimento', OUTCOME_ENGAGEMENT: 'Engajamento',
    OUTCOME_SALES: 'Vendas', OUTCOME_APP_PROMOTION: 'App',
    LEAD_GENERATION: 'Captar contatos', CONVERSIONS: 'Conversões', LINK_CLICKS: 'Cliques',
  } as Record<string, string>)[o] || o
}
function freqColor(f: number) {
  if (f >= 3.5) return 'text-danger'
  if (f >= 2.5) return 'text-warning'
  return 'text-ink-muted'
}
function ctrColor(c: number) {
  if (c < 0.01) return 'text-danger'
  if (c < 0.02) return 'text-warning'
  return 'text-ink'
}
function highlightStyle(kind: string) {
  return {
    good: { icon: CheckCircle2, badge: 'bg-success-soft text-success' },
    warn: { icon: AlertTriangle, badge: 'bg-warning-soft text-warning' },
    bad:  { icon: AlertCircle,    badge: 'bg-danger-soft text-danger' },
  }[kind] || { icon: AlertCircle, badge: 'bg-bg-muted text-ink-muted' }
}
function priorityBadge(p: string): 'danger' | 'warning' | 'neutral' {
  return p === 'high' ? 'danger' : p === 'medium' ? 'warning' : 'neutral'
}
function priorityLabel(p: string) {
  return ({ high: 'Urgente', medium: 'Importante', low: 'Quando der' } as Record<string, string>)[p] || p
}

const trendDelta = computed(() => {
  if (daily.value.length < 8) return null
  const first7  = daily.value.slice(0, 7).reduce((s, d) => s + d.spend, 0)
  const last7   = daily.value.slice(-7).reduce((s, d) => s + d.spend, 0)
  if (first7 === 0) return null
  return ((last7 - first7) / first7) * 100
})

const leadsDelta = computed(() => {
  if (daily.value.length < 8) return null
  const first7 = daily.value.slice(0, 7).reduce((s, d) => s + d.leads, 0)
  const last7  = daily.value.slice(-7).reduce((s, d) => s + d.leads, 0)
  if (first7 === 0) return last7 > 0 ? 100 : null
  return ((last7 - first7) / first7) * 100
})
</script>

<template>
  <div class="space-y-6">
    <NuxtLink to="/dashboard" class="inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
      <ArrowLeft class="h-4 w-4" /> Voltar para o painel
    </NuxtLink>

    <div v-if="loading" class="space-y-4">
      <UiSkeleton class="h-24" />
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiSkeleton v-for="i in 4" :key="i" class="h-32" />
      </div>
      <UiSkeleton class="h-64" />
    </div>

    <div v-else-if="errorMsg" class="rounded-lg bg-danger-soft p-4 text-sm text-danger">
      {{ errorMsg }}
    </div>

    <template v-else-if="account && kpis">
      <!-- Header -->
      <UiCard>
        <div class="flex flex-wrap items-start justify-between gap-6">
          <div class="flex items-start gap-4">
            <div class="rounded-xl bg-accent-soft p-3 text-accent">
              <Wallet class="h-7 w-7" />
            </div>
            <div>
              <p class="text-xs uppercase tracking-wider text-ink-faint">{{ account.bm_name || 'Conta pessoal' }}</p>
              <h1 class="mt-0.5 text-2xl font-semibold tracking-tight text-ink">{{ account.name }}</h1>
              <p class="mt-1 text-xs text-ink-muted font-mono">{{ account.meta_id }}</p>
              <div class="mt-2 flex flex-wrap gap-2">
                <UiBadge v-if="account.account_status === 1" variant="success">Ativa</UiBadge>
                <UiBadge v-else variant="warning">Status {{ account.account_status }}</UiBadge>
                <UiBadge variant="neutral">{{ account.access_kind === 'owned' ? 'Da empresa' : account.access_kind === 'client' ? 'Compartilhada' : 'Pessoal' }}</UiBadge>
                <UiBadge variant="neutral">{{ account.currency }}</UiBadge>
              </div>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-x-8 gap-y-3">
            <div>
              <p class="text-xs uppercase tracking-wider text-ink-faint">Saldo disponível</p>
              <p class="mt-1 text-2xl font-semibold tabular-nums text-ink">{{ brl(account.balance) }}</p>
            </div>
            <div>
              <p class="text-xs uppercase tracking-wider text-ink-faint">Total já investido</p>
              <p class="mt-1 text-2xl font-semibold tabular-nums text-ink">{{ brl(account.amount_spent) }}</p>
            </div>
            <div v-if="account.spend_cap > 0" class="col-span-2">
              <p class="text-xs uppercase tracking-wider text-ink-faint">Teto de gasto</p>
              <p class="mt-1 text-sm tabular-nums text-ink-muted">{{ brl(account.spend_cap) }}</p>
            </div>
          </div>
        </div>
      </UiCard>

      <!-- KPIs com sparklines -->
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs uppercase tracking-wider text-ink-faint">Gasto 7 dias</p>
            <TrendingUp class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">{{ brl(kpis.spend_7d) }}</p>
          <p v-if="trendDelta !== null" :class="['mt-1 text-xs tabular-nums', trendDelta >= 0 ? 'text-success' : 'text-danger']">
            {{ trendDelta >= 0 ? '+' : '' }}{{ trendDelta.toFixed(0) }}% vs 7d anteriores
          </p>
          <div class="mt-3"><Sparkline :values="spendSeries" :height="40" color="#1877F2" /></div>
        </UiCard>

        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs uppercase tracking-wider text-ink-faint">Contatos / leads</p>
            <MessageCircle class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">{{ num(kpis.leads_7d) }}</p>
          <p v-if="leadsDelta !== null" :class="['mt-1 text-xs tabular-nums', leadsDelta >= 0 ? 'text-success' : 'text-danger']">
            {{ leadsDelta >= 0 ? '+' : '' }}{{ leadsDelta.toFixed(0) }}% vs 7d anteriores
          </p>
          <div class="mt-3"><Sparkline :values="leadsSeries" :height="40" color="#42B72A" /></div>
        </UiCard>

        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs uppercase tracking-wider text-ink-faint">Custo por contato</p>
            <Target class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">
            {{ kpis.leads_7d ? brl(kpis.avg_cpl_7d) : '—' }}
          </p>
          <p class="mt-1 text-xs text-ink-muted">média dos 7 dias</p>
          <div class="mt-3"><Sparkline :values="cplSeries" :height="40" color="#F7B928" /></div>
        </UiCard>

        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs uppercase tracking-wider text-ink-faint">Taxa de clique (CTR)</p>
            <Activity class="h-4 w-4 text-ink-faint" />
          </div>
          <p :class="['mt-2 text-2xl font-semibold tabular-nums', ctrColor(kpis.avg_ctr_7d)]">
            {{ pct(kpis.avg_ctr_7d) }}
          </p>
          <p class="mt-1 text-xs text-ink-muted">{{ num(kpis.clicks_7d) }} cliques / {{ num(kpis.impressions_7d) }} pessoas</p>
          <div class="mt-3"><Sparkline :values="ctrSeries" :height="40" color="#4267B2" /></div>
        </UiCard>
      </div>

      <!-- Análise IA -->
      <UiCard>
        <div class="flex items-start justify-between gap-3">
          <div class="flex items-start gap-3">
            <div class="rounded-full bg-accent-soft p-2 text-accent">
              <Sparkles class="h-5 w-5" />
            </div>
            <div>
              <h2 class="text-lg font-semibold text-ink">Diagnóstico da IA</h2>
              <p class="text-sm text-ink-muted">
                Análise da conta pelos últimos 14 dias. Em linguagem direta.
                <span v-if="analysis?.created_at" class="ml-1 text-ink-faint">
                  · gerada em {{ new Date(analysis.created_at).toLocaleString('pt-BR') }}
                </span>
              </p>
            </div>
          </div>
          <UiButton variant="ghost" size="sm" :loading="analysisLoading" @click="runAnalysis">
            <RefreshCw v-if="!analysisLoading" class="h-4 w-4" />
            {{ analysis ? 'Atualizar' : 'Analisar agora' }}
          </UiButton>
        </div>

        <div v-if="analysisLoading && !analysis" class="mt-6 flex items-center gap-3 text-sm text-ink-muted">
          <Loader2 class="h-5 w-5 animate-spin text-accent" />
          DeepSeek está analisando...
        </div>

        <div v-else-if="analysisError" class="mt-4 rounded-lg bg-danger-soft p-3 text-sm text-danger">
          {{ analysisError }}
          <span v-if="analysisError.includes('api key')">
            — vá em <NuxtLink to="/ajustes/api-keys" class="underline">Ajustes → Chaves de IA</NuxtLink>
          </span>
        </div>

        <div v-else-if="analysis" class="mt-4 space-y-5">
          <p class="text-sm leading-relaxed text-ink">{{ analysis.summary }}</p>

          <div v-if="analysis.highlights?.length" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <div
              v-for="(h, i) in analysis.highlights"
              :key="i"
              class="rounded-lg border border-border p-3"
            >
              <div class="flex items-start gap-2">
                <component
                  :is="highlightStyle(h.kind).icon"
                  class="mt-0.5 h-4 w-4 shrink-0"
                  :class="highlightStyle(h.kind).badge.split(' ')[1]"
                />
                <div class="min-w-0">
                  <p class="text-sm font-medium text-ink">{{ h.title }}</p>
                  <p class="mt-0.5 text-xs text-ink-muted">{{ h.detail }}</p>
                </div>
              </div>
            </div>
          </div>

          <div v-if="analysis.next_actions?.length" class="border-t border-border pt-4">
            <p class="text-sm font-medium text-ink">Próximos passos sugeridos</p>
            <ul class="mt-2 space-y-2">
              <li
                v-for="(a, i) in analysis.next_actions"
                :key="i"
                class="flex items-start gap-3 text-sm"
              >
                <UiBadge :variant="priorityBadge(a.priority)">{{ priorityLabel(a.priority) }}</UiBadge>
                <span class="text-ink">{{ a.action }}</span>
              </li>
            </ul>
          </div>
        </div>

        <div v-else class="mt-4 text-sm text-ink-muted">
          Nenhuma análise ainda. Clique em <strong class="text-ink">Analisar agora</strong>.
        </div>
      </UiCard>

      <!-- Distribuição de gasto por dia + concentração -->
      <div class="grid gap-4 lg:grid-cols-3">
        <UiCard class="lg:col-span-2">
          <h2 class="text-lg font-semibold text-ink">Investimento dia a dia</h2>
          <p class="text-sm text-ink-muted">Últimos 14 dias</p>
          <div class="mt-4">
            <BarChart :values="spendSeries" :labels="dayLabels" :height="160" color="#1877F2" />
          </div>
        </UiCard>

        <UiCard>
          <h2 class="text-lg font-semibold text-ink">Onde a verba foi</h2>
          <p class="text-sm text-ink-muted">Distribuição entre campanhas (7d)</p>
          <div v-if="!campaigns.length || totalCampaignSpend === 0" class="mt-6 text-sm text-ink-muted">
            Sem gasto na semana.
          </div>
          <ul v-else class="mt-4 space-y-3">
            <li v-for="c in campaigns.slice(0, 5)" :key="c.id">
              <div class="flex items-center justify-between text-xs">
                <span class="truncate text-ink">{{ c.name }}</span>
                <span class="ml-2 tabular-nums text-ink-muted">{{ ((c.spend_7d / totalCampaignSpend) * 100).toFixed(0) }}%</span>
              </div>
              <div class="mt-1 h-1.5 w-full rounded-full bg-bg-muted">
                <div
                  class="h-full rounded-full bg-accent"
                  :style="{ width: `${(c.spend_7d / totalCampaignSpend) * 100}%` }"
                />
              </div>
            </li>
          </ul>
        </UiCard>
      </div>

      <!-- Tabela de campanhas -->
      <UiCard>
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-ink">Campanhas</h2>
          <p class="text-xs text-ink-muted">{{ campaigns.length }} no total · {{ kpis.active_campaigns }} ativas</p>
        </div>

        <div v-if="!campaigns.length" class="mt-6 text-sm text-ink-muted">
          Nenhuma campanha sincronizada ainda. Próxima atualização em até 30min.
        </div>

        <div v-else class="mt-4 overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border text-xs uppercase tracking-wide text-ink-muted">
                <th class="py-2 pr-4 text-left font-medium">Campanha</th>
                <th class="py-2 pr-4 text-left font-medium">Objetivo</th>
                <th class="py-2 pr-4 text-left font-medium">Status</th>
                <th class="py-2 pr-4 text-right font-medium">Gasto 7d</th>
                <th class="py-2 pr-4 text-right font-medium">Contatos</th>
                <th class="py-2 pr-4 text-right font-medium">Custo/contato</th>
                <th class="py-2 pr-4 text-right font-medium">CTR</th>
                <th class="py-2 text-right font-medium">Cansaço</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="c in campaigns"
                :key="c.id"
                class="border-b border-border last:border-b-0 hover:bg-bg-muted"
              >
                <td class="py-3 pr-4">
                  <div class="flex items-center gap-2">
                    <span
                      class="inline-block h-2 w-2 shrink-0 rounded-full"
                      :class="{
                        'bg-success':  c.health_status === 'HEALTHY',
                        'bg-warning':  c.health_status === 'AT_RISK',
                        'bg-danger':   c.health_status === 'CRITICAL',
                        'bg-ink-faint': !['HEALTHY','AT_RISK','CRITICAL'].includes(c.health_status),
                      }"
                    />
                    <span class="font-medium text-ink">{{ c.name }}</span>
                  </div>
                </td>
                <td class="py-3 pr-4 text-ink-muted">{{ objectiveLabel(c.objective) }}</td>
                <td class="py-3 pr-4">
                  <UiBadge :variant="statusBadge(c.status)">{{ statusLabel(c.status) }}</UiBadge>
                </td>
                <td class="py-3 pr-4 text-right tabular-nums text-ink">{{ brl(c.spend_7d) }}</td>
                <td class="py-3 pr-4 text-right tabular-nums text-ink">{{ num(c.leads_7d) }}</td>
                <td class="py-3 pr-4 text-right tabular-nums text-ink">
                  {{ c.leads_7d ? brl(c.cpl_7d) : '—' }}
                </td>
                <td :class="['py-3 pr-4 text-right tabular-nums', ctrColor(c.ctr_7d)]">{{ pct(c.ctr_7d) }}</td>
                <td :class="['py-3 text-right tabular-nums', freqColor(c.avg_frequency_7d)]">
                  {{ c.avg_frequency_7d ? c.avg_frequency_7d.toFixed(2) : '—' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </UiCard>
    </template>
  </div>
</template>
