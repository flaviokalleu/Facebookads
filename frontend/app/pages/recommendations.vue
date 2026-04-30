<script setup lang="ts">
import { api } from '~/lib/api'
import type { Recommendation } from '~/lib/api'
import { Lightbulb, Filter } from 'lucide-vue-next'

const recommendations = ref<Recommendation[]>([])
const loading = ref(true)
const priorityFilter = ref('all')

onMounted(async () => {
  try {
    recommendations.value = await api.dashboard.recommendations()
  } finally {
    loading.value = false
  }
})

const PRIORITY_OPTIONS = [
  { label: 'All',    value: 'all' },
  { label: 'High',   value: 'HIGH' },
  { label: 'Medium', value: 'MEDIUM' },
  { label: 'Low',    value: 'LOW' },
]

const filtered = computed(() => {
  if (priorityFilter.value === 'all') return recommendations.value
  return recommendations.value.filter(r => r.priority === priorityFilter.value)
})

const highCount = computed(() => recommendations.value.filter(r => r.priority === 'HIGH').length)
</script>

<template>
  <div class="space-y-5">
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div class="flex items-center gap-2">
        <Lightbulb class="w-5 h-5 text-amber-400" />
        <div>
          <h1 class="text-primary text-xl font-bold">Recommendations</h1>
          <p class="text-muted text-sm">AI-generated optimizations</p>
        </div>
      </div>
      <div class="flex gap-2">
        <span v-if="highCount > 0" class="text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded-full px-3 py-1.5">
          {{ highCount }} high priority
        </span>
      </div>
    </div>

    <!-- Priority filter -->
    <div class="flex gap-2 flex-wrap items-center">
      <Filter class="w-4 h-4 text-muted" />
      <button
        v-for="opt in PRIORITY_OPTIONS"
        :key="opt.value"
        class="text-xs px-3 py-1.5 rounded-full border transition-all"
        :class="priorityFilter === opt.value
          ? 'bg-blue-default/20 text-blue-bright border-blue-default/40'
          : 'text-muted border-bg-border/50 hover:text-secondary hover:border-bg-border'"
        @click="priorityFilter = opt.value"
      >
        {{ opt.label }}
      </button>
    </div>

    <template v-if="loading">
      <div class="grid md:grid-cols-2 gap-3">
        <SkeletonCard v-for="i in 4" :key="i" />
      </div>
    </template>
    <div v-else-if="filtered.length === 0" class="flex flex-col items-center justify-center py-16 text-center">
      <div class="w-16 h-16 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-4">
        <Lightbulb class="w-8 h-8 text-muted" />
      </div>
      <p class="text-primary font-medium mb-1">No recommendations</p>
      <p class="text-muted text-sm max-w-xs">Run AI analysis to generate recommendations.</p>
    </div>
    <div v-else class="grid md:grid-cols-2 gap-3">
      <RecommendationCard v-for="r in filtered" :key="r.id" :recommendation="r" />
    </div>
  </div>
</template>
