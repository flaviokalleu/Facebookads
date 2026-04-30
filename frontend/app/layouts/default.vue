<script setup lang="ts">
import { useAuthStore } from '~/stores/useAuthStore'
import {
  LayoutDashboard, Megaphone, AlertTriangle, Lightbulb,
  Palette, Wallet, Cpu, ChartNoAxesColumnIncreasing,
  LogOut, Menu, X, Bot
} from 'lucide-vue-next'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

onMounted(async () => {
  await auth.init()
  if (!auth.isAuthenticated) {
    router.push('/login')
  }
})

const sidebarOpen = ref(false)

function toggleSidebar() { sidebarOpen.value = !sidebarOpen.value }
function closeSidebar() { sidebarOpen.value = false }

const mainNav = [
  { label: 'Visão Geral',       to: '/dashboard',           icon: LayoutDashboard },
  { label: 'Campanhas',         to: '/campaigns',           icon: Megaphone },
  { label: 'Criativos',         to: '/creatives',           icon: Palette },
  { label: 'Orçamento',         to: '/budget',              icon: Wallet },
  { label: 'Anomalias',         to: '/anomalies',           icon: AlertTriangle },
  { label: 'Recomendações',     to: '/recommendations',     icon: Lightbulb },
]

const adminNav = [
  { label: 'Provedores IA',     to: '/admin/providers',     icon: Cpu },
  { label: 'Uso da IA',         to: '/admin/ai-usage',      icon: ChartNoAxesColumnIncreasing },
]

const pageTitle = computed(() => {
  const all = [...mainNav, ...adminNav]
  const match = all.find(n => route.path.startsWith(n.to))
  return match?.label ?? 'Dashboard'
})
</script>

<template>
  <div class="flex min-h-screen bg-bg-base">
    <!-- Sidebar Desktop -->
    <aside class="hidden lg:flex flex-col w-64 shrink-0 h-screen sticky top-0 bg-gradient-to-b from-[#0A1628] to-[#080F1E] border-r border-bg-border/50">
      <div class="px-5 py-5 border-b border-bg-border/50">
        <NuxtLink to="/dashboard" class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-default to-blue-glow flex items-center justify-center shadow-lg shadow-blue-default/30">
            <Bot class="w-5 h-5 text-white" />
          </div>
          <div>
            <p class="text-primary font-bold text-sm leading-tight">Meta Ads AI</p>
            <p class="text-blue-glow/60 text-2xs leading-tight font-medium">Gestor de Tráfego</p>
          </div>
        </NuxtLink>
      </div>

      <nav class="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
        <p class="px-3 pb-2 text-2xs font-semibold text-muted uppercase tracking-widest">Menu</p>
        <NuxtLink
          v-for="item in mainNav"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all duration-150"
          :class="route.path.startsWith(item.to)
            ? 'bg-gradient-to-r from-blue-default/20 to-transparent text-blue-bright font-medium border-l-2 border-blue-default'
            : 'text-secondary hover:text-primary hover:bg-bg-elevated/50'"
        >
          <component :is="item.icon" class="w-4 h-4 shrink-0" :class="route.path.startsWith(item.to) ? 'text-blue-glow' : 'text-muted'" />
          <span>{{ item.label }}</span>
        </NuxtLink>

        <template v-if="auth.isAdmin">
          <p class="mt-4 mb-1 px-3 pt-3 border-t border-bg-border/50 text-2xs font-semibold text-muted uppercase tracking-widest">Admin</p>
          <NuxtLink
            v-for="item in adminNav"
            :key="item.to"
            :to="item.to"
            class="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all duration-150"
            :class="route.path.startsWith(item.to)
              ? 'bg-gradient-to-r from-blue-default/20 to-transparent text-blue-bright font-medium border-l-2 border-blue-default'
              : 'text-secondary hover:text-primary hover:bg-bg-elevated/50'"
          >
            <component :is="item.icon" class="w-4 h-4 shrink-0" :class="route.path.startsWith(item.to) ? 'text-blue-glow' : 'text-muted'" />
            <span>{{ item.label }}</span>
          </NuxtLink>
        </template>
      </nav>

      <div class="p-4 border-t border-bg-border/50">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-full bg-gradient-to-br from-blue-default to-blue-glow flex items-center justify-center text-white text-xs font-bold shadow-lg shadow-blue-default/20">
            {{ auth.user?.name?.charAt(0)?.toUpperCase() ?? '?' }}
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-primary text-xs font-medium truncate">{{ auth.user?.name }}</p>
            <p class="text-muted text-2xs truncate">{{ auth.user?.email }}</p>
          </div>
          <button @click="auth.logout()" class="p-1.5 rounded-lg text-muted hover:text-red-400 hover:bg-red-500/10 transition-colors" title="Sair">
            <LogOut class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Drawer Mobile -->
    <Transition name="drawer">
      <div v-if="sidebarOpen" class="lg:hidden fixed inset-0 z-50">
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="closeSidebar" />
        <div class="relative w-64 h-full bg-gradient-to-b from-[#0A1628] to-[#080F1E] border-r border-bg-border/50 flex flex-col animate-slide-in">
          <div class="px-5 py-5 border-b border-bg-border/50 flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="w-9 h-9 rounded-xl bg-gradient-to-br from-blue-default to-blue-glow flex items-center justify-center">
                <Bot class="w-5 h-5 text-white" />
              </div>
              <span class="text-primary font-bold text-sm">Meta Ads AI</span>
            </div>
            <button @click="closeSidebar" class="p-1.5 rounded-lg text-muted hover:text-primary hover:bg-bg-elevated/50 transition-colors">
              <X class="w-4 h-4" />
            </button>
          </div>

          <nav class="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
            <p class="px-3 pb-2 text-2xs font-semibold text-muted uppercase tracking-widest">Menu</p>
            <NuxtLink
              v-for="item in mainNav"
              :key="item.to"
              :to="item.to"
              class="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all"
              :class="route.path.startsWith(item.to) ? 'bg-blue-default/20 text-blue-bright font-medium' : 'text-secondary'"
              @click="closeSidebar"
            >
              <component :is="item.icon" class="w-4 h-4" />
              {{ item.label }}
            </NuxtLink>

            <template v-if="auth.isAdmin">
              <p class="mt-4 mb-1 px-3 pt-3 border-t border-bg-border/50 text-2xs font-semibold text-muted uppercase tracking-widest">Admin</p>
              <NuxtLink
                v-for="item in adminNav"
                :key="item.to"
                :to="item.to"
                class="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all"
                :class="route.path.startsWith(item.to) ? 'bg-blue-default/20 text-blue-bright font-medium' : 'text-secondary'"
                @click="closeSidebar"
              >
                <component :is="item.icon" class="w-4 h-4" />
                {{ item.label }}
              </NuxtLink>
            </template>
          </nav>

          <div class="p-4 border-t border-bg-border/50">
            <div class="flex items-center gap-3">
              <div class="w-9 h-9 rounded-full bg-gradient-to-br from-blue-default to-blue-glow flex items-center justify-center text-white text-xs font-bold">{{ auth.user?.name?.charAt(0)?.toUpperCase() ?? '?' }}</div>
              <div class="flex-1 min-w-0">
                <p class="text-primary text-xs font-medium truncate">{{ auth.user?.name }}</p>
                <p class="text-muted text-2xs truncate">{{ auth.user?.email }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Main -->
    <div class="flex-1 min-w-0 flex flex-col">
      <AppTopbar :title="pageTitle" :subtitle="auth.isAdmin ? 'Administrador' : undefined" @toggle-sidebar="toggleSidebar" />
      <main class="flex-1 p-4 md:p-6 pb-24 lg:pb-6">
        <NuxtPage />
      </main>
    </div>

    <BottomNav class="lg:hidden" />
  </div>
</template>

<style scoped>
.drawer-enter-active { transition: opacity 0.25s ease-out; }
.drawer-leave-active { transition: opacity 0.15s ease-in; }
.drawer-enter-from, .drawer-leave-to { opacity: 0; }
.animate-slide-in { animation: slideInLeft 0.25s cubic-bezier(0.16, 1, 0.3, 1); }
@keyframes slideInLeft { from { transform: translateX(-100%); } to { transform: translateX(0); } }
</style>
