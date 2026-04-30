import { defineStore } from 'pinia'
import { api, type DashboardSummary, type CampaignWithMetrics, type Anomaly, type BudgetSuggestion } from '~/lib/api'

export const useDashboardStore = defineStore('dashboard', () => {
  const summary = ref<DashboardSummary | null>(null)
  const campaigns = ref<CampaignWithMetrics[]>([])
  const anomalies = ref<Anomaly[]>([])
  const budgetSuggestions = ref<BudgetSuggestion[]>([])

  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchSummary() {
    try {
      summary.value = await api.dashboard.summary()
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load summary'
    }
  }

  async function fetchCampaigns() {
    try {
      campaigns.value = (await api.dashboard.campaigns()) ?? []
    } catch { /* show empty state */ }
  }

  async function fetchAnomalies() {
    try {
      anomalies.value = (await api.dashboard.anomalies()) ?? []
    } catch { /* show empty state */ }
  }

  async function fetchBudgetSuggestions() {
    try {
      budgetSuggestions.value = (await api.dashboard.budgetSuggestions()) ?? []
    } catch { /* show empty state */ }
  }

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      await Promise.all([fetchSummary(), fetchCampaigns(), fetchAnomalies(), fetchBudgetSuggestions()])
    } finally {
      loading.value = false
    }
  }

  const activeAnomalies = computed(() => anomalies.value.filter(a => a.is_active))
  const highSeverityCount = computed(() => activeAnomalies.value.filter(a => a.severity === 'HIGH').length)
  const scalingCampaigns = computed(() => campaigns.value.filter(c => c.health_status === 'SCALING').length)
  const underperformingCampaigns = computed(() => campaigns.value.filter(c => c.health_status === 'UNDERPERFORMING').length)

  return {
    summary, campaigns, anomalies, budgetSuggestions,
    loading, error,
    fetchSummary, fetchCampaigns, fetchAnomalies, fetchBudgetSuggestions, fetchAll,
    activeAnomalies, highSeverityCount, scalingCampaigns, underperformingCampaigns,
  }
})
