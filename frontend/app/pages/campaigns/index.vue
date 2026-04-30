<script setup lang="ts">
import { useCampaignStore } from '~/stores/useCampaignStore'
import { useCampaignFilters } from '~/composables/useCampaignFilters'
import { useDateRange } from '~/composables/useDateRange'
import { api } from '~/lib/api'
import { Megaphone, RefreshCw, Search, X, Check, PenLine, Trash2, Link2, ExternalLink, LogIn, Plus, Sparkles, WandSparkles } from 'lucide-vue-next'

const store = useCampaignStore()
const { search, healthFilter, sortBy, sortDir, HEALTH_OPTIONS, applyFilters } = useCampaignFilters()
const { preset, DATE_PRESETS } = useDateRange('last_7d')

onMounted(async () => {
  await store.fetchMetaStatus()
  store.fetchAll(preset.value)
})
watch(preset, v => store.fetchAll(v))

const displayed = computed(() => applyFilters(store.campaigns as any[]))

// ─── Connect Meta Modal ─────────────────────────────────────────
const showConnect = ref(false)
const metaToken = ref('')
const metaAccountId = ref('')
const connecting = ref(false)
const connectError = ref('')
const connectSuccess = ref(false)
const availableAccounts = ref<{ id: string; name: string }[]>([])
const fetchingAccounts = ref(false)

async function fetchAccounts() {
  if (!metaToken.value.trim()) return
  fetchingAccounts.value = true
  connectError.value = ''
  try {
    const accounts = await api.auth.listAdAccounts(metaToken.value)
    availableAccounts.value = (accounts || []).map((a: any) => ({
      id: a.id.replace('act_', ''),
      name: a.name
    }))
    if (availableAccounts.value.length === 0) {
      connectError.value = 'No ad accounts found for this token'
    } else if (availableAccounts.value.length === 1) {
      metaAccountId.value = availableAccounts.value[0].id
    }
  } catch (e: any) {
    connectError.value = e.message || 'Failed to fetch ad accounts'
  } finally {
    fetchingAccounts.value = false
  }
}

async function connectMeta() {
  if (!metaToken.value.trim() || !metaAccountId.value.trim()) return
  connecting.value = true
  connectError.value = ''
  try {
    await api.auth.connectMeta({ access_token: metaToken.value, ad_account_id: metaAccountId.value })
    connectSuccess.value = true
    // Auto-sync after connecting
    await store.sync()
    setTimeout(() => {
      showConnect.value = false
      connectSuccess.value = false
      metaToken.value = ''
      metaAccountId.value = ''
      availableAccounts.value = []
    }, 1500)
  } catch (e: any) {
    connectError.value = e.message || 'Connection failed'
  } finally {
    connecting.value = false
  }
}

// ─── Inline editing ────────────────────────────────────────────
const editingId = ref<string | null>(null)
const editName = ref('')
const editBudget = ref(0)
const editStatus = ref('')

function startEdit(c: any) {
  editingId.value = c.id
  editName.value = c.name
  editBudget.value = c.daily_budget ?? 0
  editStatus.value = c.status
}

async function saveEdit(id: string) {
  try {
    await api.campaigns.update(id, {
      name: editName.value,
      daily_budget: editBudget.value || undefined,
      status: editStatus.value,
    })
    await store.fetchAll(preset.value)
    editingId.value = null
  } catch (e: any) {
    console.error('Failed to update campaign', e)
  }
}

async function deleteCampaign(id: string) {
  if (!confirm('Are you sure you want to delete this campaign?')) return
  try {
    await api.campaigns.delete(id)
    await store.fetchAll(preset.value)
  } catch (e: any) {
    console.error('Failed to delete campaign', e)
  }
}

function cancelEdit() {
  editingId.value = null
}

// Templates por Nicho
const templates = [
  { name: 'Imobiliária', icon: '🏠', objective: 'OUTCOME_LEADS', minAge: 25, maxAge: 50, gender: 'all', interests: 'imóveis, casa própria, financiamento imobiliário, apartamento' },
  { name: 'Educação', icon: '📚', objective: 'OUTCOME_LEADS', minAge: 18, maxAge: 40, gender: 'all', interests: 'cursos online, educação, faculdade, pós-graduação' },
  { name: 'Moda e Vestuário', icon: '👗', objective: 'OUTCOME_TRAFFIC', minAge: 18, maxAge: 45, gender: 'female', interests: 'moda feminina, roupas, acessórios, tendências' },
  { name: 'Saúde e Beleza', icon: '💄', objective: 'OUTCOME_ENGAGEMENT', minAge: 20, maxAge: 55, gender: 'female', interests: 'beleza, cuidados com a pele, maquiagem, estética' },
  { name: 'Alimentação', icon: '🍕', objective: 'OUTCOME_TRAFFIC', minAge: 18, maxAge: 60, gender: 'all', interests: 'gastronomia, delivery, culinária, restaurante' },
  { name: 'Academia e Fitness', icon: '💪', objective: 'OUTCOME_LEADS', minAge: 18, maxAge: 45, gender: 'all', interests: 'academia, fitness, emagrecimento, musculação' },
  { name: 'Automotivo', icon: '🚗', objective: 'OUTCOME_LEADS', minAge: 25, maxAge: 60, gender: 'male', interests: 'carros, motos, financiamento veicular, concessionária' },
  { name: 'Tecnologia', icon: '💻', objective: 'OUTCOME_TRAFFIC', minAge: 18, maxAge: 50, gender: 'male', interests: 'tecnologia, informática, celular, games' },
  { name: 'Viagens e Turismo', icon: '✈️', objective: 'OUTCOME_ENGAGEMENT', minAge: 20, maxAge: 55, gender: 'all', interests: 'viagens, turismo, hotel, passagem aérea' },
  { name: 'Serviços Financeiros', icon: '💰', objective: 'OUTCOME_LEADS', minAge: 25, maxAge: 60, gender: 'all', interests: 'financiamento, crédito, investimentos, banco' },
  { name: 'Pet Shop', icon: '🐶', objective: 'OUTCOME_TRAFFIC', minAge: 20, maxAge: 55, gender: 'female', interests: 'animais de estimação, pet shop, ração, veterinário' },
  { name: 'Decoração e Casa', icon: '🛋️', objective: 'OUTCOME_ENGAGEMENT', minAge: 25, maxAge: 55, gender: 'female', interests: 'decoração, móveis, casa própria, reforma' },
]

function applyTemplate(t: typeof templates[0]) {
  formName.value = `Campanha - ${t.name}`
  formObjective.value = t.objective
  formMinAge.value = t.minAge
  formMaxAge.value = t.maxAge
  formGender.value = t.gender
  formInterests.value = t.interests
  formBudget.value = 50
  formCities.value = [{ city: '', radius: 10 }]
  formCountry.value = 'BR'
  wizardStep.value = 2
}

// Wizard de Criação (estilo Facebook Ads)
const showCreateModal = ref(false)
const wizardStep = ref(1)
const creating = ref(false)
const createResult = ref<any>(null)
const createError = ref('')

// Step 1: Templates
const filteredTemplates = ref(templates)
const templateSearch = ref('')
watch(templateSearch, (v) => {
  if (!v) { filteredTemplates.value = templates; return }
  const q = v.toLowerCase()
  filteredTemplates.value = templates.filter(t => t.name.toLowerCase().includes(q) || t.interests.toLowerCase().includes(q))
})

// Step 2: Nome + Objetivo
const formName = ref('')
const formObjective = ref('OUTCOME_ENGAGEMENT')
const objectives = [
  { value: 'OUTCOME_ENGAGEMENT', label: 'Engajamento', desc: 'Mais curtidas, comentários e compartilhamentos' },
  { value: 'OUTCOME_LEADS', label: 'Leads', desc: 'Gerar cadastros e contatos' },
  { value: 'OUTCOME_TRAFFIC', label: 'Tráfego', desc: 'Visitas ao site ou WhatsApp' },
  { value: 'OUTCOME_SALES', label: 'Vendas', desc: 'Conversões e vendas online' },
  { value: 'OUTCOME_AWARENESS', label: 'Reconhecimento', desc: 'Alcance e visibilidade da marca' },
]

// Step 3: Público
const formMinAge = ref(18)
const formMaxAge = ref(65)
const formGender = ref('all') // all, male, female
const formInterests = ref('')

// Step 4: Localização
const formCities = ref([{ city: '', radius: 10, key: '' }])
const formCountry = ref('BR')
const locationSearch = ref('')
const locationResults = ref<{key: string; name: string; region: string}[]>([])
const searchingLocation = ref(false)
let searchTimeout: any = null

async function searchLocations(q: string) {
  if (!q || q.length < 2) { locationResults.value = []; return }
  searchingLocation.value = true
  try {
    const token = localStorage.getItem('auth_token')
    const res = await fetch(`/api/v1/locations/search?q=${encodeURIComponent(q)}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    const data = await res.json()
    locationResults.value = (data?.data || (data as any) || []).slice(0, 5)
  } catch { locationResults.value = [] }
  finally { searchingLocation.value = false }
}

function selectLocation(loc: {key: string; name: string; region: string}, idx: number) {
  formCities.value[idx].city = `${loc.name}${loc.region ? `, ${loc.region}` : ''}`
  formCities.value[idx].key = loc.key
  locationResults.value = []
  locationSearch.value = ''
}

function addCity() { formCities.value.push({ city: '', radius: 10, key: '' }) }
function removeCity(i: number) { if (formCities.value.length > 1) formCities.value.splice(i, 1) }

// Step 4: Orçamento
const formBudget = ref(50)
const formBudgetType = ref('daily') // daily, lifetime

// Step 5: Review + Create
async function createCampanha() {
  if (!formName.value) { createError.value = 'Dê um nome para a campanha'; return }
  creating.value = true
  createError.value = ''
  createResult.value = null
  try {
    // Build full prompt for the AI
    const validCities = formCities.value.filter(c => c.city)
    const citiesStr = validCities.map(c => `${c.city} (raio ${c.radius}km)`).join(', ')
    const cityDetails = JSON.stringify(validCities.map(c => ({ key: c.key || '', name: c.city.split(',')[0], radius: c.radius })))
    const result = await api.campaigns.createFull({
      name: formName.value,
      objective: formObjective.value,
      budget: formBudget.value,
      min_age: formMinAge.value,
      max_age: formMaxAge.value,
      gender: formGender.value,
      interests: formInterests.value,
      cities: citiesStr,
      country: formCountry.value,
      city_details: cityDetails,
    })
    createResult.value = result
    setTimeout(() => {
      showCreateModal.value = false
      wizardStep.value = 1
      formName.value = ''
      formBudget.value = 50
      formInterests.value = ''
      formCities.value = [{ city: '', radius: 10 }]
      createResult.value = null
      store.fetchAll(preset.value)
    }, 3000)
  } catch (e: any) {
    createError.value = e.message || 'Erro ao criar campanha'
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div class="flex items-center gap-2">
        <Megaphone class="w-5 h-5 text-blue-glow" />
        <h1 class="text-primary text-xl font-bold">Campaigns</h1>
        <span class="text-muted text-sm">({{ store.campaigns.length }})</span>
        <span
          v-if="store.metaConnected"
          class="text-2xs text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 rounded-full px-2 py-0.5 flex items-center gap-1"
        >
          <span class="w-1.5 h-1.5 rounded-full bg-emerald-400" />
          Meta connected
        </span>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="flex items-center gap-1.5 text-xs font-medium text-purple-400 hover:text-purple-300 bg-purple-500/10 hover:bg-purple-500/20 px-3 py-2 rounded-xl border border-purple-500/20 transition-all"
          @click="showCreateModal = true"
        >
          <Plus class="w-3.5 h-3.5" />
          Criar Campanha
        </button>
        <button
          class="flex items-center gap-1.5 text-xs font-medium text-emerald-400 hover:text-emerald-300 bg-emerald-500/10 hover:bg-emerald-500/20 px-3 py-2 rounded-xl border border-emerald-500/20 transition-all"
          @click="showConnect = true"
        >
          <Link2 class="w-3.5 h-3.5" />
          Conectar Meta
        </button>
        <button
          :disabled="store.syncing"
          class="flex items-center gap-2 bg-blue-default/20 hover:bg-blue-default/30 text-blue-bright text-sm font-medium px-4 py-2 rounded-xl transition-all disabled:opacity-50 border border-blue-default/20"
          @click="store.sync()"
        >
          <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': store.syncing }" />
          {{ store.syncing ? 'Sincronizando…' : 'Sincronizar' }}
        </button>
      </div>
    </div>

    <!-- Wizard Criar Campanha (estilo Facebook Ads) -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="showCreateModal = false">
          <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" />
          <div class="relative bg-bg-surface border border-bg-border/50 rounded-2xl shadow-2xl w-full max-w-xl max-h-[90vh] overflow-y-auto">

            <!-- Success State -->
            <div v-if="createResult" class="p-8 text-center">
              <div class="w-16 h-16 rounded-full bg-emerald-500/10 flex items-center justify-center mx-auto mb-4"><Check class="w-8 h-8 text-emerald-400" /></div>
              <p class="text-primary font-semibold text-lg mb-1">Campanha Criada! 🎉</p>
              <p class="text-muted text-sm mb-2">{{ createResult.campaign?.name }}</p>
              <p v-if="createResult.meta_campaign_id" class="text-emerald-400 text-xs">Campanha Meta: {{ createResult.meta_campaign_id.slice(0,12) }}...</p>
              <p v-if="createResult.meta_ad_set_id" class="text-emerald-400/70 text-2xs">Conjunto: {{ createResult.meta_ad_set_id.slice(0,12) }}...</p>
              <p v-if="createResult.meta_ad_id" class="text-emerald-400/50 text-2xs">Anúncio: {{ createResult.meta_ad_id.slice(0,12) }}...</p>
              <p class="text-muted text-xs mt-2">Criado no Meta Ads (PAUSED). Ative no Gerenciador.</p>
            </div>

            <template v-else>
              <!-- Header com Steps -->
              <div class="p-6 border-b border-bg-border/50">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-default to-blue-glow flex items-center justify-center">
                      <Megaphone class="w-5 h-5 text-white" />
                    </div>
                    <div>
                      <h2 class="text-primary font-bold text-lg">Nova Campanha</h2>
                      <p class="text-muted text-xs">Passo {{ wizardStep }} de 5</p>
                    </div>
                  </div>
                  <button @click="showCreateModal = false" class="p-1.5 rounded-lg text-muted hover:text-primary hover:bg-bg-elevated transition-colors"><X class="w-5 h-5" /></button>
                </div>
                <!-- Progress bar -->
                <div class="flex gap-1 mt-4">
                  <div v-for="s in 5" :key="s" class="h-1 flex-1 rounded-full transition-all" :class="s <= wizardStep ? 'bg-blue-default' : 'bg-bg-border'"></div>
                </div>
              </div>

              <div class="p-6 space-y-5">
                <!-- STEP 1: Templates por Nicho -->
                <template v-if="wizardStep === 1">
                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-1.5">Escolha um template ou personalize</label>
                    <input v-model="templateSearch" type="text" placeholder="Buscar nicho..." class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-4 py-2.5 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default mb-3" />
                    <div class="grid grid-cols-2 gap-2 max-h-64 overflow-y-auto pr-1">
                      <button v-for="t in filteredTemplates" :key="t.name"
                        class="flex items-center gap-2.5 p-3 rounded-xl border text-left transition-all hover:border-blue-default/40 bg-bg-elevated/30 border-bg-border/50"
                        @click="applyTemplate(t)">
                        <span class="text-xl">{{ t.icon }}</span>
                        <div class="min-w-0">
                          <p class="text-primary text-xs font-medium truncate">{{ t.name }}</p>
                          <p class="text-muted text-2xs truncate">{{ t.minAge }}-{{ t.maxAge }} anos · {{ { all: 'Todos', male: 'Homens', female: 'Mulheres' }[t.gender] }}</p>
                        </div>
                      </button>
                    </div>
                    <button class="w-full mt-3 py-2.5 rounded-xl border border-dashed border-bg-border text-secondary text-sm hover:text-primary hover:border-blue-default/40 transition-all" @click="wizardStep = 2">
                      + Configurar manualmente
                    </button>
                  </div>
                </template>

                <!-- STEP 2: Nome + Objetivo -->
                <template v-if="wizardStep === 5">
                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-1.5">Nome da campanha</label>
                    <input v-model="formName" type="text" placeholder="Ex: Imobiliária - Conversão WhatsApp" class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-4 py-3 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all" />
                  </div>

                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-2">Objetivo da campanha</label>
                    <div class="grid gap-2">
                      <button v-for="obj in objectives" :key="obj.value"
                        class="flex items-start gap-3 p-3 rounded-xl border text-left transition-all"
                        :class="formObjective === obj.value ? 'bg-blue-default/15 border-blue-default/40' : 'bg-bg-elevated/30 border-bg-border/50 hover:border-bg-border'"
                        @click="formObjective = obj.value">
                        <div class="w-5 h-5 rounded-full border-2 flex items-center justify-center mt-0.5 shrink-0" :class="formObjective === obj.value ? 'border-blue-default' : 'border-muted'">
                          <div v-if="formObjective === obj.value" class="w-2.5 h-2.5 rounded-full bg-blue-default"></div>
                        </div>
                        <div>
                          <p class="text-primary text-sm font-medium">{{ obj.label }}</p>
                          <p class="text-muted text-xs">{{ obj.desc }}</p>
                        </div>
                      </button>
                    </div>
                  </div>
                </template>

                <!-- STEP 2: Público -->
                <template v-if="wizardStep === 5">
                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-1.5">Idade</label>
                    <div class="flex items-center gap-3">
                      <input v-model.number="formMinAge" type="number" min="13" max="65" class="w-24 bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm text-center focus:outline-none focus:border-blue-default" />
                      <span class="text-muted">até</span>
                      <input v-model.number="formMaxAge" type="number" min="13" max="65" class="w-24 bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm text-center focus:outline-none focus:border-blue-default" />
                    </div>
                  </div>

                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-1.5">Gênero</label>
                    <div class="flex gap-2">
                      <button v-for="g in [{v:'all',l:'Todos'},{v:'male',l:'Homens'},{v:'female',l:'Mulheres'}]" :key="g.v"
                        class="flex-1 py-2.5 rounded-xl text-sm font-medium border transition-all"
                        :class="formGender === g.v ? 'bg-blue-default/15 text-blue-bright border-blue-default/40' : 'bg-bg-elevated/30 text-secondary border-bg-border/50 hover:border-bg-border'"
                        @click="formGender = g.v">{{ g.l }}</button>
                    </div>
                  </div>

                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-1.5">Interesses (palavras-chave)</label>
                    <input v-model="formInterests" type="text" placeholder="Ex: imóveis, casa própria, financiamento" class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-4 py-2.5 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default" />
                    <p class="text-muted text-2xs mt-1">Separe por vírgula. A IA encontra os interesses no Meta Ads.</p>
                  </div>
                </template>

                <!-- STEP 3: Localização -->
                <template v-if="wizardStep === 5">
                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-1.5">País</label>
                    <select v-model="formCountry" class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-4 py-2.5 text-primary text-sm focus:outline-none focus:border-blue-default">
                      <option value="BR">Brasil</option>
                      <option value="PT">Portugal</option>
                      <option value="US">Estados Unidos</option>
                    </select>
                  </div>

                  <div>
                    <div class="flex items-center justify-between mb-1.5">
                      <label class="text-secondary text-xs font-semibold">Cidades (com raio de alcance)</label>
                      <button @click="addCity" class="text-2xs text-blue-bright hover:underline">+ Adicionar cidade</button>
                    </div>
                    <div v-for="(city, i) in formCities" :key="i" class="flex items-center gap-2 mb-2 relative">
                      <div class="flex-1 relative">
                        <input v-model="city.city" @input="searchLocations(city.city)" type="text" placeholder="Digite o nome da cidade..." class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default" />
                        <div v-if="locationResults.length && formCities.indexOf(city) === formCities.length - 1" class="absolute top-full left-0 right-0 z-10 mt-1 bg-bg-surface border border-bg-border rounded-xl shadow-xl overflow-hidden">
                          <button v-for="loc in locationResults" :key="loc.key" @click="selectLocation(loc, i)" class="w-full flex items-center gap-2 px-3 py-2.5 text-left text-sm text-primary hover:bg-bg-elevated transition-colors border-b border-bg-border/50 last:border-0">
                            <span class="text-muted">📍</span>
                            <span>{{ loc.name }}</span>
                            <span v-if="loc.region" class="text-muted text-2xs">{{ loc.region }}</span>
                          </button>
                        </div>
                      </div>
                      <select v-model="city.radius" class="bg-bg-elevated/50 border border-bg-border rounded-lg px-2 py-2 text-primary text-xs focus:outline-none focus:border-blue-default">
                        <option :value="5">5 km</option>
                        <option :value="10">10 km</option>
                        <option :value="25">25 km</option>
                        <option :value="50">50 km</option>
                        <option :value="100">100 km</option>
                      </select>
                      <button v-if="formCities.length > 1" @click="removeCity(i)" class="p-1.5 text-muted hover:text-red-400 transition-colors"><X class="w-3.5 h-3.5" /></button>
                    </div>
                  </div>
                </template>

                <!-- STEP 4: Orçamento -->
                <template v-if="wizardStep === 5">
                  <div>
                    <label class="text-secondary text-xs font-semibold block mb-1.5">Orçamento</label>
                    <div class="flex gap-2 mb-3">
                      <button v-for="t in [{v:'daily',l:'Diário'},{v:'lifetime',l:'Total'}]" :key="t.v"
                        class="flex-1 py-2 rounded-xl text-sm font-medium border transition-all"
                        :class="formBudgetType === t.v ? 'bg-blue-default/15 text-blue-bright border-blue-default/40' : 'bg-bg-elevated/30 text-secondary border-bg-border/50 hover:border-bg-border'"
                        @click="formBudgetType = t.v">{{ t.l }}</button>
                    </div>
                    <div class="relative">
                      <span class="absolute left-4 top-1/2 -translate-y-1/2 text-primary font-semibold text-sm">R$</span>
                      <input v-model.number="formBudget" type="number" min="1" class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl pl-10 pr-4 py-3 text-primary text-lg font-bold font-mono focus:outline-none focus:border-blue-default" />
                    </div>
                    <p class="text-muted text-2xs mt-1.5">{{ formBudgetType === 'daily' ? 'Valor gasto por dia' : 'Valor total da campanha' }}</p>
                  </div>

                  <!-- Resumo -->
                  <div class="bg-bg-elevated/30 rounded-xl p-4 border border-bg-border/50 space-y-2">
                    <p class="text-xs font-semibold text-primary">Resumo da Campanha</p>
                    <div class="text-xs text-secondary space-y-1">
                      <p><span class="text-muted">Nome:</span> {{ formName || '—' }}</p>
                      <p><span class="text-muted">Objetivo:</span> {{ objectives.find(o => o.value === formObjective)?.label }}</p>
                      <p><span class="text-muted">Público:</span> {{ formMinAge }}-{{ formMaxAge }} anos, {{ formGender === 'all' ? 'Todos' : formGender === 'male' ? 'Homens' : 'Mulheres' }}</p>
                      <p><span class="text-muted">Local:</span> {{ formCities.filter(c => c.city).map(c => c.city).join(', ') || formCountry }}</p>
                      <p><span class="text-muted">Orçamento:</span> R${{ formBudget }}/{{ formBudgetType === 'daily' ? 'dia' : 'total' }}</p>
                    </div>
                  </div>
                </template>

                <div v-if="createError" class="text-red-400 text-xs bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">{{ createError }}</div>

                <!-- Navigation Buttons -->
                <div class="flex gap-3 pt-2">
                  <button v-if="wizardStep > 1" @click="wizardStep--" class="flex-1 py-3 rounded-xl text-sm font-medium border border-bg-border/50 text-secondary hover:text-primary hover:bg-bg-elevated/50 transition-all">Voltar</button>
                  <button v-if="wizardStep < 5" @click="wizardStep++" :disabled="wizardStep === 2 && !formName" class="flex-1 py-3 rounded-xl text-sm font-semibold bg-blue-default hover:bg-blue-bright text-white transition-all disabled:opacity-50 shadow-lg shadow-blue-default/20">Continuar</button>
                  <button v-if="wizardStep === 5" :disabled="creating" @click="createCampanha" class="flex-1 flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-semibold bg-gradient-to-r from-purple-600 to-purple-500 hover:from-purple-500 hover:to-purple-400 text-white transition-all disabled:opacity-50 shadow-lg shadow-purple-500/20">
                    <WandSparkles class="w-4 h-4" :class="{ 'animate-spin': creating }" />
                    {{ creating ? 'Criando...' : 'Criar Campanha 🚀' }}
                  </button>
                </div>
              </div>
            </template>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Connect Meta Modal -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showConnect" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="showConnect = false">
          <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" />
          <div class="relative bg-bg-surface border border-bg-border/50 rounded-2xl shadow-2xl w-full max-w-md p-6">
            <!-- Close -->
            <button @click="showConnect = false" class="absolute top-4 right-4 text-muted hover:text-primary transition-colors">
              <X class="w-5 h-5" />
            </button>

            <div v-if="connectSuccess" class="text-center py-8">
              <div class="w-14 h-14 rounded-full bg-emerald-500/10 flex items-center justify-center mx-auto mb-4">
                <Check class="w-7 h-7 text-emerald-400" />
              </div>
              <p class="text-primary font-semibold text-lg">Connected!</p>
              <p class="text-muted text-sm mt-1">Campaigns syncing from Meta Ads...</p>
            </div>

            <template v-else>
              <div class="flex items-center gap-3 mb-6">
                <div class="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center">
                  <ExternalLink class="w-5 h-5 text-blue-glow" />
                </div>
                <div>
                  <h2 class="text-primary font-bold text-lg">Connect Meta Ads</h2>
                  <p class="text-muted text-sm">Link your ad account to sync campaigns</p>
                </div>
              </div>

              <div class="space-y-4">
                <!-- Step 1: Token -->
                <div>
                  <label class="text-secondary text-xs font-medium block mb-1.5">Step 1: Meta Access Token</label>
                  <input
                    v-model="metaToken"
                    type="password"
                    placeholder="Paste your long-lived access token"
                    class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all font-mono text-xs"
                    @input="availableAccounts = []; metaAccountId = ''"
                  />
                  <p class="text-muted text-xs mt-1.5">
                    Get a token from
                    <a href="https://developers.facebook.com/tools/access_token/" target="_blank" class="text-blue-bright hover:underline">Meta Access Token Tool</a>
                  </p>
                </div>

                <!-- Fetch accounts button -->
                <button
                  v-if="metaToken && !availableAccounts.length"
                  :disabled="fetchingAccounts || !metaToken.trim()"
                  class="w-full flex items-center justify-center gap-2 bg-bg-elevated/50 hover:bg-bg-elevated text-secondary text-sm py-2.5 rounded-xl border border-bg-border transition-all disabled:opacity-50"
                  @click="fetchAccounts"
                >
                  <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': fetchingAccounts }" />
                  {{ fetchingAccounts ? 'Loading...' : 'Fetch my ad accounts' }}
                </button>

                <!-- Step 2: Ad Account -->
                <div v-if="availableAccounts.length">
                  <label class="text-secondary text-xs font-medium block mb-1.5">Step 2: Select Ad Account</label>
                  <select
                    v-model="metaAccountId"
                    class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
                  >
                    <option value="" disabled>Select an account...</option>
                    <option v-for="acc in availableAccounts" :key="acc.id" :value="acc.id">
                      {{ acc.name }} ({{ acc.id }})
                    </option>
                  </select>
                </div>

                <!-- Error -->
                <div v-if="connectError" class="text-red-400 text-xs bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                  {{ connectError }}
                </div>

                <!-- Connect Button -->
                <button
                  :disabled="!metaToken || !metaAccountId || connecting"
                  class="w-full flex items-center justify-center gap-2 bg-gradient-to-r from-blue-default to-blue-bright hover:from-blue-bright hover:to-blue-glow text-white font-semibold py-2.5 rounded-xl transition-all text-sm disabled:opacity-50 shadow-lg shadow-blue-default/20"
                  @click="connectMeta"
                >
                  <LogIn class="w-4 h-4" />
                  {{ connecting ? 'Connecting...' : 'Connect & Sync' }}
                </button>
              </div>
            </template>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Filters -->
    <div class="flex flex-wrap gap-2">
      <div class="relative flex-1 min-w-[200px] max-w-xs">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted" />
        <input
          v-model="search"
          type="text"
          placeholder="Search campaigns…"
          class="w-full bg-bg-elevated/50 border border-bg-border text-secondary text-sm rounded-xl pl-9 pr-3 py-2 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
        />
      </div>
      <select
        v-model="healthFilter"
        class="bg-bg-elevated/50 border border-bg-border text-secondary text-sm rounded-xl px-3 py-2 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
      >
        <option v-for="opt in HEALTH_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>
      <select
        v-model="preset"
        class="bg-bg-elevated/50 border border-bg-border text-secondary text-sm rounded-xl px-3 py-2 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
      >
        <option v-for="p in DATE_PRESETS" :key="p.value" :value="p.value">{{ p.label }}</option>
      </select>
      <select
        v-model="sortBy"
        class="bg-bg-elevated/50 border border-bg-border text-secondary text-sm rounded-xl px-3 py-2 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
      >
        <option value="spend">Sort: Spend</option>
        <option value="ctr">Sort: CTR</option>
        <option value="roas">Sort: ROAS</option>
        <option value="name">Sort: Name</option>
      </select>
    </div>

    <!-- Error -->
    <div v-if="store.error" class="text-red-400 text-sm bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3">
      {{ store.error }}
    </div>

    <!-- Loading -->
    <template v-if="store.loading">
      <div class="grid md:grid-cols-2 xl:grid-cols-3 gap-3">
        <SkeletonCard v-for="i in 6" :key="i" />
      </div>
    </template>

    <!-- Empty -->
    <div v-else-if="displayed.length === 0" class="flex flex-col items-center justify-center py-16 text-center">
      <div class="w-16 h-16 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-4">
        <Megaphone class="w-8 h-8 text-muted" />
      </div>
      <p class="text-primary font-medium mb-1">No campaigns found</p>
      <p class="text-muted text-sm max-w-xs">Click "Connect Meta" above to link your ad account and sync campaigns.</p>
      <button
        class="mt-4 flex items-center gap-2 bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 text-sm font-medium px-4 py-2 rounded-xl border border-emerald-500/20 transition-all"
        @click="showConnect = true"
      >
        <Link2 class="w-4 h-4" />
        Connect Meta Ads
      </button>
    </div>

    <!-- Grid -->
    <div v-else class="grid md:grid-cols-2 xl:grid-cols-3 gap-3">
      <template v-for="c in displayed" :key="c.id">
        <!-- Inline Edit Card -->
        <div v-if="editingId === c.id" class="card border-bg-border/50 border-2 border-blue-default/40 shadow-lg">
          <div class="space-y-3">
            <input
              v-model="editName"
              class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-blue-default"
              placeholder="Campaign name"
            />
            <div class="flex gap-2">
              <input
                v-model.number="editBudget"
                type="number"
                class="flex-1 bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-blue-default"
                placeholder="Daily budget"
              />
              <select v-model="editStatus" class="bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-blue-default">
                <option value="ACTIVE">Active</option>
                <option value="PAUSED">Paused</option>
              </select>
            </div>
            <div class="flex gap-2 justify-end">
              <button @click="cancelEdit" class="flex items-center gap-1 text-xs text-muted hover:text-secondary px-3 py-1.5 rounded-lg border border-bg-border hover:bg-bg-elevated transition-all">
                <X class="w-3 h-3" /> Cancel
              </button>
              <button @click="saveEdit(c.id)" class="flex items-center gap-1 text-xs text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 px-3 py-1.5 rounded-lg border border-emerald-500/20 transition-all">
                <Check class="w-3 h-3" /> Save
              </button>
            </div>
          </div>
        </div>

        <!-- Normal Card -->
        <div v-else class="card border-bg-border/50 shadow-lg shadow-black/10 hover:border-blue-muted/50 hover:shadow-xl hover:shadow-blue-default/5 transition-all duration-200 block group relative">
          <NuxtLink :to="`/campaigns/${c.id}`" class="block">
            <div class="flex items-start justify-between gap-2 mb-3">
              <div class="min-w-0 flex-1">
                <p class="text-primary text-sm font-semibold truncate group-hover:text-blue-bright transition-colors">{{ c.name }}</p>
                <p class="text-muted text-xs mt-0.5">{{ c.objective }}</p>
              </div>
              <HealthBadge :status="c.health_status" class="shrink-0" />
            </div>
            <div class="grid grid-cols-3 gap-2">
              <div class="bg-bg-elevated/30 rounded-lg p-2">
                <p class="text-muted text-2xs mb-0.5">Spend</p>
                <p class="text-primary text-sm font-bold font-mono">${{ (c.spend_30d ?? 0).toLocaleString() }}</p>
              </div>
              <div class="bg-bg-elevated/30 rounded-lg p-2">
                <p class="text-muted text-2xs mb-0.5">CTR</p>
                <p class="text-primary text-sm font-bold font-mono">{{ ((c.avg_ctr_7d ?? 0) * 100).toFixed(2) }}%</p>
              </div>
              <div class="bg-bg-elevated/30 rounded-lg p-2">
                <p class="text-muted text-2xs mb-0.5">ROAS</p>
                <p class="text-primary text-sm font-bold font-mono">{{ (c.avg_roas_7d ?? 0).toFixed(2) }}x</p>
              </div>
            </div>
          </NuxtLink>

          <div class="absolute top-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <button @click.stop="startEdit(c)" class="p-1.5 rounded-lg bg-bg-surface hover:bg-bg-elevated text-muted hover:text-blue-bright border border-bg-border/50 transition-all" title="Edit">
              <PenLine class="w-3.5 h-3.5" />
            </button>
            <button @click.stop="deleteCampaign(c.id)" class="p-1.5 rounded-lg bg-bg-surface hover:bg-red-500/10 text-muted hover:text-red-400 border border-bg-border/50 transition-all" title="Delete">
              <Trash2 class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
