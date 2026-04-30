import { defineStore } from 'pinia'
import { api } from '~/lib/api'

export const useCreativeStore = defineStore('creative', () => {
  const topCreatives = ref<any[]>([])
  const bottomCreatives = ref<any[]>([])
  const aiInsights = ref<any>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      const result = await api.campaigns.creativeInsights()
      topCreatives.value = result.filter((c: any) => c.ctr > 0).slice(0, 5)
      bottomCreatives.value = result.filter((c: any) => c.ctr > 0).slice(-5).reverse()
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load creatives'
    } finally {
      loading.value = false
    }
  }

  return { topCreatives, bottomCreatives, aiInsights, loading, error, fetchAll }
})
