<script setup lang="ts">
import { Inbox, RefreshCw } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import ActionCard from '~/components/ai/ActionCard.vue'
import { useAiActions, type AIAction } from '~/composables/useAiActions'

const ai = useAiActions()
const toast = useToast()
const filter = ref<'all' | AIAction['status']>('all')
const items = ref<AIAction[]>([])
const loading = ref(true)
const errorMsg = ref<string | null>(null)

const tabs: { key: 'all' | AIAction['status'], label: string }[] = [
  { key: 'pending',  label: 'Aguardando' },
  { key: 'executed', label: 'Executadas' },
  { key: 'rejected', label: 'Recusadas' },
  { key: 'failed',   label: 'Falhas' },
  { key: 'all',      label: 'Tudo' },
]

async function load() {
  loading.value = true
  errorMsg.value = null
  try {
    items.value = await ai.list(filter.value === 'all' ? undefined : filter.value, 100)
  } catch (e: any) {
    errorMsg.value = e?.message || 'Não foi possível carregar.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(filter, load)

async function onApprove(id: string) {
  try {
    await ai.approve(id)
    toast.success('Ação aprovada e executada')
    await load()
  } catch (e: any) {
    const m = e?.data?.error?.message || e?.message || 'Falha ao aprovar'
    errorMsg.value = m
    toast.error('Não foi possível aprovar', m)
  }
}
async function onReject(id: string) {
  try {
    await ai.reject(id)
    toast.info('Ação recusada')
    await load()
  } catch (e: any) {
    const m = e?.data?.error?.message || e?.message || 'Falha ao recusar'
    errorMsg.value = m
    toast.error('Não foi possível recusar', m)
  }
}
async function onRevert(id: string) {
  try {
    await ai.revert(id)
    toast.success('Ação revertida', 'O anúncio voltou ao estado anterior.')
    await load()
  } catch (e: any) {
    const m = e?.data?.error?.message || e?.message || 'Falha ao reverter'
    errorMsg.value = m
    toast.error('Não foi possível reverter', m)
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-ink">O que a IA fez (e propôs)</h1>
        <p class="mt-1 text-sm text-ink-muted">
          Pausas automáticas seguras já foram aplicadas. Mudanças mais ousadas ficam aqui esperando seu OK.
        </p>
      </div>
      <UiButton variant="ghost" size="sm" @click="load">
        <RefreshCw class="h-4 w-4" /> Atualizar
      </UiButton>
    </div>

    <div class="flex flex-wrap gap-2">
      <button
        v-for="t in tabs"
        :key="t.key"
        type="button"
        :class="[
          'rounded-full px-3 py-1.5 text-sm transition',
          filter === t.key
            ? 'bg-accent text-white'
            : 'border border-border bg-bg text-ink-muted hover:bg-bg-muted',
        ]"
        @click="filter = t.key"
      >
        {{ t.label }}
      </button>
    </div>

    <div v-if="loading" class="space-y-3">
      <UiSkeleton class="h-24" />
      <UiSkeleton class="h-24" />
      <UiSkeleton class="h-24" />
    </div>

    <div v-else-if="errorMsg" class="text-sm text-danger">{{ errorMsg }}</div>

    <div v-else-if="items.length === 0">
      <UiCard>
        <div class="flex items-center gap-4 text-ink-muted">
          <Inbox class="h-6 w-6 text-ink-faint" />
          <div>
            <p class="font-medium text-ink">Nenhuma ação por enquanto.</p>
            <p class="mt-0.5 text-sm">A IA roda a cada hora. Quando algo precisar de atenção, aparece aqui.</p>
          </div>
        </div>
      </UiCard>
    </div>

    <div v-else class="space-y-3">
      <ActionCard
        v-for="item in items"
        :key="item.id"
        :action="item"
        @approve="onApprove"
        @reject="onReject"
        @revert="onRevert"
      />
    </div>
  </div>
</template>
