import { api, type ProviderInfo } from '~/lib/api'

// Polls provider health every 60 seconds.
export function useProviderStatus() {
  const providers = ref<ProviderInfo[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let intervalId: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    loading.value = true
    try {
      providers.value = await api.admin.providers()
      error.value = null
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load providers'
    } finally {
      loading.value = false
    }
  }

  function startPolling() {
    refresh()
    intervalId = setInterval(refresh, 60000)
  }

  function stopPolling() {
    if (intervalId) {
      clearInterval(intervalId)
      intervalId = null
    }
  }

  const availableCount = computed(() => providers.value.filter(p => p.available).length)
  const totalCount = computed(() => providers.value.length)

  onUnmounted(() => stopPolling())

  return { providers, loading, error, refresh, startPolling, stopPolling, availableCount, totalCount }
}
