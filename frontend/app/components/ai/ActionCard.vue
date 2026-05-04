<script setup lang="ts">
import { CheckCircle2, XCircle, RotateCcw, AlertTriangle, Bot, Shield, Clock } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiButton from '~/components/ui/UiButton.vue'
import type { AIAction } from '~/composables/useAiActions'

interface Props { action: AIAction }
const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'approve', id: string): void
  (e: 'reject', id: string): void
  (e: 'revert', id: string): void
}>()

const labels: Record<string, string> = {
  pause_ad: 'Pausar anúncio',
  pause_adset: 'Pausar conjunto',
  scale_budget: 'Ajustar verba',
  rotate_creative: 'Rotacionar criativo',
  duplicate_adset: 'Duplicar conjunto vencedor',
  create_campaign: 'Criar nova campanha',
  alert: 'Atenção',
}

const statusVariant = computed(() => {
  switch (props.action.status) {
    case 'pending': return 'warning'
    case 'approved': return 'success'
    case 'executed': return 'success'
    case 'failed': return 'danger'
    case 'rejected': return 'neutral'
    case 'reverted': return 'neutral'
    default: return 'neutral'
  }
})

const statusLabel = computed(() => ({
  pending: 'Aguardando aprovação',
  approved: 'Aprovado',
  executed: 'Executado',
  rejected: 'Recusado',
  failed: 'Falhou',
  reverted: 'Revertido',
}[props.action.status] || props.action.status))

const sourceLabel = computed(() => props.action.source === 'deepseek' ? 'IA DeepSeek' : 'Regra automática')

function formatBRL(v: any) {
  const n = typeof v === 'string' ? parseFloat(v) : (v as number)
  if (Number.isNaN(n) || n == null) return '—'
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(n)
}

function formatTime(s?: string) {
  if (!s) return ''
  const d = new Date(s)
  return d.toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const metricLines = computed(() => {
  const m = props.action.metric_snapshot || {}
  const lines: { label: string, value: string }[] = []
  if (m.cpl != null)               lines.push({ label: 'Custo por contato', value: formatBRL(m.cpl) })
  if (m.account_avg_cpl != null)   lines.push({ label: 'Média da conta', value: formatBRL(m.account_avg_cpl) })
  if (m.spend != null)             lines.push({ label: 'Gasto', value: formatBRL(m.spend) })
  if (m.ctr != null)               lines.push({ label: 'CTR', value: `${(m.ctr * 100).toFixed(2)}%` })
  if (m.frequency != null)         lines.push({ label: 'Frequência', value: Number(m.frequency).toFixed(2) })
  return lines
})
</script>

<template>
  <UiCard>
    <div class="flex items-start gap-4">
      <div class="mt-0.5 rounded-full p-2"
           :class="action.mode === 'auto' ? 'bg-success-soft text-success' : 'bg-accent-soft text-accent'">
        <Shield v-if="action.mode === 'auto'" class="h-5 w-5" />
        <Bot v-else class="h-5 w-5" />
      </div>

      <div class="flex-1 min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <h3 class="text-sm font-semibold text-ink">
            {{ labels[action.action_type] || action.action_type }}
          </h3>
          <UiBadge :variant="(statusVariant as any)">{{ statusLabel }}</UiBadge>
          <UiBadge variant="neutral">{{ sourceLabel }}</UiBadge>
          <span class="ml-auto inline-flex items-center gap-1 text-xs text-ink-faint">
            <Clock class="h-3.5 w-3.5" /> {{ formatTime(action.created_at) }}
          </span>
        </div>

        <p class="mt-2 text-sm text-ink-muted">{{ action.reason }}</p>

        <dl v-if="metricLines.length" class="mt-3 grid grid-cols-2 gap-x-6 gap-y-1 text-xs sm:grid-cols-3">
          <template v-for="m in metricLines" :key="m.label">
            <dt class="text-ink-faint">{{ m.label }}</dt>
            <dd class="font-medium text-ink">{{ m.value }}</dd>
          </template>
        </dl>

        <div class="mt-4 flex flex-wrap items-center gap-2">
          <template v-if="action.status === 'pending'">
            <UiButton variant="primary" size="sm" @click="emit('approve', action.id)">
              <CheckCircle2 class="h-4 w-4" /> Aprovar
            </UiButton>
            <UiButton variant="ghost" size="sm" @click="emit('reject', action.id)">
              <XCircle class="h-4 w-4" /> Recusar
            </UiButton>
          </template>
          <template v-else-if="action.status === 'executed'">
            <UiButton variant="ghost" size="sm" @click="emit('revert', action.id)">
              <RotateCcw class="h-4 w-4" /> Reverter
            </UiButton>
          </template>
          <template v-else-if="action.status === 'failed'">
            <span class="inline-flex items-center gap-1 text-xs text-danger">
              <AlertTriangle class="h-4 w-4" /> Falhou ao aplicar
            </span>
          </template>
        </div>
      </div>
    </div>
  </UiCard>
</template>
