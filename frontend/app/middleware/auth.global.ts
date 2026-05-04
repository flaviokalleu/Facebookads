import { useAuthStore } from '~/stores/auth'

const PUBLIC_ROUTES = new Set(['/', '/login'])

export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  auth.hydrate()
  if (PUBLIC_ROUTES.has(to.path)) return
  if (!auth.token) {
    return navigateTo('/login')
  }
})
