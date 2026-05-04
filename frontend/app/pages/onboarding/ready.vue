<script setup lang="ts">
import { CheckCircle2, Loader2 } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import OnboardingProgress from '~/components/onboarding/OnboardingProgress.vue'
import { useMetaTree } from '~/composables/useMetaTree'

definePageMeta({ layout: 'blank' })

const { fetchTree } = useMetaTree()

const steps = ref([
  { label: 'Validando acesso', done: false },
  { label: 'Localizando empresas', done: false },
  { label: 'Mapeando contas de anúncio', done: false },
])

let interval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  steps.value[0].done = true
  interval = setInterval(async () => {
    const tree = await fetchTree()
    steps.value[1].done = true
    if ((tree.businesses && tree.businesses.length) || (tree.personal_accounts && tree.personal_accounts.length)) {
      steps.value[2].done = true
      if (interval) clearInterval(interval)
      setTimeout(() => navigateTo('/dashboard'), 600)
    }
  }, 1500)
})

onBeforeUnmount(() => {
  if (interval) clearInterval(interval)
})
</script>

<template>
  <div class="min-h-screen bg-bg">
    <OnboardingProgress :step="3" :total="3" />
    <section class="mx-auto max-w-xl px-6 py-12">
      <h1 class="text-3xl font-semibold tracking-tight text-ink">
        Estamos preparando seu painel.
      </h1>
      <p class="mt-3 text-ink-muted">
        Isso leva alguns segundos. Você pode aguardar nesta tela.
      </p>

      <UiCard class="mt-8">
        <ul class="space-y-3">
          <li
            v-for="(s, i) in steps"
            :key="i"
            class="flex items-center gap-3 text-sm"
          >
            <CheckCircle2 v-if="s.done" class="h-5 w-5 text-success" />
            <Loader2 v-else class="h-5 w-5 animate-spin text-ink-faint" />
            <span :class="s.done ? 'text-ink' : 'text-ink-muted'">{{ s.label }}</span>
          </li>
        </ul>
      </UiCard>

      <p class="mt-6 text-center text-sm text-ink-muted">
        Você pode <NuxtLink to="/dashboard" class="text-accent hover:underline">ir para o painel</NuxtLink> a qualquer momento.
      </p>
    </section>
  </div>
</template>
