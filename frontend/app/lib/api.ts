// All API calls go here. Never call $fetch directly in components or stores.

export interface ApiResponse<T> {
  data: T
  meta?: Record<string, unknown>
  error?: { code: string; message: string }
}

let _token: string | null = null

export function setToken(token: string) {
  _token = token
  if (import.meta.client) localStorage.setItem('auth_token', token)
}

export function getToken(): string | null {
  if (_token) return _token
  if (import.meta.client) _token = localStorage.getItem('auth_token')
  return _token
}

export function clearToken() {
  _token = null
  if (import.meta.client) localStorage.removeItem('auth_token')
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const config = useRuntimeConfig()
  const base = config.public.apiBase as string

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> ?? {}),
  }
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(`${base}${path}`, { ...options, headers })
  const json = await res.json().catch(() => null)

  if (!res.ok) {
    const msg = json?.error?.message ?? `HTTP ${res.status}`
    throw new Error(msg)
  }
  // Backend wraps in { data: ... }; unwrap automatically
  return (json?.data ?? json) as T
}

// ─── Auth ─────────────────────────────────────────────────────────────────────
export const api = {
  auth: {
    register: (body: { email: string; password: string; name: string }) =>
      request<{ token: string; user: User }>('/auth/register', {
        method: 'POST', body: JSON.stringify(body),
      }),
    login: (body: { email: string; password: string }) =>
      request<{ token: string; user: User }>('/auth/login', {
        method: 'POST', body: JSON.stringify(body),
      }),
    connectMeta: (body: { access_token: string; ad_account_id: string }) =>
      request<unknown>('/auth/meta/connect', {
        method: 'POST', body: JSON.stringify(body),
      }),
    metaStatus: () =>
      request<{ accounts: MetaAccount[] }>('/auth/meta/status'),
    listAdAccounts: (accessToken: string) =>
      request<MetaAccount[]>(`/auth/meta/accounts?access_token=${encodeURIComponent(accessToken)}`),
    me: () =>
      request<User>('/auth/me'),
  },

  // ─── Dashboard ──────────────────────────────────────────────────────────────
  dashboard: {
    summary: () =>
      request<DashboardSummary>('/dashboard/summary'),
    campaigns: () =>
      request<CampaignWithMetrics[]>('/dashboard/campaigns'),
    anomalies: () =>
      request<Anomaly[]>('/dashboard/anomalies'),
    recommendations: () =>
      request<Recommendation[]>('/dashboard/recommendations'),
    budgetSuggestions: () =>
      request<BudgetSuggestion[]>('/dashboard/budget-advisor'),
  },

  // ─── Campaigns ──────────────────────────────────────────────────────────────
  campaigns: {
    list: (datePreset = 'last_7d') =>
      request<CampaignWithMetrics[]>(`/campaigns?date_preset=${datePreset}`),
    get: (id: string) =>
      request<Campaign>(`/campaigns/${id}`),
    create: (body: { name: string; ad_account_id: string; objective?: string; daily_budget?: number; lifetime_budget?: number }) =>
      request<Campaign>('/campaigns', { method: 'POST', body: JSON.stringify(body) }),
    update: (id: string, body: { name?: string; status?: string; objective?: string; daily_budget?: number; lifetime_budget?: number }) =>
      request<Campaign>(`/campaigns/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    delete: (id: string) =>
      request<void>(`/campaigns/${id}`, { method: 'DELETE' }),
    sync: () =>
      request<{ synced_campaigns: number }>('/campaigns/sync', { method: 'POST' }),
    insights: (id: string, datePreset = 'last_7d') =>
      request<CampaignInsight[]>(`/campaigns/${id}/insights?date_preset=${datePreset}`),
    recommendations: (id: string) =>
      request<Recommendation[]>(`/campaigns/${id}/recommendations`),
    creativeInsights: () =>
      request<CreativeInsight[]>('/campaigns/creatives'),
    applyRecommendation: (id: string) =>
      request<{ applied: boolean }>(`/recommendations/${id}/apply`, { method: 'POST' }),
    applyBudgetSuggestion: (id: string) =>
      request<{ applied: boolean }>(`/budget-suggestions/${id}/apply`, { method: 'POST' }),
    autoOptimize: (id: string, body: { niche: string; min_age: number; max_age: number; location?: string; interests?: string }) =>
      request<{ targeting: any; results: any[]; model_used: string }>(`/campaigns/${id}/auto-optimize`, { method: 'POST', body: JSON.stringify(body) }),
    createFromPrompt: (prompt: string) =>
      request<{ campaign: Campaign; model_used: string }>('/campaigns/create-from-prompt', { method: 'POST', body: JSON.stringify({ prompt }) }),
    createFull: (body: { name: string; objective: string; budget: number; min_age: number; max_age: number; gender: string; interests: string; cities: string; country: string; city_details?: string; page_id?: string }) =>
      request<{ campaign: Campaign; meta_campaign_id: string; meta_ad_set_id: string; meta_ad_id: string }>('/campaigns/create-full', { method: 'POST', body: JSON.stringify(body) }),
    getRules: (id: string) =>
      request<any[]>('/campaigns/${id}/rules'),
    saveRules: (id: string, body: { rules: any[] }) =>
      request<{ saved: boolean }>('/campaigns/${id}/rules', { method: 'POST', body: JSON.stringify(body) }),
    abTest: (id: string, body: { test_type: string; variant_name: string; min_age?: number; max_age?: number; gender?: string; interests?: string }) =>
      request<{ ad_set_id: string; note: string }>('/campaigns/${id}/ab-test', { method: 'POST', body: JSON.stringify(body) }),
  },

  // ─── Creatives ─────────────────────────────────────────────────────────────
  creatives: {
    list: () =>
      request<CreativeInsight[]>('/creatives'),
    analyze: () =>
      request<{ creatives: any[]; ai_insights: any; ai_analyzed: boolean; model_used?: string; message?: string }>('/creatives/analyze', { method: 'POST' }),
    improve: (body: { instructions: string; creative_id?: string; campaign_name?: string; headline?: string; body?: string; cta?: string; target_audience?: string }) =>
      request<{ variations: any[]; model_used: string }>('/creatives/improve', { method: 'POST', body: JSON.stringify(body) }),
  },

  // ─── Admin ──────────────────────────────────────────────────────────────────
  admin: {
    configs: () =>
      request<ConfigEntry[]>('/admin/config'),
    setConfig: (key: string, value: string, isSecret: boolean) =>
      request<unknown>(`/admin/config/${key}`, {
        method: 'PUT', body: JSON.stringify({ value, is_secret: isSecret }),
      }),
    providers: () =>
      request<ProviderInfo[]>('/admin/providers'),
    aiUsageSummary: () =>
      request<LLMProviderSummary[]>('/admin/ai-usage'),
    aiUsageDaily: () =>
      request<LLMDailyCost[]>('/admin/ai-usage/daily'),
  },
}

// ─── Types (mirror backend domain) ────────────────────────────────────────────

export interface User {
  id: string
  name: string
  email: string
  is_admin: boolean
  created_at: string
}

export interface MetaAccount {
  id: string
  name: string
  currency: string
}

export interface DashboardSummary {
  total_spend: number
  total_leads: number
  avg_ctr: number
  avg_cpc: number
  avg_roas: number
  spend_delta?: number
  leads_delta?: number
  ctr_delta?: number
  roas_delta?: number
  daily_spend?: { date: string; spend: number; leads: number }[]
  last_synced_at?: string
  is_stale: boolean
}

export interface Campaign {
  id: string
  meta_campaign_id: string
  name: string
  objective: string
  status: string
  buying_type?: string
  daily_budget?: number
  lifetime_budget?: number
  start_time?: string
  stop_time?: string
  health_status: 'SCALING' | 'HEALTHY' | 'AT_RISK' | 'UNDERPERFORMING'
  last_synced_at?: string
  recommendations?: Recommendation[]
}

export interface CampaignWithMetrics extends Campaign {
  spend_30d?: number
  leads_30d?: number
  avg_ctr_7d?: number
  avg_cpc_7d?: number
  avg_roas_7d?: number
}

export interface CampaignInsight {
  date: string
  spend: number
  ctr: number
  cpc: number
  roas: number
  leads: number
  impressions: number
  purchases?: number
  purchase_value?: number
}

export interface Anomaly {
  id: string
  campaign_id: string
  campaign_name?: string
  type: string
  severity: 'HIGH' | 'MEDIUM' | 'LOW'
  description: string
  is_active: boolean
  detected_at: string
}

export interface Recommendation {
  id: string
  campaign_id: string
  campaign_name?: string
  priority: 'HIGH' | 'MEDIUM' | 'LOW'
  category: string
  action: string
  expected_impact: string
  rationale: string
  model_used: string
  is_applied: boolean
  created_at?: string
}

export interface BudgetSuggestion {
  id: string
  campaign_id?: string
  campaign_name?: string
  current_budget: number
  suggested_budget: number
  suggested_change: number
  rationale: string
  expected_impact: string
  model_used: string
  created_at?: string
}

export interface CreativeInsight {
  id: string
  campaign_name: string
  ad_id?: string
  headline?: string
  fatigue_score: number
  ctr: number
  impressions: number
  frequency?: number
  recommendation?: string
}

export interface ConfigEntry {
  key: string
  value: string
  is_secret: boolean
  description?: string
  updated_at: string
}

export interface ProviderInfo {
  name: string
  model_id: string
  available: boolean
  cost_per_1m_input?: number
  cost_per_1m_output?: number
  total_requests?: number
  total_cost?: number
}

export interface LLMProviderSummary {
  provider: string
  model: string
  total_requests: number
  total_tokens?: number
  total_cost: number
  avg_latency_ms?: number
}

export interface LLMDailyCost {
  date: string
  provider: string
  total_cost: number
}
