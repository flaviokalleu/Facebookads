import { defineStore } from 'pinia'

interface User {
  id?: number
  email?: string
  name?: string
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    token: null as string | null,
  }),
  getters: {
    isAuthenticated: (s) => !!s.token,
  },
  actions: {
    hydrate() {
      if (import.meta.client) {
        const t = localStorage.getItem('auth_token')
        const u = localStorage.getItem('auth_user')
        if (t) this.token = t
        if (u) {
          try { this.user = JSON.parse(u) } catch {}
        }
      }
    },
    setSession(token: string, user: User | null) {
      this.token = token
      this.user = user
      if (import.meta.client) {
        localStorage.setItem('auth_token', token)
        if (user) localStorage.setItem('auth_user', JSON.stringify(user))
      }
    },
    logout() {
      this.token = null
      this.user = null
      if (import.meta.client) {
        localStorage.removeItem('auth_token')
        localStorage.removeItem('auth_user')
      }
    },
  },
})
