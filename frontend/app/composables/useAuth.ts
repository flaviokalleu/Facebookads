import { useAuthStore } from '~/stores/auth'

interface LoginPayload { email: string; password: string }
interface RegisterPayload { email: string; password: string; name?: string }
interface AuthEnvelope { data: { token: string; expires_at?: string; user?: any } }

export function useAuth() {
  const auth = useAuthStore()
  const api = useApi()

  async function login(payload: LoginPayload) {
    const res = await api.post<AuthEnvelope>('/auth/login', payload)
    const d = res.data
    auth.setSession(d.token, d.user || { email: payload.email })
    return d
  }

  async function register(payload: RegisterPayload) {
    const res = await api.post<AuthEnvelope>('/auth/register', payload)
    const d = res.data
    auth.setSession(d.token, d.user || { email: payload.email, name: payload.name })
    return d
  }

  async function me() {
    const res = await api.get<{ data: any }>('/auth/me')
    if (res?.data) auth.user = res.data
    return res?.data
  }

  function logout() {
    auth.logout()
  }

  return { login, register, me, logout }
}
