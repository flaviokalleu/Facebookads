// Wraps API calls with loading, error, and data state.
// Use in components/stores for consistent async handling.

export function useApi<T>(
  fn: () => Promise<T>,
  options: { immediate?: boolean } = {},
) {
  const data = ref<T | null>(null) as Ref<T | null>
  const error = ref<string | null>(null)
  const loading = ref(false)
  const called = ref(false)

  async function execute(overrides?: { signal?: AbortSignal }): Promise<T | undefined> {
    loading.value = true
    error.value = null
    called.value = true
    try {
      const result = await fn()
      data.value = result
      return result
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === 'AbortError') return
      error.value = e instanceof Error ? e.message : 'Unknown error'
      return undefined
    } finally {
      loading.value = false
    }
  }

  if (options.immediate) execute()

  function reset() {
    data.value = null
    error.value = null
    loading.value = false
    called.value = false
  }

  return { data, error, loading, called, execute, reset }
}
