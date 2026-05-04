<script setup lang="ts">
import { Building2, Wallet } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import type { BusinessManager, AdAccount } from '~/composables/useMetaTree'

defineProps<{
  businesses: BusinessManager[]
  personalAccounts?: AdAccount[]
}>()

function badgeVariant(status?: string): 'success' | 'warning' | 'neutral' {
  if (!status) return 'neutral'
  const s = status.toLowerCase()
  if (s.includes('verified')) return 'success'
  if (s.includes('pending')) return 'warning'
  return 'neutral'
}
</script>

<template>
  <div class="space-y-4">
    <UiCard v-for="bm in businesses" :key="bm.meta_id">
      <details class="group">
        <summary class="flex cursor-pointer list-none items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <Building2 class="h-5 w-5 text-accent" />
            <div>
              <p class="font-medium text-ink">{{ bm.name }}</p>
              <p class="text-xs text-ink-muted">Empresa</p>
            </div>
          </div>
          <UiBadge :variant="badgeVariant(bm.verification_status)">
            {{ bm.verification_status || 'sem verificação' }}
          </UiBadge>
        </summary>
        <ul class="mt-4 space-y-2 border-t border-border pt-4">
          <li v-for="acc in bm.accounts || []" :key="acc.meta_id">
            <NuxtLink
              :to="`/contas/${acc.meta_id}`"
              class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-bg-muted"
            >
              <div class="flex items-center gap-3">
                <Wallet class="h-4 w-4 text-ink-muted" />
                <span class="text-sm text-ink">{{ acc.name }}</span>
              </div>
              <span class="text-xs text-ink-faint">{{ acc.currency || '' }}</span>
            </NuxtLink>
          </li>
          <li v-if="!(bm.accounts && bm.accounts.length)" class="text-sm text-ink-muted px-3">
            Nenhuma conta de anúncio nesta empresa.
          </li>
        </ul>
      </details>
    </UiCard>

    <UiCard v-if="personalAccounts && personalAccounts.length">
      <p class="mb-3 font-medium text-ink">Contas pessoais</p>
      <ul class="space-y-2">
        <li v-for="acc in personalAccounts" :key="acc.meta_id">
          <NuxtLink
            :to="`/contas/${acc.meta_id}`"
            class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-bg-muted"
          >
            <div class="flex items-center gap-3">
              <Wallet class="h-4 w-4 text-ink-muted" />
              <span class="text-sm text-ink">{{ acc.name }}</span>
            </div>
            <span class="text-xs text-ink-faint">{{ acc.currency || '' }}</span>
          </NuxtLink>
        </li>
      </ul>
    </UiCard>
  </div>
</template>
