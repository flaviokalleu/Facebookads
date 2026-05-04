export interface AIAction {
  id: string
  account_meta_id: string
  action_type: string
  target_meta_id: string
  target_kind: string
  reason: string
  metric_snapshot?: Record<string, any>
  proposed_change?: Record<string, any>
  source: 'rules' | 'deepseek'
  mode: 'auto' | 'propose'
  status: 'pending' | 'approved' | 'executed' | 'rejected' | 'failed' | 'reverted'
  meta_response?: Record<string, any>
  created_at: string
  decided_at?: string
  executed_at?: string
}

export interface SafetyRule {
  rule_key: string
  rule_value: number
  account_meta_id: string | null
  is_default: boolean
  description?: string
}

export interface SafetyRulesResponse {
  defaults: Record<string, number>
  effective: Record<string, number>
  overrides: Array<{ rule_key: string; rule_value: number; account_meta_id: string | null }>
}

export function useAiActions() {
  const api = useApi()

  async function list(status?: AIAction['status'], limit = 50): Promise<AIAction[]> {
    const qs = new URLSearchParams()
    if (status) qs.set('status', status)
    qs.set('limit', String(limit))
    try {
      const res = await api.get<{ data: AIAction[] }>(`/ai/actions?${qs.toString()}`)
      return res?.data ?? []
    } catch {
      return []
    }
  }

  async function approve(id: string) {
    return api.post(`/ai/actions/${id}/approve`)
  }

  async function reject(id: string) {
    return api.post(`/ai/actions/${id}/reject`)
  }

  async function revert(id: string) {
    return api.post(`/ai/actions/${id}/revert`)
  }

  async function listRules(): Promise<SafetyRule[]> {
    try {
      const res = await api.get<{ data: SafetyRulesResponse }>('/ai/safety-rules')
      const d = res?.data
      if (!d) return []
      const overrideMap = new Map(d.overrides.map(o => [o.rule_key, o]))
      return Object.entries(d.effective).map(([key, value]) => {
        const override = overrideMap.get(key)
        return {
          rule_key: key,
          rule_value: value,
          account_meta_id: override?.account_meta_id ?? null,
          is_default: !override,
        }
      })
    } catch {
      return []
    }
  }

  async function setRule(ruleKey: string, value: number, accountMetaId?: string) {
    return api.put(`/ai/safety-rules/${ruleKey}`, { value, account_meta_id: accountMetaId ?? null })
  }

  return { list, approve, reject, revert, listRules, setRule }
}
