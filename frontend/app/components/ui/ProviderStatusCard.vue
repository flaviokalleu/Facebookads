<script setup lang="ts">
defineProps<{
  provider: {
    name: string
    model_id: string
    available: boolean
    cost_per_1m_input?: number
    cost_per_1m_output?: number
    total_requests?: number
    total_cost?: number
  }
}>()

const providerIcons: Record<string, string> = {
  anthropic: '🟠',
  openai: '🟢',
  google: '🔵',
  deepseek: '🔷',
  zhipu: '🟣',
  moonshot: '🌙',
  alibaba: '🔶',
  xai: '✖️',
}
</script>

<template>
  <div class="card">
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <span class="text-lg">{{ providerIcons[provider.name.toLowerCase()] ?? '🤖' }}</span>
        <div>
          <p class="text-primary text-sm font-semibold capitalize">{{ provider.name }}</p>
          <p class="text-muted text-xs font-mono">{{ provider.model_id }}</p>
        </div>
      </div>
      <span
        class="text-xs px-2 py-0.5 rounded-full border font-medium"
        :class="provider.available
          ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20'
          : 'text-red-400 bg-red-500/10 border-red-500/20'"
      >
        {{ provider.available ? 'Online' : 'Offline' }}
      </span>
    </div>

    <div class="grid grid-cols-2 gap-2 text-xs">
      <div class="bg-bg-elevated rounded-lg p-2">
        <p class="text-muted mb-0.5">Requests</p>
        <p class="text-primary font-mono font-medium">{{ (provider.total_requests ?? 0).toLocaleString() }}</p>
      </div>
      <div class="bg-bg-elevated rounded-lg p-2">
        <p class="text-muted mb-0.5">Total Cost</p>
        <p class="text-primary font-mono font-medium">${{ (provider.total_cost ?? 0).toFixed(2) }}</p>
      </div>
    </div>

    <div v-if="provider.cost_per_1m_input" class="mt-2 flex gap-3 text-xs text-muted">
      <span>In: ${{ provider.cost_per_1m_input }}/1M</span>
      <span>Out: ${{ provider.cost_per_1m_output }}/1M</span>
    </div>
  </div>
</template>
