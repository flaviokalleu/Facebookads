<script setup lang="ts">
import {
  ArrowLeft, ExternalLink, Calendar, Target, MessageCircle, Activity,
  TrendingUp, TrendingDown, Minus, Layers, Megaphone, Sparkles,
  CheckCircle2, AlertTriangle, AlertCircle, Wallet,
} from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import UiButton from '~/components/ui/UiButton.vue'
import DateRangePicker from '~/components/ui/DateRangePicker.vue'
import ComboChart from '~/components/charts/ComboChart.vue'

interface Campaign {
  id: string; meta_campaign_id: string; name: string; status: string; objective: string
  daily_budget: number; lifetime_budget: number; health_status: string
  account_meta_id: string; account_name: string; bm_name: string
  meta_created_time?: string; meta_start_time?: string; meta_stop_time?: string
  first_insight_date?: string; last_insight_date?: string
  days_running: number
}
interface Kpis {
  spend: number; impressions: number; clicks: number; leads: number
  ctr: number; cpl: number; avg_frequency: number
  spend_prev: number; leads_prev: number; cpl_prev: number
}
interface DailyPoint { date: string; spend: number; leads: number; impressions: number; clicks: number; frequency: number }
interface AdSet {
  id: string; meta_adset_id: string; name: string; status: string
  daily_budget: number; optimization_goal: string; billing_event: string
  meta_start_time?: string; meta_end_time?: string
  ads_count: number; active_ads_count: number
}
interface Ad {
  id: string; meta_ad_id: string; ad_set_id: string; adset_name: string
  name: string; status: string
  creative_title: string; creative_body: string; image_url: string; cta_type: string
}
interface AIAction {
  id: string; action_type: string; target_kind: string; reason: string
  status: string; source: string; mode: string; created_at: string
}

const route = useRoute()
const api = useApi()
const days = ref(14)

const campaign = ref<Campaign | null>(null)
const kpis = ref<Kpis | null>(null)
const daily = ref<DailyPoint[]>([])
const adsets = ref<AdSet[]>([])
const ads = ref<Ad[]>([])
const aiActions = ref<AIAction[]>([])
const loading = ref(true)
const errorMsg = ref<string | null>(null)

async function load() {
  loading.value = true
  errorMsg.value = null
  try {
    const res = await api.get<{ data: any }>(`/campanhas/${route.params.id}?days=${days.value}`)
    const d = res.data
    campaign.value = d.campaign
    kpis.value = d.kpis
    daily.value = d.daily || []
    adsets.value = d.adsets || []
    ads.value = d.ads || []
    aiActions.value = d.ai_actions || []
  } catch (e: any) {
    errorMsg.value = e?.data?.error?.message || e?.message || 'Não foi possível carregar.'
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch(days, load)

const brl = (v: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v || 0)
const num = (v: number) => new Intl.NumberFormat('pt-BR').format(v || 0)
const pct = (v: number) => `${((v || 0) * 100).toFixed(2)}%`

function deltaPct(now: number, prev: number): number | null {
  if (prev === 0) return now > 0 ? 100 : null
  return ((now - prev) / prev) * 100
}
const spendDelta = computed(() => kpis.value ? deltaPct(kpis.value.spend, kpis.value.spend_prev) : null)
const leadsDelta = computed(() => kpis.value ? deltaPct(kpis.value.leads, kpis.value.leads_prev) : null)
const cplDelta   = computed(() => kpis.value && kpis.value.cpl_prev > 0 ? deltaPct(kpis.value.cpl, kpis.value.cpl_prev) : null)

const dailyLabels = computed(() => daily.value.map((d) => {
  const [, m, dd] = d.date.split('-'); return `${dd}/${m}`
}))
const spendSeries = computed(() => daily.value.map((d) => d.spend))
const leadsSeries = computed(() => daily.value.map((d) => d.leads))

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
function optGoalLabel(g: string) {
  return ({
    CONVERSATIONS: 'Conversas no WhatsApp',
    LEAD_GENERATION: 'Geração de leads',
    OFFSITE_CONVERSIONS: 'Conversões no site',
    LINK_CLICKS: 'Cliques no link',
    LANDING_PAGE_VIEWS: 'Visualizações de página',
    IMPRESSIONS: 'Impressões',
    REACH: 'Alcance',
    THRUPLAY: 'Vídeo até o fim',
    QUALITY_LEAD: 'Lead qualificado',
  } as Record<string, string>)[g] || g
}

function metaManagerLink() {
  if (!campaign.value) return '#'
  const acct = campaign.value.account_meta_id.replace(/^act_/, '')
  return `https://adsmanager.facebook.com/adsmanager/manage/ads?act=${acct}&selected_campaign_ids=${campaign.value.meta_campaign_id}`
}

const phaseHint = computed(() => {
  if (!campaign.value) return null
  const d = campaign.value.days_running
  if (d < 7) return { tone: 'warning', text: `Em fase de aprendizagem — ${7 - d} ${7 - d === 1 ? 'dia restante' : 'dias restantes'}. A IA não vai mexer ainda.` }
  if (kpis.value && kpis.value.leads < 50) return { tone: 'warning', text: `Tem ${kpis.value.leads} contatos em ${days.value}d. Meta exige ~50 conversões/7d pra otimização ficar estável.` }
  return null
})

function aiActionLabel(t: string) {
  return ({
    pause_ad: 'Pausa de anúncio',
    pause_adset: 'Pausa de conjunto',
    scale_budget: 'Ajuste de verba',
    rotate_creative: 'Rotação de criativo',
    duplicate_adset: 'Duplicação de conjunto',
    create_campaign: 'Nova campanha',
    alert: 'Alerta',
  } as Record<string, string>)[t] || t
}

function formatTime(s?: string) {
  if (!s) return '—'
  return new Date(s).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function adsByAdSet(adsetID: string) {
  return ads.value.filter((a) => a.ad_set_id === adsetID)
}

function ctrColor(v: number) { return v < 0.01 ? 'text-danger' : v < 0.02 ? 'text-warning' : 'text-ink' }
function freqColor(v: number) { return v >= 3.5 ? 'text-danger' : v >= 2.5 ? 'text-warning' : 'text-ink-muted' }
</script>

<template>
  <div class="space-y-5">
    <NuxtLink to="/campanhas" class="inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
      <ArrowLeft class="h-4 w-4" /> Voltar para anúncios
    </NuxtLink>

    <div v-if="loading" class="space-y-4">
      <UiSkeleton class="h-32" />
      <div class="grid gap-4 sm:grid-cols-4">
        <UiSkeleton v-for="i in 4" :key="i" class="h-24" />
      </div>
      <UiSkeleton class="h-64" />
    </div>

    <div v-else-if="errorMsg || !campaign" class="rounded-xl bg-danger-soft p-4 text-sm text-danger">
      {{ errorMsg || 'Campanha não encontrada.' }}
    </div>

    <template v-else>
      <!-- Header -->
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p class="text-xs uppercase tracking-wider text-ink-faint">
            <NuxtLink :to="`/contas/${campaign.account_meta_id}`" class="hover:text-ink hover:underline">
              {{ campaign.account_name }}
            </NuxtLink>
            <span v-if="campaign.bm_name"> · {{ campaign.bm_name }}</span>
          </p>
          <h1 class="mt-1 text-2xl font-semibold tracking-tight text-ink">{{ campaign.name }}</h1>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <UiBadge :variant="statusBadge(campaign.status)">{{ statusLabel(campaign.status) }}</UiBadge>
            <UiBadge variant="neutral">{{ objectiveLabel(campaign.objective) }}</UiBadge>
            <span class="inline-flex items-center gap-1 text-xs text-ink-muted">
              <Calendar class="h-3 w-3" />
              {{ campaign.days_running }} {{ campaign.days_running === 1 ? 'dia rodando' : 'dias rodando' }}
              <span v-if="campaign.meta_start_time">
                (desde {{ new Date(campaign.meta_start_time).toLocaleDateString('pt-BR') }})
              </span>
            </span>
            <span v-if="campaign.meta_stop_time" class="inline-flex items-center gap-1 text-xs text-warning">
              · termina em {{ new Date(campaign.meta_stop_time).toLocaleDateString('pt-BR') }}
            </span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <DateRangePicker v-model="days" />
          <a :href="metaManagerLink()" target="_blank" rel="noopener">
            <UiButton variant="ghost" size="sm">
              <ExternalLink class="h-4 w-4" /> Abrir no Meta
            </UiButton>
          </a>
        </div>
      </div>

      <div v-if="phaseHint" class="flex items-start gap-2 rounded-lg bg-warning-soft px-3 py-2 text-xs text-warning">
        <AlertTriangle class="h-4 w-4 shrink-0" />
        <span>{{ phaseHint.text }}</span>
      </div>

      <!-- KPIs -->
      <div v-if="kpis" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">Investido</p>
            <Wallet class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">{{ brl(kpis.spend) }}</p>
          <div v-if="spendDelta !== null" class="mt-1 flex items-center gap-1 text-xs text-ink-muted">
            <TrendingUp v-if="spendDelta > 0" class="h-3.5 w-3.5" />
            <TrendingDown v-else-if="spendDelta < 0" class="h-3.5 w-3.5" />
            <Minus v-else class="h-3.5 w-3.5" />
            <span>{{ spendDelta > 0 ? '+' : '' }}{{ spendDelta.toFixed(0) }}% vs anterior</span>
          </div>
        </UiCard>
        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">Contatos</p>
            <MessageCircle class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">{{ num(kpis.leads) }}</p>
          <div v-if="leadsDelta !== null" class="mt-1 flex items-center gap-1 text-xs">
            <TrendingUp v-if="leadsDelta > 0" class="h-3.5 w-3.5 text-success" />
            <TrendingDown v-else-if="leadsDelta < 0" class="h-3.5 w-3.5 text-danger" />
            <Minus v-else class="h-3.5 w-3.5 text-ink-faint" />
            <span :class="leadsDelta > 0 ? 'text-success' : leadsDelta < 0 ? 'text-danger' : 'text-ink-faint'">
              {{ leadsDelta > 0 ? '+' : '' }}{{ leadsDelta.toFixed(0) }}% vs anterior
            </span>
          </div>
        </UiCard>
        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">Custo por contato</p>
            <Target class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">
            {{ kpis.leads > 0 ? brl(kpis.cpl) : '—' }}
          </p>
          <div v-if="cplDelta !== null" class="mt-1 flex items-center gap-1 text-xs">
            <TrendingDown v-if="cplDelta < 0" class="h-3.5 w-3.5 text-success" />
            <TrendingUp v-else-if="cplDelta > 0" class="h-3.5 w-3.5 text-danger" />
            <Minus v-else class="h-3.5 w-3.5 text-ink-faint" />
            <span :class="cplDelta < 0 ? 'text-success' : cplDelta > 0 ? 'text-danger' : 'text-ink-faint'">
              {{ cplDelta > 0 ? '+' : '' }}{{ cplDelta.toFixed(0) }}% vs anterior
            </span>
          </div>
        </UiCard>
        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">CTR / Frequência</p>
            <Activity class="h-4 w-4 text-ink-faint" />
          </div>
          <p :class="['mt-2 text-2xl font-semibold tabular-nums', ctrColor(kpis.ctr)]">{{ pct(kpis.ctr) }}</p>
          <p :class="['mt-1 text-xs', freqColor(kpis.avg_frequency)]">
            Frequência {{ kpis.avg_frequency ? kpis.avg_frequency.toFixed(2) : '—' }}× por pessoa
          </p>
        </UiCard>
      </div>

      <UiCard>
        <h2 class="text-base font-semibold text-ink">Investimento × contatos por dia</h2>
        <p class="text-sm text-ink-muted">Últimos {{ Math.max(days, 7) }} dias</p>
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

      <!-- Conjuntos + anúncios -->
      <div>
        <div class="mb-3 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Layers class="h-4 w-4 text-accent" />
            <h2 class="text-base font-semibold text-ink">Conjuntos de anúncios</h2>
          </div>
          <p class="text-xs text-ink-muted">{{ adsets.length }} {{ adsets.length === 1 ? 'conjunto' : 'conjuntos' }} · {{ ads.length }} {{ ads.length === 1 ? 'anúncio' : 'anúncios' }}</p>
        </div>

        <UiCard v-if="!adsets.length" class="!p-6">
          <p class="text-sm text-ink-muted">Nenhum conjunto sincronizado ainda.</p>
        </UiCard>

        <div v-else class="space-y-3">
          <UiCard v-for="aset in adsets" :key="aset.id" class="!p-4">
            <details class="group" open>
              <summary class="flex cursor-pointer list-none items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="font-medium text-ink truncate">{{ aset.name }}</h3>
                    <UiBadge :variant="statusBadge(aset.status)">{{ statusLabel(aset.status) }}</UiBadge>
                    <UiBadge v-if="aset.optimization_goal" variant="neutral">{{ optGoalLabel(aset.optimization_goal) }}</UiBadge>
                  </div>
                  <p class="mt-1 text-xs text-ink-muted">
                    <span v-if="aset.daily_budget > 0">{{ brl(aset.daily_budget) }}/dia · </span>
                    <Megaphone class="inline h-3 w-3" /> {{ aset.active_ads_count }} no ar / {{ aset.ads_count }} {{ aset.ads_count === 1 ? 'anúncio' : 'anúncios' }}
                  </p>
                </div>
                <span class="text-ink-faint group-open:rotate-180 transition">▾</span>
              </summary>

              <div class="mt-4 space-y-2 border-t border-border pt-4">
                <div
                  v-for="ad in adsByAdSet(aset.id)"
                  :key="ad.id"
                  class="rounded-lg bg-bg-muted p-3"
                >
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0 flex-1">
                      <div class="flex flex-wrap items-center gap-2">
                        <p class="text-sm font-medium text-ink truncate">{{ ad.name }}</p>
                        <UiBadge :variant="statusBadge(ad.status)">{{ statusLabel(ad.status) }}</UiBadge>
                      </div>
                      <p v-if="ad.creative_title" class="mt-1 text-xs font-medium text-ink-muted">{{ ad.creative_title }}</p>
                      <p
                        v-if="ad.creative_body"
                        class="mt-1 text-xs text-ink-muted whitespace-pre-line"
                      >{{ ad.creative_body.length > 220 ? ad.creative_body.slice(0, 220) + '...' : ad.creative_body }}</p>
                    </div>
                    <img
                      v-if="ad.image_url"
                      :src="ad.image_url"
                      class="h-16 w-16 shrink-0 rounded-md object-cover bg-bg"
                      alt=""
                    />
                  </div>
                </div>
                <p v-if="!adsByAdSet(aset.id).length" class="text-sm text-ink-muted">
                  Nenhum anúncio sincronizado neste conjunto.
                </p>
              </div>
            </details>
          </UiCard>
        </div>
      </div>

      <!-- Histórico da IA -->
      <UiCard v-if="aiActions.length">
        <div class="flex items-center gap-2">
          <Sparkles class="h-4 w-4 text-accent" />
          <h2 class="text-base font-semibold text-ink">O que a IA fez aqui</h2>
        </div>
        <ul class="mt-4 space-y-2">
          <li
            v-for="a in aiActions"
            :key="a.id"
            class="flex items-start gap-3 rounded-lg border border-border p-3"
          >
            <CheckCircle2 v-if="a.status === 'executed'" class="mt-0.5 h-4 w-4 shrink-0 text-success" />
            <AlertTriangle v-else-if="a.status === 'pending'" class="mt-0.5 h-4 w-4 shrink-0 text-warning" />
            <AlertCircle v-else class="mt-0.5 h-4 w-4 shrink-0 text-ink-faint" />
            <div class="flex-1 min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-sm font-medium text-ink">{{ aiActionLabel(a.action_type) }}</p>
                <UiBadge variant="neutral">{{ a.source === 'deepseek' ? 'IA DeepSeek' : 'Regra automática' }}</UiBadge>
                <span class="ml-auto text-xs text-ink-faint">{{ formatTime(a.created_at) }}</span>
              </div>
              <p class="mt-1 text-sm text-ink-muted">{{ a.reason }}</p>
            </div>
          </li>
        </ul>
      </UiCard>
    </template>
  </div>
</template>
