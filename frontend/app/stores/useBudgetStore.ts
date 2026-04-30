import { defineStore } from 'pinia'
import { api } from '~/lib/api'
import type { BudgetSuggestion } from '~/lib/api'

export const useBudgetStore = defineStore('budget', () => {
  const suggestions = ref<BudgetSuggestion[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const totalIncrease = computed(() =>
    suggestions.value
      .filter(s => s.suggested_change > 0)
      .reduce((sum, s) => sum + s.suggested_change, 0)
  )

  const totalDecrease = computed(() =>
    suggestions.value
      .filter(s => s.suggested_change < 0)
      .reduce((sum, s) => sum + Math.abs(s.suggested_change), 0)
  )

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      suggestions.value = (await api.dashboard.budgetSuggestions()) ?? []
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  return { suggestions, loading, error, totalIncrease, totalDecrease, fetchAll }
})
