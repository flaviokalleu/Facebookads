export function useCampaignFilters() {
  const search = ref('')
  const healthFilter = ref('all')
  const sortBy = ref<'spend' | 'ctr' | 'roas' | 'name'>('spend')
  const sortDir = ref<'asc' | 'desc'>('desc')

  const HEALTH_OPTIONS = [
    { label: 'All',             value: 'all' },
    { label: 'Scaling',         value: 'SCALING' },
    { label: 'Healthy',         value: 'HEALTHY' },
    { label: 'At Risk',         value: 'AT_RISK' },
    { label: 'Underperforming', value: 'UNDERPERFORMING' },
  ]

  function applyFilters<T extends { name: string; health_status: string; spend_today?: number; ctr?: number; roas?: number }>(
    list: T[]
  ): T[] {
    let out = [...list]

    if (healthFilter.value !== 'all') {
      out = out.filter(c => c.health_status === healthFilter.value)
    }

    if (search.value.trim()) {
      const q = search.value.toLowerCase()
      out = out.filter(c => c.name.toLowerCase().includes(q))
    }

    out.sort((a, b) => {
      let av = 0, bv = 0
      if (sortBy.value === 'name') return sortDir.value === 'asc'
        ? a.name.localeCompare(b.name)
        : b.name.localeCompare(a.name)
      if (sortBy.value === 'spend') { av = a.spend_30d ?? 0; bv = b.spend_30d ?? 0 }
      if (sortBy.value === 'ctr')   { av = a.avg_ctr_7d ?? 0;         bv = b.avg_ctr_7d ?? 0 }
      if (sortBy.value === 'roas')  { av = a.avg_roas_7d ?? 0;        bv = b.avg_roas_7d ?? 0 }
      return sortDir.value === 'asc' ? av - bv : bv - av
    })

    return out
  }

  return { search, healthFilter, sortBy, sortDir, HEALTH_OPTIONS, applyFilters }
}
