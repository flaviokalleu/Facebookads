import { defineStore } from 'pinia'
import { api } from '~/lib/api'

export const useProvidersStore = defineStore('providers', () => {
  const providers = ref<any[]>([])
  const configs = ref<any[]>([])
  const loading = ref(false)
  const saving = ref(false)

  async function fetchProviders() {
    loading.value = true
    try {
      providers.value = await api.admin.providers()
    } finally {
      loading.value = false
    }
  }

  async function fetchConfigs() {
    loading.value = true
    try {
      configs.value = await api.admin.configs()
    } finally {
      loading.value = false
    }
  }

  async function saveConfig(key: string, value: string, isSecret: boolean) {
    saving.value = true
    try {
      await api.admin.setConfig(key, value, isSecret)
      await fetchConfigs()
    } finally {
      saving.value = false
    }
  }

  return { providers, configs, loading, saving, fetchProviders, fetchConfigs, saveConfig }
})
