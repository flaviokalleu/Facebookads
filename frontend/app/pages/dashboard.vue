<script setup lang="ts">
import { useDashboardStore } from '~/stores/useDashboardStore'
import { useDateRange } from '~/composables/useDateRange'
import { DollarSign, Target, MousePointerClick, TrendingUp, Bot, Sparkles, Plus, BarChart3, FileText } from 'lucide-vue-next'
import { ref } from 'vue'

const store = useDashboardStore()
const { preset, label, DATE_PRESETS } = useDateRange('last_7d')

onMounted(() => store.fetchAll())
watch(preset, () => store.fetchAll())

const autoPilotOn = ref(false)

function fmtCurrency(n: number) {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 }).format(n)
}

const spendLabels = computed(() =>
  (store.summary?.daily_spend ?? []).map((d: any) => d.date?.slice(5) ?? '')
)
const spendValues = computed(() =>
  (store.summary?.daily_spend ?? []).map((d: any) => d.spend ?? 0)
)
const leadsValues = computed(() =>
  (store.summary?.daily_spend ?? []).map((d: any) => d.leads ?? 0)
)

const isStale = computed(() => store.summary?.is_stale ?? false)
</script>

<template>
  <div class="space-y-6">
    <!-- Header + Auto-Pilot -->
    <div class="flex items-center justify-between flex-wrap gap-4">
      <div>
        <h1 class="text-primary text-2xl font-bold">Visão Geral</h1>
        <p class="text-muted text-sm">Suas campanhas em tempo real</p>
      </div>
      <div class="flex items-center gap-3">
        <select v-model="preset" class="bg-bg-elevated/50 border border-bg-border text-secondary text-sm rounded-xl px-3 py-2.5 focus:outline-none focus:border-blue-default">
          <option v-for="p in DATE_PRESETS" :key="p.value" :value="p.value">{{ p.label }}</option>
        </select>
        <!-- Auto-Pilot Toggle -->
        <button
          class="flex items-center gap-2 px-5 py-2.5 rounded-xl font-semibold text-sm transition-all border-2"
          :class="autoPilotOn
            ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40 shadow-lg shadow-emerald-500/10'
            : 'bg-bg-elevated/50 text-muted border-bg-border hover:text-secondary'"
          @click="autoPilotOn = !autoPilotOn"
        >
          <Bot class="w-5 h-5" :class="{ 'animate-pulse': autoPilotOn }" />
          <span>{{ autoPilotOn ? 'Auto-Pilot Ativado' : 'Ativar Auto-Pilot' }}</span>
        </button>
      </div>
    </div>

    <!-- KPI Cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
      <KpiCard title="Gasto Total" :value="fmtCurrency(store.summary?.total_spend ?? 0)" :delta="store.summary?.spend_delta" delta-label="vs período" :icon="DollarSign" :loading="store.loading" />
      <KpiCard title="Leads" :value="(store.summary?.total_leads ?? 0).toLocaleString()" :delta="store.summary?.leads_delta" delta-label="vs período" :icon="Target" :loading="store.loading" />
      <KpiCard title="CTR Médio" :value="`${((store.summary?.avg_ctr ?? 0) * 100).toFixed(2)}%`" :delta="store.summary?.ctr_delta" delta-label="vs período" :icon="MousePointerClick" :loading="store.loading" />
      <KpiCard title="ROAS" :value="`${(store.summary?.avg_roas ?? 0).toFixed(2)}x`" :delta="store.summary?.roas_delta" delta-label="vs período" :icon="TrendingUp" :loading="store.loading" />
    </div>

    <!-- Charts -->
    <div class="grid md:grid-cols-3 gap-4">
      <div class="card md:col-span-2 border-bg-border/50 shadow-lg shadow-black/10">
        <h2 class="text-primary text-sm font-semibold mb-4">Gastos &amp; Leads</h2>
        <template v-if="store.loading"><div class="skeleton h-56 w-full rounded-lg" /></template>
        <SpendLeadsChart v-else :labels="spendLabels" :spend="spendValues" :leads="leadsValues" />
      </div>
      <div class="card border-bg-border/50 shadow-lg shadow-black/10">
        <h2 class="text-primary text-sm font-semibold mb-4">Saúde das Campanhas</h2>
        <template v-if="store.loading"><div class="skeleton h-52 w-full rounded-lg" /></template>
        <CampaignHealthDonut v-else :scaling="store.scalingCampaigns" :healthy="store.campaigns.filter(c => c.health_status === 'HEALTHY').length" :at-risk="store.campaigns.filter(c => c.health_status === 'AT_RISK').length" :underperforming="store.underperformingCampaigns" />
      </div>
    </div>

    <!-- Anomalias + Orçamento -->
    <div class="grid md:grid-cols-2 gap-4">
      <div class="card border-bg-border/50 shadow-lg shadow-black/10">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-primary text-sm font-semibold">Problemas Detectados</h2>
          <span v-if="store.highSeverityCount > 0" class="text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded-full px-2 py-0.5">{{ store.highSeverityCount }} crítico</span>
          <NuxtLink to="/anomalies" class="text-xs text-blue-bright hover:underline">Ver tudo</NuxtLink>
        </div>
        <template v-if="store.loading"><SkeletonCard v-for="i in 3" :key="i" /></template>
        <template v-else-if="store.activeAnomalies.length === 0">
          <div class="flex flex-col items-center justify-center py-8 text-center">
            <div class="w-12 h-12 rounded-full bg-emerald-500/10 flex items-center justify-center mb-3">
              <svg class="w-6 h-6 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
            </div>
            <p class="text-primary font-medium mb-1">Nada detectado</p>
            <p class="text-muted text-sm">Tudo funcionando normalmente</p>
          </div>
        </template>
        <div v-else class="space-y-2"><AnomalyCard v-for="a in store.activeAnomalies.slice(0, 5)" :key="a.id" :anomaly="a" /></div>
      </div>

      <div class="card border-bg-border/50 shadow-lg shadow-black/10">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-primary text-sm font-semibold">Sugestões de Orçamento</h2>
          <NuxtLink to="/budget" class="text-xs text-blue-bright hover:underline">Ver tudo</NuxtLink>
        </div>
        <template v-if="store.loading"><SkeletonCard v-for="i in 3" :key="i" /></template>
        <template v-else-if="store.budgetSuggestions.length === 0">
          <div class="flex flex-col items-center justify-center py-8 text-center">
            <div class="w-12 h-12 rounded-full bg-blue-500/10 flex items-center justify-center mb-3"><DollarSign class="w-6 h-6 text-blue-glow" /></div>
            <p class="text-primary font-medium mb-1">Sem sugestões</p>
            <p class="text-muted text-sm">Orçamentos estão otimizados</p>
          </div>
        </template>
        <div v-else class="space-y-2">
          <div v-for="s in store.budgetSuggestions.slice(0, 5)" :key="s.id" class="bg-bg-elevated/50 rounded-xl p-3 border border-bg-border/30">
            <div class="flex items-start justify-between gap-2">
              <p class="text-primary text-xs font-medium truncate">{{ s.campaign_name }}</p>
              <span class="text-xs font-bold shrink-0" :class="(s.suggested_change ?? 0) > 0 ? 'text-emerald-400' : 'text-red-400'">
                {{ (s.suggested_change ?? 0) > 0 ? '+' : '' }}{{ (s.suggested_change ?? 0).toFixed(0) }}%
              </span>
            </div>
            <p class="text-muted text-xs mt-1">{{ s.change_reason || s.rationale }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Relatório Semanal -->
    <div class="card border-bg-border/50 shadow-lg shadow-black/10">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-primary text-sm font-semibold flex items-center gap-2">
          <FileText class="w-4 h-4 text-blue-glow" />
          Relatório de Otimizações
        </h2>
        <span class="text-muted text-2xs">Últimos 7 dias</span>
      </div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div class="bg-bg-elevated/30 rounded-lg p-3 text-center">
          <p class="text-2xl font-bold text-emerald-400 font-mono">{{ store.anomalies.length }}</p>
          <p class="text-muted text-2xs">Problemas detectados</p>
        </div>
        <div class="bg-bg-elevated/30 rounded-lg p-3 text-center">
          <p class="text-2xl font-bold text-blue-bright font-mono">{{ (store.summary?.total_spend ?? 0) > 0 ? '✓' : '—' }}</p>
          <p class="text-muted text-2xs">Monitoramento ativo</p>
        </div>
        <div class="bg-bg-elevated/30 rounded-lg p-3 text-center">
          <p class="text-2xl font-bold text-purple-400 font-mono">{{ store.campaigns.length }}</p>
          <p class="text-muted text-2xs">Campanhas</p>
        </div>
        <div class="bg-bg-elevated/30 rounded-lg p-3 text-center">
          <p class="text-2xl font-bold text-amber-400 font-mono">{{ store.highSeverityCount }}</p>
          <p class="text-muted text-2xs">Alertas críticos</p>
        </div>
      </div>
      <p class="text-muted text-2xs mt-3 text-center">Auto-Pilot monitora suas campanhas a cada 6 horas</p>
    </div>

    <!-- Preview: Criar Campanha -->
    <NuxtLink to="/campaigns" class="fixed bottom-20 lg:bottom-6 right-6 z-40 flex items-center gap-2 bg-gradient-to-r from-blue-default to-blue-bright hover:from-blue-bright hover:to-blue-glow text-white font-semibold px-5 py-3 rounded-xl shadow-xl shadow-blue-default/30 transition-all">
      <Plus class="w-5 h-5" />
      Nova Campanha
    </NuxtLink>
  </div>
</template>
