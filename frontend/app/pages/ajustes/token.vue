<script setup lang="ts">
import { ShieldCheck, ShieldAlert, ShieldX, RefreshCw, KeyRound, Calendar, User as UserIcon } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'

interface Health {
  connected: boolean
  meta_user_id?: string
  app_id?: string
  token_type?: string
  scopes?: string[]
  is_active?: boolean
  last_refresh?: string
  expires_at?: string | null
  days_remaining?: number | null
  live_valid?: boolean
  live_scopes?: string[]
  live_expires_at?: string
  live_error?: string
}

const api = useApi()
const toast = useToast()
const health = ref<Health | null>(null)
const loading = ref(true)
const refreshing = ref(false)
const message = ref<{ ok: boolean; text: string } | null>(null)

async function load() {
  loading.value = true
  try {
    const res = await api.get<{ data: Health }>('/auth/meta/token/health')
    health.value = res.data
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function onRefresh() {
  refreshing.value = true
  message.value = null
  try {
    const res = await api.post<{ data: { refreshed: boolean; days_remaining: number } }>('/auth/meta/token/refresh')
    const days = res.data.days_remaining
    message.value = { ok: true, text: `Token renovado. Vale por ${days} dias.` }
    toast.success('Token renovado', `Novo token vale ${days} dias.`)
    await load()
  } catch (e: any) {
    const msg = e?.data?.error?.message || e?.message || 'Falha ao renovar.'
    message.value = { ok: false, text: msg }
    toast.error('Não foi possível renovar', msg)
  } finally {
    refreshing.value = false
  }
}

const totalDays = 60
const pct = computed(() => {
  const d = health.value?.days_remaining ?? 0
  return Math.max(2, Math.min(100, (d / totalDays) * 100))
})

const variant = computed<'success' | 'warning' | 'danger'>(() => {
  const d = health.value?.days_remaining ?? 0
  if (!health.value?.connected || health.value?.live_valid === false) return 'danger'
  if (d > 14) return 'success'
  if (d > 7) return 'warning'
  return 'danger'
})

const barColor = computed(() => ({
  success: 'bg-success', warning: 'bg-warning', danger: 'bg-danger',
}[variant.value]))

const headlineIcon = computed(() => ({
  success: ShieldCheck, warning: ShieldAlert, danger: ShieldX,
}[variant.value]))

const headline = computed(() => {
  if (!health.value?.connected) return 'Nenhum token Meta conectado'
  if (health.value?.live_valid === false) return 'Token rejeitado pelo Meta'
  const d = health.value?.days_remaining ?? 0
  if (d <= 0) return 'Token expirado — renove agora'
  if (d > 30) return 'Token saudável'
  if (d > 14) return 'Token vai expirar em algumas semanas'
  if (d > 7) return 'Atenção — renove esta semana'
  return 'Crítico — renove imediatamente'
})

const subline = computed(() => {
  if (!health.value?.connected) return 'Volte ao onboarding e conecte sua conta Meta.'
  if (health.value?.live_valid === false) return health.value?.live_error || 'A Meta marcou seu token como inválido.'
  const d = health.value?.days_remaining ?? 0
  if (d <= 0) return 'Sem renovação, a sincronização para de funcionar.'
  if (health.value?.expires_at) {
    return `Vale até ${new Date(health.value.expires_at).toLocaleDateString('pt-BR')}.`
  }
  return 'Token sem data de expiração (System User).'
})

function formatDateTime(s?: string) {
  if (!s) return '—'
  return new Date(s).toLocaleString('pt-BR')
}

function tokenTypeLabel(t?: string) {
  return ({
    user: 'Usuário',
    system_user: 'System User (não expira)',
    page: 'Página',
  } as Record<string, string>)[t || ''] || t || '—'
}
</script>

<template>
  <div class="max-w-3xl space-y-5">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-ink">Acesso Meta</h1>
      <p class="text-sm text-ink-muted">Saúde do token que conecta o sistema à API da Meta.</p>
    </div>

    <UiSkeleton v-if="loading" class="h-48" />

    <UiCard v-else-if="!health?.connected" class="!p-8 text-center">
      <ShieldX class="mx-auto h-10 w-10 text-danger" />
      <p class="mt-3 font-semibold text-ink">Nenhuma conta Meta conectada</p>
      <p class="mt-1 text-sm text-ink-muted">Você precisa fazer o onboarding pra usar o sistema.</p>
      <div class="mt-4">
        <NuxtLink to="/onboarding"><UiButton variant="primary">Conectar agora</UiButton></NuxtLink>
      </div>
    </UiCard>

    <template v-else>
      <UiCard>
        <div class="flex items-start gap-4">
          <component
            :is="headlineIcon"
            :class="[
              'mt-0.5 h-6 w-6 shrink-0',
              variant === 'success' ? 'text-success' : variant === 'warning' ? 'text-warning' : 'text-danger',
            ]"
          />
          <div class="flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <p class="font-semibold text-ink">{{ headline }}</p>
              <UiBadge v-if="health.days_remaining !== null && health.days_remaining !== undefined" :variant="variant">
                {{ health.days_remaining }} dias restantes
              </UiBadge>
              <UiBadge v-else variant="success">Não expira</UiBadge>
            </div>
            <p class="mt-1 text-sm text-ink-muted">{{ subline }}</p>

            <div v-if="health.expires_at && health.days_remaining !== null" class="mt-5">
              <div class="h-2 overflow-hidden rounded-full bg-bg-muted">
                <div :class="['h-full rounded-full transition-all', barColor]" :style="{ width: `${pct}%` }" />
              </div>
              <div class="mt-2 flex justify-between text-xs text-ink-muted">
                <span>Renovado em {{ formatDateTime(health.last_refresh) }}</span>
                <span>Expira em {{ formatDateTime(health.expires_at) }}</span>
              </div>
            </div>

            <div class="mt-5 flex flex-wrap items-center gap-2">
              <UiButton variant="primary" :loading="refreshing" @click="onRefresh">
                <RefreshCw v-if="!refreshing" class="h-4 w-4" /> Renovar agora
              </UiButton>
              <span v-if="message" :class="message.ok ? 'text-sm text-success' : 'text-sm text-danger'">
                {{ message.text }}
              </span>
            </div>
          </div>
        </div>
      </UiCard>

      <!-- Detalhes técnicos -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Detalhes técnicos</h2>
        <dl class="mt-4 grid gap-3 sm:grid-cols-2">
          <div>
            <dt class="flex items-center gap-1 text-xs text-ink-faint">
              <UserIcon class="h-3 w-3" /> Usuário Meta
            </dt>
            <dd class="mt-0.5 font-mono text-sm text-ink">{{ health.meta_user_id }}</dd>
          </div>
          <div>
            <dt class="flex items-center gap-1 text-xs text-ink-faint">
              <KeyRound class="h-3 w-3" /> App ID
            </dt>
            <dd class="mt-0.5 font-mono text-sm text-ink">{{ health.app_id }}</dd>
          </div>
          <div>
            <dt class="flex items-center gap-1 text-xs text-ink-faint">
              <ShieldCheck class="h-3 w-3" /> Tipo de token
            </dt>
            <dd class="mt-0.5 text-sm text-ink">{{ tokenTypeLabel(health.token_type) }}</dd>
          </div>
          <div>
            <dt class="flex items-center gap-1 text-xs text-ink-faint">
              <Calendar class="h-3 w-3" /> Última verificação
            </dt>
            <dd class="mt-0.5 text-sm text-ink">{{ formatDateTime(health.last_refresh) }}</dd>
          </div>
        </dl>
      </UiCard>

      <!-- Permissões -->
      <UiCard v-if="health.scopes?.length">
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Permissões concedidas</h2>
        <p class="mt-1 text-xs text-ink-muted">O que esse token está autorizado a fazer na Meta.</p>
        <div class="mt-4 flex flex-wrap gap-2">
          <UiBadge v-for="s in health.scopes" :key="s" variant="neutral">{{ s }}</UiBadge>
        </div>
      </UiCard>

      <UiCard v-if="health.live_valid === false" class="border-danger/40">
        <div class="flex items-start gap-3">
          <ShieldX class="mt-0.5 h-5 w-5 text-danger shrink-0" />
          <div>
            <p class="font-semibold text-ink">A Meta rejeitou seu token agora</p>
            <p class="mt-1 text-sm text-ink-muted">
              Mesmo dentro do prazo, o token não está mais aceito.
              Pode ter sido revogado pelo dono do app, pela Meta por inatividade, ou alguém mexeu nas permissões.
            </p>
            <p v-if="health.live_error" class="mt-2 text-xs text-danger">{{ health.live_error }}</p>
            <div class="mt-3">
              <NuxtLink to="/onboarding"><UiButton variant="ghost" size="sm">Refazer onboarding</UiButton></NuxtLink>
            </div>
          </div>
        </div>
      </UiCard>
    </template>
  </div>
</template>
