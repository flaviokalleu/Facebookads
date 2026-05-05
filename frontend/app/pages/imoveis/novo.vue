<script setup lang="ts">
import { ArrowLeft, X, Plus, Image as ImageIcon } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiInput from '~/components/ui/UiInput.vue'
import UiTextarea from '~/components/ui/UiTextarea.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import { useImoveis, type ImovelInput } from '~/composables/useImoveis'

const router = useRouter()
const { create } = useImoveis()
const toast = useToast()

const form = reactive<Partial<ImovelInput>>({
  nome: '',
  segmento: 'mcmv',
  cidade: '',
  bairro: '',
  preco_min: undefined,
  preco_max: undefined,
  quartos: undefined,
  area_m2: undefined,
  tipologia: 'apartamento',
  diferenciais: [],
  fotos: [],
  whatsapp_destino: '',
  link_landing: '',
  status: 'rascunho',
})

const novoDiferencial = ref('')
const novaFoto = ref('')
const saving = ref(false)
const errorMsg = ref<string | null>(null)

function addDiferencial() {
  const v = novoDiferencial.value.trim()
  if (!v) return
  form.diferenciais = [...(form.diferenciais || []), v]
  novoDiferencial.value = ''
}
function removeDiferencial(i: number) {
  form.diferenciais = (form.diferenciais || []).filter((_, idx) => idx !== i)
}
function addFoto() {
  const v = novaFoto.value.trim()
  if (!v) return
  form.fotos = [...(form.fotos || []), v]
  novaFoto.value = ''
}
function removeFoto(i: number) {
  form.fotos = (form.fotos || []).filter((_, idx) => idx !== i)
}

async function onSubmit() {
  if (!form.nome?.trim()) {
    errorMsg.value = 'Nome do imóvel é obrigatório.'
    return
  }
  saving.value = true
  errorMsg.value = null
  try {
    const created = await create(form)
    toast.success('Imóvel cadastrado', `"${created.nome}" salvo no catálogo.`)
    router.push(`/imoveis/${created.id}`)
  } catch (e: any) {
    const msg = e?.data?.error?.message || e?.message || 'Falha ao salvar.'
    errorMsg.value = msg
    toast.error('Não foi possível salvar', msg)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-5 max-w-3xl">
    <NuxtLink to="/imoveis" class="inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
      <ArrowLeft class="h-4 w-4" /> Voltar
    </NuxtLink>

    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-ink">Cadastrar imóvel</h1>
      <p class="text-sm text-ink-muted">A IA vai usar isso pra escrever anúncios específicos pra esse imóvel.</p>
    </div>

    <form class="space-y-5" @submit.prevent="onSubmit">
      <!-- Bloco 1: identificação -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Identificação</h2>
        <div class="mt-4 space-y-4">
          <UiInput v-model="form.nome" label="Nome do imóvel" placeholder="Ex: Residencial Bosque das Acácias" />

          <div class="grid gap-4 sm:grid-cols-2">
            <label class="block">
              <span class="mb-1.5 block text-sm font-medium text-ink">Segmento</span>
              <select
                v-model="form.segmento"
                class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none"
              >
                <option value="mcmv">MCMV (até R$ 300k)</option>
                <option value="medio">Médio padrão (R$ 300k–700k)</option>
                <option value="alto">Alto padrão (acima de R$ 700k)</option>
                <option value="comercial">Comercial</option>
                <option value="terreno">Terreno</option>
                <option value="lancamento">Lançamento</option>
              </select>
            </label>
            <label class="block">
              <span class="mb-1.5 block text-sm font-medium text-ink">Tipologia</span>
              <select
                v-model="form.tipologia"
                class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none"
              >
                <option value="apartamento">Apartamento</option>
                <option value="casa">Casa</option>
                <option value="terreno">Terreno</option>
                <option value="sala">Sala comercial</option>
                <option value="galpao">Galpão</option>
              </select>
            </label>
          </div>
        </div>
      </UiCard>

      <!-- Bloco 2: localização -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Localização</h2>
        <div class="mt-4 grid gap-4 sm:grid-cols-2">
          <UiInput v-model="form.cidade" label="Cidade" placeholder="Ex: Águas Lindas" />
          <UiInput v-model="form.bairro" label="Bairro (opcional)" placeholder="Ex: Centro" />
        </div>
      </UiCard>

      <!-- Bloco 3: características -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Características</h2>
        <div class="mt-4 grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Preço de — R$</span>
            <input
              v-model.number="form.preco_min"
              type="number"
              placeholder="180000"
              class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm tabular-nums text-ink focus:border-accent focus:shadow-focus focus:outline-none"
            />
          </label>
          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Preço até — R$</span>
            <input
              v-model.number="form.preco_max"
              type="number"
              placeholder="220000"
              class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm tabular-nums text-ink focus:border-accent focus:shadow-focus focus:outline-none"
            />
          </label>
          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Quartos</span>
            <input
              v-model.number="form.quartos"
              type="number"
              placeholder="2"
              class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm tabular-nums text-ink focus:border-accent focus:shadow-focus focus:outline-none"
            />
          </label>
          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Área (m²)</span>
            <input
              v-model.number="form.area_m2"
              type="number"
              step="0.1"
              placeholder="42.5"
              class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm tabular-nums text-ink focus:border-accent focus:shadow-focus focus:outline-none"
            />
          </label>
        </div>
      </UiCard>

      <!-- Bloco 4: diferenciais (chips) -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Diferenciais</h2>
        <p class="mt-1 text-xs text-ink-muted">Coisas que fazem esse imóvel se destacar — a IA vai citar nos anúncios.</p>

        <div class="mt-4 flex flex-wrap gap-2">
          <span
            v-for="(d, i) in form.diferenciais"
            :key="i"
            class="inline-flex items-center gap-1 rounded-full bg-accent-soft px-3 py-1 text-xs text-accent"
          >
            {{ d }}
            <button type="button" class="hover:text-accent-deep" @click="removeDiferencial(i)">
              <X class="h-3 w-3" />
            </button>
          </span>
        </div>

        <div class="mt-3 flex gap-2">
          <input
            v-model="novoDiferencial"
            placeholder="Ex: Lazer completo, Pet friendly, Vista mar..."
            class="flex-1 rounded-lg border border-border bg-bg px-3 py-2 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none"
            @keydown.enter.prevent="addDiferencial"
          />
          <UiButton variant="ghost" size="sm" type="button" @click="addDiferencial">
            <Plus class="h-4 w-4" /> Adicionar
          </UiButton>
        </div>
      </UiCard>

      <!-- Bloco 5: fotos (URLs por enquanto) -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Fotos</h2>
        <p class="mt-1 text-xs text-ink-muted">Cole URLs das fotos. Upload de arquivo vem em breve.</p>

        <div v-if="form.fotos?.length" class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div v-for="(f, i) in form.fotos" :key="i" class="relative aspect-square overflow-hidden rounded-lg bg-bg-muted">
            <img :src="f" class="h-full w-full object-cover" alt="" />
            <button
              type="button"
              class="absolute right-1 top-1 rounded-full bg-black/60 p-1 text-white hover:bg-black/80"
              @click="removeFoto(i)"
            >
              <X class="h-3 w-3" />
            </button>
          </div>
        </div>

        <div class="mt-3 flex gap-2">
          <input
            v-model="novaFoto"
            placeholder="https://..."
            class="flex-1 rounded-lg border border-border bg-bg px-3 py-2 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none"
            @keydown.enter.prevent="addFoto"
          />
          <UiButton variant="ghost" size="sm" type="button" @click="addFoto">
            <ImageIcon class="h-4 w-4" /> Adicionar
          </UiButton>
        </div>
      </UiCard>

      <!-- Bloco 6: contato + status -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Contato e status</h2>
        <div class="mt-4 space-y-4">
          <UiInput
            v-model="form.whatsapp_destino"
            label="WhatsApp (com DDD)"
            placeholder="+5561999999999"
            hint="Será o destino dos anúncios Click-to-WhatsApp."
          />
          <UiInput
            v-model="form.link_landing"
            label="Link de landing (opcional)"
            placeholder="https://..."
          />

          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Status</span>
            <select
              v-model="form.status"
              class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none"
            >
              <option value="rascunho">Rascunho — só pra eu organizar</option>
              <option value="ativo">Ativo — pode anunciar</option>
              <option value="pausado">Pausado — não anunciar agora</option>
              <option value="vendido">Vendido — não anunciar mais</option>
            </select>
          </label>
        </div>
      </UiCard>

      <UiBadge v-if="errorMsg" variant="danger">{{ errorMsg }}</UiBadge>

      <div class="flex justify-end gap-2">
        <NuxtLink to="/imoveis"><UiButton variant="ghost" type="button">Cancelar</UiButton></NuxtLink>
        <UiButton type="submit" variant="primary" :loading="saving">Salvar imóvel</UiButton>
      </div>
    </form>
  </div>
</template>
