<script setup lang="ts">
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import { ShieldCheck } from 'lucide-vue-next'

const daysLeft = ref(60)
const totalDays = 60
const pct = computed(() => Math.max(0, Math.min(100, (daysLeft.value / totalDays) * 100)))
const variant = computed<'success' | 'warning' | 'danger'>(() => {
  if (daysLeft.value > 14) return 'success'
  if (daysLeft.value > 7) return 'warning'
  return 'danger'
})
const barColor = computed(() => {
  if (variant.value === 'success') return 'bg-success'
  if (variant.value === 'warning') return 'bg-warning'
  return 'bg-danger'
})
</script>

<template>
  <div class="max-w-2xl space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-ink">Acesso Meta</h1>
      <p class="text-sm text-ink-muted">Saúde do seu token de acesso à plataforma Meta.</p>
    </div>

    <UiCard>
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-start gap-3">
          <ShieldCheck class="mt-0.5 h-5 w-5 text-accent" />
          <div>
            <p class="font-medium text-ink">Token ativo</p>
            <p class="mt-1 text-sm text-ink-muted">
              Seu token de longa duração está válido.
            </p>
          </div>
        </div>
        <UiBadge :variant="variant">
          {{ daysLeft }} dias restantes
        </UiBadge>
      </div>

      <div class="mt-6">
        <div class="h-2 overflow-hidden rounded-full bg-bg-muted">
          <div :class="['h-full rounded-full transition-all', barColor]" :style="{ width: `${pct}%` }" />
        </div>
        <div class="mt-2 flex justify-between text-xs text-ink-muted">
          <span>0 dias</span>
          <span>{{ totalDays }} dias</span>
        </div>
      </div>

      <div class="mt-6 flex justify-end">
        <UiButton variant="primary">Renovar agora</UiButton>
      </div>
    </UiCard>
  </div>
</template>
