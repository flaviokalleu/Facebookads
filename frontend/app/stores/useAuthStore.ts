import { defineStore } from 'pinia'
import { api, setToken, clearToken, getToken } from '~/lib/api'
import type { User } from '~/lib/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.is_admin ?? false)

  async function init() {
    const token = getToken()
    if (!token) return
    try {
      user.value = await api.auth.me()
    } catch {
      clearToken()
    }
  }

  async function login(email: string, password: string) {
    loading.value = true
    error.value = null
    try {
      const res = await api.auth.login({ email, password })
      setToken(res.token)
      user.value = res.user
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function register(name: string, email: string, password: string) {
    loading.value = true
    error.value = null
    try {
      const res = await api.auth.register({ name, email, password })
      setToken(res.token)
      user.value = res.user
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Registration failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  function logout() {
    clearToken()
    user.value = null
    navigateTo('/login')
  }

  return { user, loading, error, isAuthenticated, isAdmin, init, login, register, logout }
})
