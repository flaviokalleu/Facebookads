<script setup lang="ts">
import {
  ArrowLeft, Building2, MapPin, BedDouble, Ruler, MessageCircle, ExternalLink,
  Edit2, Trash2, Save, X, Plus, Image as ImageIcon, Loader2, AlertCircle,
} from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiInput from '~/components/ui/UiInput.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiSkeleton from '~/components/ui/UiSkeleton.vue'
import { useImoveis, type Imovel, SEGMENTO_LABELS, TIPOLOGIA_LABELS, STATUS_LABELS } from '~/composables/useImoveis'

const route = useRoute()
const router = useRouter()
const { get, update, remove } = useImoveis()
const toast = useToast()

const item = ref<Imovel | null>(null)
const loading = ref(true)
const editing = ref(false)
const saving = ref(false)
const errorMsg = ref<string | null>(null)

const form = reactive<Partial<Imovel>>({})
const novoDiferencial = ref('')
const novaFoto = ref('')

async function load() {
  loading.value = true
  const im = await get(String(route.params.id))
  if (im) {
    item.value = im
    Object.assign(form, im)
  }
  loading.value = false
}
onMounted(load)

function startEdit() {
  if (!item.value) return
  Object.assign(form, item.value)
  editing.value = true
}
function cancelEdit() {
  editing.value = false
  if (item.value) Object.assign(form, item.value)
}
async function saveEdit() {
  if (!item.value) return
  saving.value = true
  errorMsg.value = null
  try {
    const updated = await update(item.value.id, form as any)
    item.value = updated
    Object.assign(form, updated)
    editing.value = false
    toast.success('Alterações salvas')
  } catch (e: any) {
    const msg = e?.data?.error?.message || e?.message || 'Falha ao salvar.'
    errorMsg.value = msg
    toast.error('Não foi possível salvar', msg)
  } finally {
    saving.value = false
  }
}
async function onDelete() {
  if (!item.value) return
  if (!confirm(`Apagar "${item.value.nome}"? Não dá pra desfazer.`)) return
  const name = item.value.nome
  try {
    await remove(item.value.id)
    toast.success('Imóvel apagado', `"${name}" foi removido do catálogo.`)
    router.push('/imoveis')
  } catch (e: any) {
    const msg = e?.data?.error?.message || e?.message || 'Falha ao apagar.'
    errorMsg.value = msg
    toast.error('Não foi possível apagar', msg)
  }
}
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
  return s === 'ativo' ? 'success' : s === 'pausado' ? 'warning' : 'neutral'
}
</script>

<template>
  <div class="space-y-5 max-w-4xl">
    <NuxtLink to="/imoveis" class="inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
      <ArrowLeft class="h-4 w-4" /> Voltar para imóveis
    </NuxtLink>

    <div v-if="loading" class="space-y-4">
      <UiSkeleton class="h-32" />
      <UiSkeleton class="h-48" />
    </div>

    <UiCard v-else-if="!item" class="!p-8 text-center">
      <AlertCircle class="mx-auto h-8 w-8 text-ink-faint" />
      <p class="mt-2 text-ink">Imóvel não encontrado.</p>
    </UiCard>

    <template v-else>
      <!-- Header -->
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="flex items-start gap-4">
          <div class="rounded-xl bg-accent-soft p-3 text-accent">
            <Building2 class="h-7 w-7" />
          </div>
          <div>
            <p class="text-xs uppercase tracking-wider text-ink-faint">{{ SEGMENTO_LABELS[item.segmento] }}</p>
            <h1 class="mt-0.5 text-2xl font-semibold tracking-tight text-ink">{{ item.nome }}</h1>
            <p v-if="item.cidade" class="mt-1 text-sm text-ink-muted flex items-center gap-1">
              <MapPin class="h-3.5 w-3.5" /> {{ item.cidade }}<span v-if="item.bairro">, {{ item.bairro }}</span>
            </p>
            <div class="mt-2 flex flex-wrap gap-2">
              <UiBadge :variant="statusBadge(item.status)">{{ STATUS_LABELS[item.status] }}</UiBadge>
              <UiBadge v-if="item.tipologia" variant="neutral">{{ TIPOLOGIA_LABELS[item.tipologia] }}</UiBadge>
            </div>
          </div>
        </div>
        <div class="flex gap-2">
          <UiButton v-if="!editing" variant="ghost" size="sm" @click="startEdit">
            <Edit2 class="h-4 w-4" /> Editar
          </UiButton>
          <UiButton v-if="!editing" variant="danger" size="sm" @click="onDelete">
            <Trash2 class="h-4 w-4" /> Apagar
          </UiButton>
          <UiButton v-if="editing" variant="ghost" size="sm" @click="cancelEdit">
            <X class="h-4 w-4" /> Cancelar
          </UiButton>
          <UiButton v-if="editing" variant="primary" size="sm" :loading="saving" @click="saveEdit">
            <Save class="h-4 w-4" /> Salvar
          </UiButton>
        </div>
      </div>

      <!-- Galeria -->
      <UiCard v-if="item.fotos?.length || editing" class="!p-4">
        <div v-if="(item.fotos?.length && !editing)" class="grid gap-3 grid-cols-2 sm:grid-cols-3 lg:grid-cols-4">
          <div v-for="(f, i) in item.fotos" :key="i" class="aspect-square overflow-hidden rounded-lg bg-bg-muted">
            <img :src="f" :alt="`${item.nome} foto ${i + 1}`" class="h-full w-full object-cover" />
          </div>
        </div>
        <div v-else-if="editing">
          <h3 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Fotos</h3>
          <div v-if="form.fotos?.length" class="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
            <div v-for="(f, i) in form.fotos" :key="i" class="relative aspect-square overflow-hidden rounded-lg bg-bg-muted">
              <img :src="f" class="h-full w-full object-cover" alt="" />
              <button type="button" class="absolute right-1 top-1 rounded-full bg-black/60 p-1 text-white" @click="removeFoto(i)">
                <X class="h-3 w-3" />
              </button>
            </div>
          </div>
          <div class="mt-3 flex gap-2">
            <input
              v-model="novaFoto"
              placeholder="URL da foto"
              class="flex-1 rounded-lg border border-border bg-bg px-3 py-2 text-sm focus:border-accent focus:shadow-focus focus:outline-none"
              @keydown.enter.prevent="addFoto"
            />
            <UiButton variant="ghost" size="sm" @click="addFoto"><ImageIcon class="h-4 w-4" /></UiButton>
          </div>
        </div>
      </UiCard>

      <!-- Características -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Características</h2>
        <div v-if="!editing" class="mt-4 grid gap-4 sm:grid-cols-2">
          <div>
            <p class="text-xs text-ink-faint">Faixa de preço</p>
            <p class="mt-1 text-xl font-semibold tabular-nums text-ink">{{ priceRange(item) }}</p>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <p class="text-xs text-ink-faint">Quartos</p>
              <p class="mt-1 text-base font-medium tabular-nums text-ink">
                <BedDouble class="inline h-4 w-4 mr-1 text-ink-muted" />
                {{ item.quartos ?? '—' }}
              </p>
            </div>
            <div>
              <p class="text-xs text-ink-faint">Área</p>
              <p class="mt-1 text-base font-medium tabular-nums text-ink">
                <Ruler class="inline h-4 w-4 mr-1 text-ink-muted" />
                {{ item.area_m2 ? `${item.area_m2} m²` : '—' }}
              </p>
            </div>
          </div>
        </div>

        <div v-else class="mt-4 grid gap-4 sm:grid-cols-2">
          <UiInput v-model="form.nome" label="Nome" />
          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Segmento</span>
            <select v-model="form.segmento" class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none">
              <option value="mcmv">MCMV</option>
              <option value="medio">Médio padrão</option>
              <option value="alto">Alto padrão</option>
              <option value="comercial">Comercial</option>
              <option value="terreno">Terreno</option>
              <option value="lancamento">Lançamento</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Tipologia</span>
            <select v-model="form.tipologia" class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none">
              <option value="apartamento">Apartamento</option>
              <option value="casa">Casa</option>
              <option value="terreno">Terreno</option>
              <option value="sala">Sala</option>
              <option value="galpao">Galpão</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1.5 block text-sm font-medium text-ink">Status</span>
            <select v-model="form.status" class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-ink focus:border-accent focus:shadow-focus focus:outline-none">
              <option value="rascunho">Rascunho</option>
              <option value="ativo">Ativo</option>
              <option value="pausado">Pausado</option>
              <option value="vendido">Vendido</option>
            </select>
          </label>
          <UiInput v-model="form.cidade" label="Cidade" />
          <UiInput v-model="form.bairro" label="Bairro" />
          <label class="block"><span class="mb-1.5 block text-sm font-medium text-ink">Preço de — R$</span><input v-model.number="form.preco_min" type="number" class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm" /></label>
          <label class="block"><span class="mb-1.5 block text-sm font-medium text-ink">Preço até — R$</span><input v-model.number="form.preco_max" type="number" class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm" /></label>
          <label class="block"><span class="mb-1.5 block text-sm font-medium text-ink">Quartos</span><input v-model.number="form.quartos" type="number" class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm" /></label>
          <label class="block"><span class="mb-1.5 block text-sm font-medium text-ink">Área (m²)</span><input v-model.number="form.area_m2" type="number" step="0.1" class="block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm" /></label>
        </div>
      </UiCard>

      <!-- Diferenciais -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Diferenciais</h2>
        <div v-if="!editing" class="mt-3 flex flex-wrap gap-2">
          <span v-if="!item.diferenciais?.length" class="text-sm text-ink-muted">Sem diferenciais cadastrados.</span>
          <span v-for="(d, i) in item.diferenciais" :key="i" class="inline-flex items-center rounded-full bg-accent-soft px-3 py-1 text-xs text-accent">
            {{ d }}
          </span>
        </div>
        <div v-else>
          <div class="mt-3 flex flex-wrap gap-2">
            <span v-for="(d, i) in form.diferenciais" :key="i" class="inline-flex items-center gap-1 rounded-full bg-accent-soft px-3 py-1 text-xs text-accent">
              {{ d }}
              <button type="button" @click="removeDiferencial(i)"><X class="h-3 w-3" /></button>
            </span>
          </div>
          <div class="mt-3 flex gap-2">
            <input v-model="novoDiferencial" placeholder="Ex: Lazer completo" class="flex-1 rounded-lg border border-border bg-bg px-3 py-2 text-sm focus:border-accent focus:shadow-focus focus:outline-none" @keydown.enter.prevent="addDiferencial" />
            <UiButton variant="ghost" size="sm" @click="addDiferencial"><Plus class="h-4 w-4" /></UiButton>
          </div>
        </div>
      </UiCard>

      <!-- Contato -->
      <UiCard>
        <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">Contato</h2>
        <div v-if="!editing" class="mt-3 space-y-2 text-sm">
          <div v-if="item.whatsapp_destino" class="flex items-center gap-2 text-ink">
            <MessageCircle class="h-4 w-4 text-success" />
            <a :href="`https://wa.me/${item.whatsapp_destino.replace(/\\D/g,'')}`" target="_blank" rel="noopener" class="text-accent hover:underline">{{ item.whatsapp_destino }}</a>
          </div>
          <div v-if="item.link_landing" class="flex items-center gap-2 text-ink">
            <ExternalLink class="h-4 w-4 text-ink-muted" />
            <a :href="item.link_landing" target="_blank" rel="noopener" class="text-accent hover:underline truncate">{{ item.link_landing }}</a>
          </div>
          <p v-if="!item.whatsapp_destino && !item.link_landing" class="text-ink-muted">Nenhum contato cadastrado.</p>
        </div>
        <div v-else class="mt-3 space-y-3">
          <UiInput v-model="form.whatsapp_destino" label="WhatsApp (com DDD)" placeholder="+5561999999999" />
          <UiInput v-model="form.link_landing" label="Link de landing" placeholder="https://..." />
        </div>
      </UiCard>

      <UiBadge v-if="errorMsg" variant="danger">{{ errorMsg }}</UiBadge>
    </template>
  </div>
</template>
