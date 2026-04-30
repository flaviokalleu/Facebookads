import { defineStore } from 'pinia'
import { api } from '~/lib/api'
import type { CampaignWithMetrics, Campaign } from '~/lib/api'

export const useCampaignStore = defineStore('campaign', () => {
  const campaigns = ref<CampaignWithMetrics[]>([])
  const current = ref<Campaign | null>(null)
  const loading = ref(false)
  const syncing = ref(false)
  const error = ref<string | null>(null)
  const metaConnected = ref(false)
  const metaAccounts = ref<string[]>([])

  const search = ref('')
  const statusFilter = ref<string>('all')
  const datePreset = ref('last_7d')

  const filtered = computed(() => {
    let result = campaigns.value
    if (statusFilter.value !== 'all') {
      result = result.filter(c => c.health_status === statusFilter.value)
    }
    if (search.value.trim()) {
      const q = search.value.toLowerCase()
      result = result.filter(c => c.name.toLowerCase().includes(q))
    }
    return result
  })

  async function fetchMetaStatus() {
    try {
      const res = await api.auth.metaStatus()
      metaAccounts.value = (res as any)?.accounts?.map((a: any) => a.ad_account_id) ?? []
      metaConnected.value = metaAccounts.value.length > 0
    } catch {
      metaConnected.value = false
    }
  }

  async function fetchAll(preset = datePreset.value) {
    loading.value = true
    error.value = null
    try {
      const data = await api.campaigns.list(preset)
      campaigns.value = data ?? []
      datePreset.value = preset
    } catch (e: any) {
      if (e.message?.includes('no Meta ad accounts')) {
        error.value = 'No Meta account connected. Click "Connect Meta" to link your ad account.'
      } else {
        error.value = e.message
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(id: string) {
    loading.value = true
    error.value = null
    try {
      current.value = await api.campaigns.get(id)
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function sync() {
    syncing.value = true
    error.value = null
    try {
      await api.campaigns.sync()
      await fetchAll()
    } catch (e: any) {
      error.value = e.message || 'Sync failed'
    } finally {
      syncing.value = false
    }
  }

  return {
    campaigns, current, loading, syncing, error,
    search, statusFilter, datePreset, filtered,
    metaConnected, metaAccounts,
    fetchMetaStatus, fetchAll, fetchOne, sync,
  }
})
