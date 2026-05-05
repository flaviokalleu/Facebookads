<script setup lang="ts">
import {
  TrendingUp, TrendingDown, Minus, Trophy, AlertCircle, AlertTriangle,
  ChevronRight, Wallet, MessageCircle, Target, Activity, Building2,
  ChevronDown, ChevronUp,
} from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import UiButton from '~/components/ui/UiButton.vue'
import DateRangePicker from '~/components/ui/DateRangePicker.vue'
import ComboChart from '~/components/charts/ComboChart.vue'
import BMTree from '~/components/tree/BMTree.vue'
import { useMetaTree, type MetaTree } from '~/composables/useMetaTree'

interface Kpis {
  spend: number; spend_prev: number
  leads: number; leads_prev: number
  avg_cpl: number; avg_cpl_prev: number
  impressions: number; clicks: number
  accounts_total: number; accounts_active: number; bms_total: number
  active_campaigns: number
  low_balance_count: number; burning_no_leads: number
}
interface AccountRow {
  meta_id: string; name: string; bm_name: string
  balance: number; spend: number; leads: number; cpl: number
  impressions: number; clicks: number
  days_balance_left?: number
  status: string
}
interface DailyPoint { date: string; spend: number; leads: number; impressions: number; clicks: number }

const api = useApi()
const { fetchTree } = useMetaTree()

const days = ref(7)
const kpis = ref<Kpis | null>(null)
const topBySpend  = ref<AccountRow[]>([])
const bestByCPL   = ref<AccountRow[]>([])
const worstByCPL  = ref<AccountRow[]>([])
const lowBalance  = ref<AccountRow[]>([])
const burningNoLeads = ref<AccountRow[]>([])
const daily = ref<DailyPoint[]>([])
const tree = ref<MetaTree>({ businesses: [], personal_accounts: [] })
const loading = ref(true)
const treeOpen = ref(false)

async function load() {
  loading.value = true
  try {
    const [overview, t] = await Promise.all([
      api.get<{ data: any }>(`/dashboard/overview?days=${days.value}`),
      fetchTree(),
    ])
    const d = overview.data
    kpis.value         = d.kpis
    topBySpend.value   = d.top_by_spend     || []
    bestByCPL.value    = d.best_by_cpl      || []
    worstByCPL.value   = d.worst_by_cpl     || []
    lowBalance.value   = d.low_balance      || []
    burningNoLeads.value = d.burning_no_leads || []
    daily.value        = d.daily            || []
    tree.value         = t
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch(days, load)

const brl = (v: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v || 0)
const num = (v: number) => new Intl.NumberFormat('pt-BR').format(v || 0)

function deltaPct(now: number, prev: number): number | null {
  if (prev === 0) return now > 0 ? 100 : null
  return ((now - prev) / prev) * 100
}

const spendDelta = computed(() => kpis.value ? deltaPct(kpis.value.spend, kpis.value.spend_prev) : null)
const leadsDelta = computed(() => kpis.value ? deltaPct(kpis.value.leads, kpis.value.leads_prev) : null)
const cplDelta   = computed(() => kpis.value && kpis.value.avg_cpl_prev > 0 ? deltaPct(kpis.value.avg_cpl, kpis.value.avg_cpl_prev) : null)

const dailyLabels = computed(() => daily.value.map((d) => {
  const [, m, dd] = d.date.split('-'); return `${dd}/${m}`
}))
const spendSeries = computed(() => daily.value.map((d) => d.spend))
const leadsSeries = computed(() => daily.value.map((d) => d.leads))

const hasMeta = computed(() => (tree.value.businesses?.length || 0) + (tree.value.personal_accounts?.length || 0) > 0)

const hero = computed<{ tone: 'good' | 'warn' | 'bad'; light: string; bg: string; headline: string; body: string }>(() => {
  if (!kpis.value) return { tone: 'warn', light: 'bg-warning', bg: 'bg-warning-soft border-warning/20', headline: '', body: '' }
  const k = kpis.value
  if (k.spend === 0 && k.leads === 0) {
    return {
      tone: 'warn', light: 'bg-ink-faint',
      bg: 'bg-bg-muted border-border',
      headline: 'Nenhuma conta investiu nos últimos dias',
      body: 'Conecte uma campanha ativa ou verifique se o saldo das suas contas está zerado.',
    }
  }
  let tone: 'good' | 'warn' | 'bad' = 'good'
  const reasons: string[] = []
  if (k.burning_no_leads > 0) { tone = 'bad'; reasons.push(`${k.burning_no_leads} ${k.burning_no_leads === 1 ? 'conta gastando' : 'contas gastando'} sem trazer contatos`) }
  if (k.low_balance_count > 0) {
    if (tone === 'good') tone = 'warn'
    reasons.push(`${k.low_balance_count} ${k.low_balance_count === 1 ? 'conta' : 'contas'} com saldo pra menos de 3 dias`)
  }
  const headline = tone === 'good'
    ? `Suas ${k.accounts_active} contas geraram ${k.leads} contatos a R$ ${k.avg_cpl.toFixed(2)} cada nos últimos ${days.value} ${days.value === 1 ? 'dia' : 'dias'}`
    : `Atenção — ${reasons.join('; ')}`
  const body = tone === 'good'
    ? `Investimento total: ${brl(k.spend)}. ${k.active_campaigns} ${k.active_campaigns === 1 ? 'campanha rodando' : 'campanhas rodando'} em ${k.bms_total} ${k.bms_total === 1 ? 'empresa' : 'empresas'}.`
    : `Veja as contas em "Atenção urgente" abaixo e tome ação.`
  return tone === 'good'
    ? { tone, light: 'bg-success', bg: 'bg-success-soft border-success/20', headline, body }
    : tone === 'warn'
      ? { tone, light: 'bg-warning', bg: 'bg-warning-soft border-warning/20', headline, body }
      : { tone, light: 'bg-danger', bg: 'bg-danger-soft border-danger/20', headline, body }
})
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-ink">Painel geral</h1>
        <p class="text-sm text-ink-muted">Visão consolidada de todas as suas contas.</p>
      </div>
      <DateRangePicker v-model="days" />
    </div>

    <div v-if="loading" class="space-y-4">
      <UiSkeleton class="h-32" />
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiSkeleton v-for="i in 4" :key="i" class="h-28" />
      </div>
      <UiSkeleton class="h-64" />
    </div>

    <UiCard v-else-if="!hasMeta">
      <div class="text-center py-6">
        <p class="text-ink">Nenhuma empresa conectada ainda.</p>
        <p class="mt-1 text-sm text-ink-muted">Conecte sua conta Meta para começar a usar a IA.</p>
        <div class="mt-4 flex justify-center">
          <NuxtLink to="/onboarding"><UiButton variant="primary">Conectar conta</UiButton></NuxtLink>
        </div>
      </div>
    </UiCard>

    <template v-else-if="kpis">
      <!-- HERO STATUS -->
      <div :class="['rounded-2xl border p-6', hero.bg]">
        <div class="flex items-start gap-4">
          <span :class="['mt-2 inline-block h-3 w-3 shrink-0 rounded-full', hero.light]" />
          <div class="flex-1">
            <h2 class="text-lg font-semibold leading-tight text-ink">{{ hero.headline }}</h2>
            <p class="mt-1 text-sm text-ink-muted">{{ hero.body }}</p>
          </div>
        </div>
      </div>

      <!-- KPIs -->
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">Investimento total</p>
            <Wallet class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">{{ brl(kpis.spend) }}</p>
          <p class="mt-1 text-xs text-ink-muted">nos últimos {{ days }} {{ days === 1 ? 'dia' : 'dias' }}</p>
          <div v-if="spendDelta !== null" class="mt-2 flex items-center gap-1 text-xs text-ink-muted">
            <TrendingUp v-if="spendDelta > 0" class="h-3.5 w-3.5" />
            <TrendingDown v-else-if="spendDelta < 0" class="h-3.5 w-3.5" />
            <Minus v-else class="h-3.5 w-3.5" />
            <span>{{ spendDelta > 0 ? '+' : '' }}{{ spendDelta.toFixed(0) }}% vs período anterior</span>
          </div>
        </UiCard>

        <UiCard>
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">Contatos / leads</p>
            <MessageCircle class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">{{ num(kpis.leads) }}</p>
          <p class="mt-1 text-xs text-ink-muted">total entre as contas</p>
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
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">Custo médio por contato</p>
            <Target class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">
            {{ kpis.leads > 0 ? brl(kpis.avg_cpl) : '—' }}
          </p>
          <p class="mt-1 text-xs text-ink-muted">média ponderada</p>
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
          <div class="flex items-center justify-between">
            <p class="text-xs text-ink-muted">Contas em risco</p>
            <Activity class="h-4 w-4 text-ink-faint" />
          </div>
          <p class="mt-2 text-2xl font-semibold tabular-nums text-ink">
            {{ kpis.low_balance_count + kpis.burning_no_leads }}
          </p>
          <p class="mt-1 text-xs text-ink-muted">
            {{ kpis.low_balance_count }} sem saldo · {{ kpis.burning_no_leads }} sem retorno
          </p>
          <p class="mt-2 text-xs text-ink-faint">
            de {{ kpis.accounts_active }} contas ativas
          </p>
        </UiCard>
      </div>

      <!-- ATENÇÃO URGENTE -->
      <UiCard v-if="lowBalance.length || burningNoLeads.length" class="border-warning/40">
        <div class="flex items-center gap-2">
          <AlertTriangle class="h-5 w-5 text-warning" />
          <h2 class="text-base font-semibold text-ink">Atenção urgente</h2>
        </div>

        <div v-if="lowBalance.length" class="mt-4">
          <p class="mb-2 text-xs uppercase tracking-wide text-ink-faint">Saldo acabando</p>
          <ul class="space-y-1.5">
            <li v-for="acc in lowBalance.slice(0, 5)" :key="acc.meta_id">
              <NuxtLink :to="`/contas/${acc.meta_id}`" class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-bg-muted">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-ink">{{ acc.name }}</p>
                  <p class="text-xs text-ink-muted">{{ acc.bm_name || 'Pessoal' }}</p>
                </div>
                <div class="text-right">
                  <p class="text-sm tabular-nums text-danger font-medium">
                    {{ brl(acc.balance) }}
                  </p>
                  <p class="text-xs text-ink-muted">
                    dura {{ acc.days_balance_left ? `~${Math.max(0, Math.floor(acc.days_balance_left))} ${Math.floor(acc.days_balance_left) === 1 ? 'dia' : 'dias'}` : '—' }}
                  </p>
                </div>
                <ChevronRight class="ml-3 h-4 w-4 text-ink-faint" />
              </NuxtLink>
            </li>
          </ul>
        </div>

        <div v-if="burningNoLeads.length" class="mt-4">
          <p class="mb-2 text-xs uppercase tracking-wide text-ink-faint">Gastando sem trazer contatos</p>
          <ul class="space-y-1.5">
            <li v-for="acc in burningNoLeads.slice(0, 5)" :key="acc.meta_id">
              <NuxtLink :to="`/contas/${acc.meta_id}`" class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-bg-muted">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-ink">{{ acc.name }}</p>
                  <p class="text-xs text-ink-muted">{{ acc.bm_name || 'Pessoal' }}</p>
                </div>
                <div class="text-right">
                  <p class="text-sm tabular-nums text-danger font-medium">{{ brl(acc.spend) }} gasto</p>
                  <p class="text-xs text-ink-muted">{{ acc.leads }} contatos</p>
                </div>
                <ChevronRight class="ml-3 h-4 w-4 text-ink-faint" />
              </NuxtLink>
            </li>
          </ul>
        </div>
      </UiCard>

      <!-- GRÁFICO -->
      <UiCard>
        <h2 class="text-base font-semibold text-ink">Investimento × contatos por dia</h2>
        <p class="text-sm text-ink-muted">Soma de todas as contas — últimos {{ Math.max(days, 7) }} dias</p>
        <div class="mt-4">
          <ComboChart
            :bars="spendSeries"
            :line="leadsSeries"
            :labels="dailyLabels"
            :height="200"
            bar-color="#1877F2"
            line-color="#42B72A"
            bar-label="Investimento"
            line-label="Contatos"
          />
        </div>
      </UiCard>

      <!-- TOP / WORST -->
      <div class="grid gap-4 lg:grid-cols-2">
        <UiCard>
          <div class="flex items-center gap-2">
            <Trophy class="h-4 w-4 text-success" />
            <h2 class="text-base font-semibold text-ink">Melhores contas</h2>
          </div>
          <p class="mt-1 text-xs text-ink-muted">Menor custo por contato (mín. 5 contatos)</p>
          <ul v-if="bestByCPL.length" class="mt-4 space-y-1.5">
            <li v-for="(acc, i) in bestByCPL" :key="acc.meta_id">
              <NuxtLink :to="`/contas/${acc.meta_id}`" class="flex items-center gap-3 rounded-lg px-3 py-2 hover:bg-bg-muted">
                <span class="w-5 text-center text-xs font-bold text-ink-faint">{{ i + 1 }}</span>
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-ink">{{ acc.name }}</p>
                  <p class="truncate text-xs text-ink-muted">{{ acc.bm_name || 'Pessoal' }}</p>
                </div>
                <div class="text-right">
                  <p class="text-sm tabular-nums text-success font-semibold">{{ brl(acc.cpl) }}</p>
                  <p class="text-xs text-ink-muted">{{ acc.leads }} contatos</p>
                </div>
              </NuxtLink>
            </li>
          </ul>
          <p v-else class="mt-4 text-sm text-ink-muted">Nenhuma conta com volume suficiente ainda.</p>
        </UiCard>

        <UiCard>
          <div class="flex items-center gap-2">
            <AlertCircle class="h-4 w-4 text-danger" />
            <h2 class="text-base font-semibold text-ink">Mais caras</h2>
          </div>
          <p class="mt-1 text-xs text-ink-muted">Maior custo por contato (mín. 5 contatos)</p>
          <ul v-if="worstByCPL.length" class="mt-4 space-y-1.5">
            <li v-for="(acc, i) in worstByCPL" :key="acc.meta_id">
              <NuxtLink :to="`/contas/${acc.meta_id}`" class="flex items-center gap-3 rounded-lg px-3 py-2 hover:bg-bg-muted">
                <span class="w-5 text-center text-xs font-bold text-ink-faint">{{ i + 1 }}</span>
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-ink">{{ acc.name }}</p>
                  <p class="truncate text-xs text-ink-muted">{{ acc.bm_name || 'Pessoal' }}</p>
                </div>
                <div class="text-right">
                  <p class="text-sm tabular-nums text-danger font-semibold">{{ brl(acc.cpl) }}</p>
                  <p class="text-xs text-ink-muted">{{ acc.leads }} contatos · {{ brl(acc.spend) }}</p>
                </div>
              </NuxtLink>
            </li>
          </ul>
          <p v-else class="mt-4 text-sm text-ink-muted">Nada por aqui.</p>
        </UiCard>
      </div>

      <!-- TOP POR GASTO -->
      <UiCard v-if="topBySpend.length">
        <h2 class="text-base font-semibold text-ink">Onde sua verba está indo</h2>
        <p class="mt-1 text-xs text-ink-muted">Top 5 contas por investimento — últimos {{ days }} {{ days === 1 ? 'dia' : 'dias' }}</p>
        <ul class="mt-4 space-y-2">
          <li v-for="acc in topBySpend" :key="acc.meta_id">
            <NuxtLink :to="`/contas/${acc.meta_id}`" class="block rounded-lg px-3 py-2 hover:bg-bg-muted">
              <div class="flex items-center justify-between gap-3">
                <p class="truncate text-sm font-medium text-ink">{{ acc.name }}</p>
                <p class="shrink-0 text-sm tabular-nums text-ink">{{ brl(acc.spend) }}</p>
              </div>
              <div class="mt-1 flex items-center justify-between text-xs text-ink-muted">
                <span class="truncate">{{ acc.bm_name || 'Pessoal' }}</span>
                <span class="shrink-0">{{ acc.leads }} contatos · {{ acc.leads > 0 ? brl(acc.cpl) : '—' }} cada</span>
              </div>
              <div class="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-bg-muted">
                <div
                  class="h-full rounded-full bg-accent"
                  :style="{ width: `${Math.min((acc.spend / (topBySpend[0]?.spend || 1)) * 100, 100)}%` }"
                />
              </div>
            </NuxtLink>
          </li>
        </ul>
      </UiCard>

      <!-- ÁRVORE BM (colapsável) -->
      <UiCard>
        <button type="button" class="flex w-full items-center justify-between" @click="treeOpen = !treeOpen">
          <div class="flex items-center gap-2">
            <Building2 class="h-4 w-4 text-accent" />
            <span class="font-medium text-ink">Suas {{ kpis.bms_total }} empresas e {{ kpis.accounts_total }} contas</span>
          </div>
          <component :is="treeOpen ? ChevronUp : ChevronDown" class="h-4 w-4 text-ink-muted" />
        </button>
        <div v-if="treeOpen" class="mt-4">
          <BMTree :businesses="tree.businesses" :personal-accounts="tree.personal_accounts" />
        </div>
      </UiCard>
    </template>
  </div>
</template>
