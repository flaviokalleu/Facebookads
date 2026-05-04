import { defineStore } from 'pinia'

interface User {
  id?: string
  email?: string
  name?: string
  is_admin?: boolean
}

const TOKEN_COOKIE = 'auth_token'
const USER_COOKIE  = 'auth_user'

const cookieOpts = {
  maxAge: 60 * 60 * 24 * 7,   // 7 dias
  sameSite: 'lax' as const,
  path: '/',
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    token: null as string | null,
    _hydrated: false,
  }),
  getters: {
    isAuthenticated: (s) => !!s.token,
  },
  actions: {
    hydrate() {
      if (this._hydrated) return
      const tokenCookie = useCookie<string | null>(TOKEN_COOKIE, cookieOpts)
      const userCookie  = useCookie<User  | null>(USER_COOKIE,  cookieOpts)
      this.token = tokenCookie.value || null
      this.user  = (userCookie.value as User | null) || null
      this._hydrated = true
    },
    setSession(token: string, user: User | null) {
      this.token = token
      this.user = user
      const tokenCookie = useCookie<string | null>(TOKEN_COOKIE, cookieOpts)
      const userCookie  = useCookie<User  | null>(USER_COOKIE,  cookieOpts)
      tokenCookie.value = token
      userCookie.value  = user
    },
    logout() {
      this.token = null
      this.user = null
      const tokenCookie = useCookie<string | null>(TOKEN_COOKIE, cookieOpts)
      const userCookie  = useCookie<User  | null>(USER_COOKIE,  cookieOpts)
      tokenCookie.value = null
      userCookie.value  = null
    },
  },
})
