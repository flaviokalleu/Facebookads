<script setup lang="ts">
import { Plus, Search, Building2, MapPin, BedDouble, Ruler } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiInput from '~/components/ui/UiInput.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import { useImoveis, type Imovel, SEGMENTO_LABELS, STATUS_LABELS } from '~/composables/useImoveis'

const { list } = useImoveis()
const items = ref<Imovel[]>([])
const loading = ref(true)
const segmentoFilter = ref<'all' | Imovel['segmento']>('all')
const statusFilter = ref<'all' | Imovel['status']>('all')
const searchQuery = ref('')

async function load() {
  loading.value = true
  try {
    items.value = await list()
  } finally {
    loading.value = false
  }
}
onMounted(load)

const segmentTabs: { value: 'all' | Imovel['segmento']; label: string }[] = [
  { value: 'all', label: 'Todos' },
  { value: 'mcmv', label: 'MCMV' },
  { value: 'medio', label: 'Médio' },
  { value: 'alto', label: 'Alto padrão' },
  { value: 'comercial', label: 'Comercial' },
  { value: 'terreno', label: 'Terrenos' },
  { value: 'lancamento', label: 'Lançamentos' },
]

const filtered = computed(() => {
  return items.value.filter((im) => {
    if (segmentoFilter.value !== 'all' && im.segmento !== segmentoFilter.value) return false
    if (statusFilter.value !== 'all' && im.status !== statusFilter.value) return false
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      return im.nome.toLowerCase().includes(q) || im.cidade.toLowerCase().includes(q)
    }
    return true
  })
})

function brl(v?: number) {
  if (v == null) return '—'
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 }).format(v)
}
function priceRange(im: Imovel) {
  if (im.preco_min && im.preco_max && im.preco_min !== im.preco_max) {
    return `${brl(im.preco_min)} – ${brl(im.preco_max)}`
  }
  return brl(im.preco_min || im.preco_max)
}
function statusBadge(s: Imovel['status']): 'success' | 'warning' | 'neutral' | 'danger' {
  return s === 'ativo' ? 'success' : s === 'pausado' ? 'warning' : s === 'vendido' ? 'neutral' : 'neutral'
}
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-ink">Imóveis</h1>
        <p class="text-sm text-ink-muted">Catálogo dos imóveis que você está vendendo. A IA usa pra criar copy específica.</p>
      </div>
      <NuxtLink to="/imoveis/novo">
        <UiButton variant="primary"><Plus class="h-4 w-4" /> Cadastrar imóvel</UiButton>
      </NuxtLink>
    </div>

    <!-- Tabs de segmento -->
    <div class="flex flex-wrap gap-2">
      <button
        v-for="t in segmentTabs"
        :key="t.value"
        type="button"
        :class="[
          'rounded-full px-3 py-1.5 text-sm transition',
          segmentoFilter === t.value
            ? 'bg-accent text-white'
            : 'border border-border bg-bg text-ink-muted hover:text-ink',
        ]"
        @click="segmentoFilter = t.value"
      >
        {{ t.label }}
      </button>
    </div>

    <!-- Status + busca -->
    <div class="flex flex-wrap items-center gap-3">
      <select
        v-model="statusFilter"
        class="rounded-lg border border-border bg-bg px-3 py-2 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none"
      >
        <option value="all">Todos os status</option>
        <option value="rascunho">Rascunho</option>
        <option value="ativo">Ativo</option>
        <option value="pausado">Pausado</option>
        <option value="vendido">Vendido</option>
      </select>
      <div class="flex-1 max-w-sm relative">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-faint" />
        <input
          v-model="searchQuery"
          placeholder="Buscar por nome ou cidade..."
          class="w-full rounded-lg border border-border bg-bg pl-9 pr-3 py-2 text-sm text-ink placeholder:text-ink-faint focus:border-accent focus:shadow-focus focus:outline-none"
        />
      </div>
      <p class="ml-auto text-xs text-ink-muted">{{ filtered.length }} {{ filtered.length === 1 ? 'imóvel' : 'imóveis' }}</p>
    </div>

    <!-- Estado de carregamento / vazio -->
    <div v-if="loading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <UiSkeleton v-for="i in 3" :key="i" class="h-48" />
    </div>
    <UiCard v-else-if="!items.length" class="!p-8">
      <div class="text-center">
        <Building2 class="mx-auto h-10 w-10 text-ink-faint" />
        <h2 class="mt-3 text-lg font-semibold text-ink">Nenhum imóvel cadastrado ainda</h2>
        <p class="mt-1 text-sm text-ink-muted max-w-md mx-auto">
          Cadastre os imóveis que você está vendendo. A IA vai usar essas informações pra escrever
          anúncios específicos pra cada um.
        </p>
        <div class="mt-6">
          <NuxtLink to="/imoveis/novo">
            <UiButton variant="primary"><Plus class="h-4 w-4" /> Cadastrar primeiro imóvel</UiButton>
          </NuxtLink>
        </div>
      </div>
    </UiCard>
    <UiCard v-else-if="!filtered.length" class="!p-6">
      <p class="text-sm text-ink-muted">Nenhum imóvel encontrado com esses filtros.</p>
    </UiCard>

    <!-- Grade -->
    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink
        v-for="im in filtered"
        :key="im.id"
        :to="`/imoveis/${im.id}`"
        class="group block"
      >
        <UiCard class="h-full transition group-hover:shadow-md">
          <div v-if="im.fotos?.[0]" class="-mx-6 -mt-6 mb-4 aspect-[16/9] overflow-hidden rounded-t-xl bg-bg-muted">
            <img :src="im.fotos[0]" :alt="im.nome" class="h-full w-full object-cover" />
          </div>
          <div v-else class="-mx-6 -mt-6 mb-4 flex aspect-[16/9] items-center justify-center rounded-t-xl bg-bg-muted">
            <Building2 class="h-10 w-10 text-ink-faint" />
          </div>

          <div class="flex items-start justify-between gap-2">
            <h3 class="font-semibold text-ink truncate">{{ im.nome }}</h3>
            <UiBadge :variant="statusBadge(im.status)">{{ STATUS_LABELS[im.status] }}</UiBadge>
          </div>
          <p class="mt-1 text-xs text-ink-muted">
            <UiBadge variant="neutral">{{ SEGMENTO_LABELS[im.segmento] }}</UiBadge>
          </p>

          <p class="mt-3 text-lg font-semibold text-ink tabular-nums">{{ priceRange(im) }}</p>

          <div class="mt-3 flex flex-wrap gap-x-3 gap-y-1 text-xs text-ink-muted">
            <span v-if="im.cidade" class="flex items-center gap-1"><MapPin class="h-3 w-3" />{{ im.cidade }}<span v-if="im.bairro">, {{ im.bairro }}</span></span>
            <span v-if="im.quartos" class="flex items-center gap-1"><BedDouble class="h-3 w-3" />{{ im.quartos }} {{ im.quartos === 1 ? 'quarto' : 'quartos' }}</span>
            <span v-if="im.area_m2" class="flex items-center gap-1"><Ruler class="h-3 w-3" />{{ im.area_m2 }} m²</span>
          </div>
        </UiCard>
      </NuxtLink>
    </div>
  </div>
</template>
