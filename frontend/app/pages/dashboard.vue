<script setup lang="ts">
import KpiCard from '~/components/ui/KpiCard.vue'
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'
import BMTree from '~/components/tree/BMTree.vue'
import { useMetaTree, type MetaTree } from '~/composables/useMetaTree'

const { fetchTree } = useMetaTree()
const tree = ref<MetaTree>({ businesses: [], personal_accounts: [] })
const loading = ref(true)

onMounted(async () => {
  try {
    tree.value = await fetchTree()
  } finally {
    loading.value = false
  }
})

const hasData = computed(() =>
  (tree.value.businesses?.length || 0) + (tree.value.personal_accounts?.length || 0) > 0,
)
</script>

<template>
  <div class="space-y-8">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-ink">Painel</h1>
      <p class="text-sm text-ink-muted">Visão geral dos seus anúncios hoje.</p>
    </div>

    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <KpiCard label="Gasto hoje" value="R$ 0,00" hint="Total investido nas suas contas no dia atual." />
      <KpiCard label="Contatos no WhatsApp" value="0" hint="Pessoas que clicaram para conversar via WhatsApp." />
      <KpiCard label="Custo por contato" value="R$ 0,00" hint="Quanto custa, em média, cada contato no WhatsApp." />
      <KpiCard label="Anúncios ativos" value="0" hint="Anúncios atualmente publicados e rodando." />
    </div>

    <section>
      <h2 class="mb-4 text-lg font-semibold text-ink">Suas empresas e contas</h2>

      <UiCard v-if="loading">
        <p class="text-sm text-ink-muted">Carregando árvore...</p>
      </UiCard>

      <UiCard v-else-if="!hasData">
        <div class="text-center">
          <p class="text-ink">Nenhuma empresa conectada ainda.</p>
          <p class="mt-1 text-sm text-ink-muted">
            Conecte sua conta Meta para começar a usar a IA.
          </p>
          <div class="mt-4 flex justify-center">
            <NuxtLink to="/onboarding">
              <UiButton variant="primary">Conectar conta</UiButton>
            </NuxtLink>
          </div>
        </div>
      </UiCard>

      <BMTree
        v-else
        :businesses="tree.businesses"
        :personal-accounts="tree.personal_accounts"
      />
    </section>
  </div>
</template>
