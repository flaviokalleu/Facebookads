import type { Anomaly } from '~/lib/api'

export function useAnomalyFilter(anomalies: Ref<Anomaly[]>) {
  const severityFilter = ref<'ALL' | 'HIGH' | 'MEDIUM' | 'LOW'>('ALL')
  const showResolved = ref(false)

  const filtered = computed(() => {
    let list = anomalies.value
    if (!showResolved.value) {
      list = list.filter(a => a.is_active)
    }
    if (severityFilter.value !== 'ALL') {
      list = list.filter(a => a.severity === severityFilter.value)
    }
    return list
  })

  const highCount = computed(() => anomalies.value.filter(a => a.is_active && a.severity === 'HIGH').length)
  const mediumCount = computed(() => anomalies.value.filter(a => a.is_active && a.severity === 'MEDIUM').length)
  const lowCount = computed(() => anomalies.value.filter(a => a.is_active && a.severity === 'LOW').length)

  function setFilter(s: typeof severityFilter.value) {
    severityFilter.value = s
  }

  return {
    severityFilter,
    showResolved,
    filtered,
    highCount,
    mediumCount,
    lowCount,
    setFilter,
  }
}
