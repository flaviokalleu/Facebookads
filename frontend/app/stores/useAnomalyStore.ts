import { defineStore } from 'pinia'
import { api } from '~/lib/api'
import type { Anomaly } from '~/lib/api'

export const useAnomalyStore = defineStore('anomaly', () => {
  const anomalies = ref<Anomaly[]>([])
  const loading = ref(false)
  const severityFilter = ref<string>('all')

  const filtered = computed(() => {
    if (severityFilter.value === 'all') return anomalies.value
    return anomalies.value.filter(a => a.severity === severityFilter.value)
  })

  const highCount = computed(() => anomalies.value.filter(a => a.severity === 'HIGH').length)
  const mediumCount = computed(() => anomalies.value.filter(a => a.severity === 'MEDIUM').length)

  async function fetchAll() {
    loading.value = true
    try {
      anomalies.value = (await api.dashboard.anomalies()) ?? []
    } finally {
      loading.value = false
    }
  }

  return { anomalies, loading, severityFilter, filtered, highCount, mediumCount, fetchAll }
})
