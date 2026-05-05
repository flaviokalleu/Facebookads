export interface BreakdownRow {
  dim: Record<string, string>
  spend: number
  impressions: number
  clicks: number
  leads: number
  cpl: number
  ctr: number
}

export type BreakdownDim = 'region' | 'hour' | 'age_gender' | 'placement' | 'device'

export function useBreakdowns() {
  const api = useApi()

  async function fetch(accountId: string, dim: BreakdownDim, days = 7): Promise<BreakdownRow[]> {
    try {
      const res = await api.get<{ data: { rows: BreakdownRow[] } }>(
        `/contas/${accountId}/breakdowns?dim=${dim}&days=${days}`,
      )
      return res?.data?.rows || []
    } catch {
      return []
    }
  }

  return { fetch }
}
