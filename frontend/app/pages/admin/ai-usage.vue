<script setup lang="ts">
import { useAiUsageStore } from '~/stores/useAiUsageStore'
import { ChartNoAxesColumnIncreasing, DollarSign, Hash, Cpu, BarChart3 } from 'lucide-vue-next'

const store = useAiUsageStore()
onMounted(() => store.fetchAll())

const chartLabels = computed(() => {
  const dates = new Set<string>()
  store.dailyCost.forEach((d: any) => dates.add(d.date?.slice(5, 10) ?? ''))
  return [...dates].sort()
})

const providers = computed(() => {
  const names = new Set<string>()
  store.dailyCost.forEach((d: any) => names.add(d.provider))
  return [...names]
})

const maxCost = computed(() => {
  let max = 0
  store.dailyCost.forEach((d: any) => { if (d.total_cost > max) max = d.total_cost })
  return max || 1
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center gap-2">
      <ChartNoAxesColumnIncreasing class="w-5 h-5 text-blue-glow" />
      <div>
        <h1 class="text-primary text-xl font-bold">AI Usage</h1>
        <p class="text-muted text-sm">Cost and request analytics by provider</p>
      </div>
    </div>

    <!-- Summary KPIs -->
    <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
      <KpiCard title="Total Cost" :value="`$${store.totalCost.toFixed(4)}`" :icon="DollarSign" :loading="store.loading" />
      <KpiCard title="Total Requests" :value="store.totalRequests.toLocaleString()" :icon="Hash" :loading="store.loading" />
      <KpiCard title="Active Providers" :value="store.summary.length.toString()" :icon="Cpu" :loading="store.loading" />
    </div>

    <!-- Daily cost chart (inline bar chart) -->
    <div class="card border-bg-border/50 shadow-lg shadow-black/10">
      <h2 class="text-primary text-sm font-semibold mb-4 flex items-center gap-2">
        <BarChart3 class="w-4 h-4 text-blue-glow" />
        Daily Cost by Provider
      </h2>
      <template v-if="store.loading">
        <div class="skeleton h-52 w-full rounded-lg" />
      </template>
      <div v-else-if="chartLabels.length" class="space-y-2">
        <div v-for="date in chartLabels" :key="date" class="flex items-center gap-3">
          <span class="text-muted text-xs w-12 shrink-0">{{ date }}</span>
          <div class="flex-1 flex gap-0.5 h-6 items-end">
            <div
              v-for="prov in providers"
              :key="prov"
              class="flex-1 rounded-t transition-all duration-300"
              :style="{
                height: `${Math.max(8, (store.dailyCost.find((d: any) => d.date?.slice(5,10) === date && d.provider === prov)?.total_cost ?? 0) / maxCost * 100)}%`,
                background: prov === 'anthropic' ? '#D97706' : prov === 'openai' ? '#10A37F' : prov === 'deepseek' ? '#4F46E5' : '#3B82F6',
                opacity: 0.8
              }"
              :title="`${prov}: $${(store.dailyCost.find((d: any) => d.date?.slice(5,10) === date && d.provider === prov)?.total_cost ?? 0).toFixed(6)}`"
            />
          </div>
        </div>
      </div>
      <div v-else class="flex flex-col items-center justify-center py-12 text-center">
        <div class="w-12 h-12 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-3">
          <BarChart3 class="w-6 h-6 text-muted" />
        </div>
        <p class="text-primary font-medium">No usage data yet</p>
      </div>
    </div>

    <!-- Provider table -->
    <div class="card border-bg-border/50 shadow-lg shadow-black/10 overflow-x-auto">
      <h2 class="text-primary text-sm font-semibold mb-4 flex items-center gap-2">
        <Cpu class="w-4 h-4 text-blue-glow" />
        By Provider
      </h2>
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-muted text-xs border-b border-bg-border/50">
            <th class="pb-2 pr-4 font-medium">Provider</th>
            <th class="pb-2 pr-4 font-medium">Model</th>
            <th class="pb-2 pr-4 font-medium text-right">Requests</th>
            <th class="pb-2 pr-4 font-medium text-right">Tokens</th>
            <th class="pb-2 font-medium text-right">Cost</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="p in store.summary"
            :key="`${p.provider}-${p.model}`"
            class="border-b border-bg-border/30 hover:bg-bg-elevated/30 transition-colors"
          >
            <td class="py-2.5 pr-4 text-primary capitalize">{{ p.provider }}</td>
            <td class="py-2.5 pr-4 text-secondary font-mono text-xs">{{ p.model }}</td>
            <td class="py-2.5 pr-4 text-secondary text-right font-mono">{{ (p.total_requests ?? 0).toLocaleString() }}</td>
            <td class="py-2.5 pr-4 text-secondary text-right font-mono">{{ ((p.total_tokens ?? 0) / 1000).toFixed(1) }}K</td>
            <td class="py-2.5 text-primary text-right font-mono font-medium">${{ (p.total_cost ?? 0).toFixed(4) }}</td>
          </tr>
        </tbody>
      </table>
      <div v-if="!store.loading && store.summary.length === 0" class="flex flex-col items-center justify-center py-12 text-center">
        <div class="w-12 h-12 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-3">
          <Cpu class="w-6 h-6 text-muted" />
        </div>
        <p class="text-primary font-medium">No usage data yet</p>
      </div>
    </div>
  </div>
</template>
