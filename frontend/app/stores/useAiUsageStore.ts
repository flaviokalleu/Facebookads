import { defineStore } from 'pinia'
import { api } from '~/lib/api'

export const useAiUsageStore = defineStore('aiUsage', () => {
  const summary = ref<any[]>([])
  const dailyCost = ref<any[]>([])
  const loading = ref(false)

  const totalCost = computed(() =>
    summary.value.reduce((sum: number, p: any) => sum + (p.total_cost ?? 0), 0)
  )
  const totalRequests = computed(() =>
    summary.value.reduce((sum: number, p: any) => sum + (p.total_requests ?? 0), 0)
  )

  async function fetchAll() {
    loading.value = true
    try {
      const [s, d] = await Promise.all([
        api.admin.aiUsageSummary(),
        api.admin.aiUsageDaily(),
      ])
      summary.value = s
      dailyCost.value = d
    } finally {
      loading.value = false
    }
  }

  return { summary, dailyCost, loading, totalCost, totalRequests, fetchAll }
})
