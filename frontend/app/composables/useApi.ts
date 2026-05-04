import { useAuthStore } from '~/stores/auth'

export function useApi() {
  const config = useRuntimeConfig()
  const auth = useAuthStore()

  const request = <T>(path: string, opts: any = {}): Promise<T> => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(opts.headers || {}),
    }
    if (auth.token) headers.Authorization = `Bearer ${auth.token}`
    return $fetch<T>(`${config.public.apiBase}${path}`, { ...opts, headers })
  }

  return {
    get:    <T>(path: string, opts: any = {}) => request<T>(path, { ...opts, method: 'GET' }),
    post:   <T>(path: string, body?: any, opts: any = {}) => request<T>(path, { ...opts, method: 'POST', body }),
    put:    <T>(path: string, body?: any, opts: any = {}) => request<T>(path, { ...opts, method: 'PUT', body }),
    del:    <T>(path: string, opts: any = {}) => request<T>(path, { ...opts, method: 'DELETE' }),
  }
}
