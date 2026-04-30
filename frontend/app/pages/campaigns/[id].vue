<script setup lang="ts">
import { useCampaignStore } from '~/stores/useCampaignStore'
import { api } from '~/lib/api'
import { DollarSign, Calendar, Activity, Lightbulb, ArrowLeft, TrendingUp, Sparkles, Target, WandSparkles, Bot, CheckCircle, XCircle, Gauge, FlaskConical } from 'lucide-vue-next'

const route = useRoute()
const store = useCampaignStore()

onMounted(() => store.fetchOne(route.params.id as string))

const c = computed(() => store.current)

function fmtCurrency(n: number) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(n)
}

// Auto-Optimize
const showOptimizer = ref(false)
const niche = ref('')
const minAge = ref(20)
const maxAge = ref(45)
const location = ref('Brasil')
const interests = ref('')
const optimizing = ref(false)
const optimizeResult = ref<any>(null)
const optimizeError = ref('')

async function autoOptimize() {
  if (!niche.value.trim()) return
  optimizing.value = true
  optimizeError.value = ''
  optimizeResult.value = null
  try {
    const result = await api.campaigns.autoOptimize(route.params.id as string, {
      niche: niche.value,
      min_age: minAge.value,
      max_age: maxAge.value,
      location: location.value || undefined,
      interests: interests.value || undefined,
    })
    optimizeResult.value = result
  } catch (e: any) {
    optimizeError.value = e.message || 'Optimization failed'
  } finally {
    optimizing.value = false
  }
}

// Smart Rules
const smartRules = ref<any[]>([{metric:'ctr',operator:'lt',value:0.5,action:'alert',days:3,enabled:true}])
const savingRules = ref(false)

onMounted(async () => {
  store.fetchOne(route.params.id as string)
  try {
    const r = await api.campaigns.getRules(route.params.id as string)
    if (r && r.length) smartRules.value = r
  } catch {}
})

async function saveRules() {
  savingRules.value = true
  try {
    await api.campaigns.saveRules(route.params.id as string, { rules: smartRules.value })
  } finally { savingRules.value = false }
}

// A/B Test
const abName = ref('')
const abMinAge = ref(18)
const abMaxAge = ref(65)
const creatingAB = ref(false)
const abResult = ref('')

async function createABTest() {
  if (!abName.value) return
  creatingAB.value = true
  abResult.value = ''
  try {
    const r = await api.campaigns.abTest(route.params.id as string, {
      test_type: 'audience',
      variant_name: abName.value,
      min_age: abMinAge.value,
      max_age: abMaxAge.value,
    })
    abResult.value = r.note || 'Teste criado!'
  } catch (e: any) {
    abResult.value = 'Erro: ' + (e.message || '')
  } finally { creatingAB.value = false }
}
</script>

<template>
  <div class="space-y-5">
    <NuxtLink to="/campaigns" class="inline-flex items-center gap-1.5 text-muted hover:text-secondary text-sm transition-colors">
      <ArrowLeft class="w-4 h-4" />
      Back to campaigns
    </NuxtLink>

    <template v-if="store.loading || !c">
      <SkeletonCard :lines="4" />
    </template>

    <template v-else>
      <div class="flex items-start justify-between flex-wrap gap-3">
        <div>
          <h1 class="text-primary text-xl font-bold">{{ c.name }}</h1>
          <p class="text-muted text-sm mt-0.5">{{ c.objective }} · {{ c.buying_type || 'Standard' }}</p>
        </div>
        <HealthBadge :status="c.health_status" />
      </div>

      <!-- KPIs -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div class="card border-bg-border/50 shadow-lg shadow-black/10">
          <div class="flex items-center gap-1.5 text-muted mb-1">
            <DollarSign class="w-3.5 h-3.5" />
            <p class="text-xs">Daily Budget</p>
          </div>
          <p class="text-primary text-lg font-bold font-mono">{{ fmtCurrency(c.daily_budget ?? 0) }}</p>
        </div>
        <div class="card border-bg-border/50 shadow-lg shadow-black/10">
          <div class="flex items-center gap-1.5 text-muted mb-1">
            <DollarSign class="w-3.5 h-3.5" />
            <p class="text-xs">Lifetime Budget</p>
          </div>
          <p class="text-primary text-lg font-bold font-mono">{{ fmtCurrency(c.lifetime_budget ?? 0) }}</p>
        </div>
        <div class="card border-bg-border/50 shadow-lg shadow-black/10">
          <div class="flex items-center gap-1.5 text-muted mb-1">
            <Activity class="w-3.5 h-3.5" />
            <p class="text-xs">Status</p>
          </div>
          <p class="text-primary text-lg font-bold">{{ c.status }}</p>
        </div>
        <div class="card border-bg-border/50 shadow-lg shadow-black/10">
          <div class="flex items-center gap-1.5 text-muted mb-1">
            <Calendar class="w-3.5 h-3.5" />
            <p class="text-xs">Start Date</p>
          </div>
          <p class="text-primary text-sm font-bold">{{ c.start_time?.slice(0, 10) ?? '—' }}</p>
        </div>
      </div>

      <!-- Auto-Optimize Button -->
      <div class="card border-bg-border/50 shadow-lg shadow-black/10 bg-gradient-to-r from-purple-500/5 to-transparent">
        <div class="flex items-center justify-between flex-wrap gap-3">
          <div class="flex items-center gap-2">
            <Bot class="w-5 h-5 text-purple-400" />
            <div>
              <h2 class="text-primary text-sm font-semibold">Otimização Inteligente 🤖</h2>
              <p class="text-muted text-xs">Defina seu público ideal e a IA ajusta a campanha automaticamente</p>
            </div>
          </div>
          <button
            v-if="!showOptimizer"
            class="flex items-center gap-2 bg-purple-500/20 hover:bg-purple-500/30 text-purple-300 text-sm font-medium px-4 py-2 rounded-xl transition-all border border-purple-500/20"
            @click="showOptimizer = true"
          >
            <Sparkles class="w-4 h-4" />
            Otimizar Agora 🚀
          </button>
        </div>

        <div v-if="showOptimizer" class="mt-4 space-y-3">
          <div class="grid md:grid-cols-2 gap-3">
            <div>
              <label class="text-secondary text-2xs font-medium block mb-1">Nichos / Segmento *</label>
              <input
                v-model="niche"
                placeholder="ex: imobiliária, educação, moda"
                class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-purple-500"
              />
            </div>
            <div>
              <label class="text-secondary text-2xs font-medium block mb-1">Localização</label>
              <input
                v-model="location"
                placeholder="Brasil"
                class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-purple-500"
              />
            </div>
            <div>
              <label class="text-secondary text-2xs font-medium block mb-1">Idade mínima</label>
              <input
                v-model.number="minAge"
                type="number"
                class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-purple-500"
              />
            </div>
            <div>
              <label class="text-secondary text-2xs font-medium block mb-1">Idade máxima</label>
              <input
                v-model.number="maxAge"
                type="number"
                class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-purple-500"
              />
            </div>
            <div class="md:col-span-2">
              <label class="text-secondary text-2xs font-medium block mb-1">Interesses (opcional)</label>
              <input
                v-model="interests"
                placeholder="ex: casa própria, financiamento, decoração"
                class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-sm focus:outline-none focus:border-purple-500"
              />
            </div>
          </div>

          <div class="flex gap-2">
            <button
              :disabled="!niche.trim() || optimizing"
              class="flex items-center gap-2 bg-gradient-to-r from-purple-600 to-purple-500 hover:from-purple-500 hover:to-purple-400 text-white font-medium px-5 py-2.5 rounded-xl transition-all text-sm disabled:opacity-50 shadow-lg shadow-purple-500/20"
              @click="autoOptimize"
            >
              <WandSparkles class="w-4 h-4" :class="{ 'animate-spin': optimizing }" />
              {{ optimizing ? 'Otimizando...' : 'Aplicar Segmentação Inteligente' }}
            </button>
            <button
              class="text-muted hover:text-secondary text-sm px-3 py-2 rounded-lg hover:bg-bg-elevated/50 transition-colors"
              @click="showOptimizer = false"
            >
              Cancelar
            </button>
          </div>

          <div v-if="optimizeError" class="text-red-400 text-xs bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">{{ optimizeError }}</div>

          <!-- Results -->
          <div v-if="optimizeResult" class="mt-4 space-y-3">
            <h3 class="text-primary text-xs font-semibold flex items-center gap-1.5">
              <CheckCircle class="w-4 h-4 text-emerald-400" />
              Resultado da Otimização
            </h3>

            <!-- Targeting Applied -->
            <div class="bg-bg-elevated/30 rounded-xl p-3 border border-bg-border/50">
              <p class="text-2xs text-muted font-medium mb-2">Segmentação Gerada pela IA</p>
              <div class="grid grid-cols-2 gap-2 text-xs">
                <div><span class="text-muted">Idade:</span> <span class="text-primary">{{ optimizeResult.targeting?.age_min }}-{{ optimizeResult.targeting?.age_max }} anos</span></div>
                <div><span class="text-muted">Gênero:</span> <span class="text-primary">{{ optimizeResult.targeting?.genders?.includes(0) ? 'Todos' : optimizeResult.targeting?.genders?.includes(1) ? 'Masculino' : 'Feminino' }}</span></div>
                <div><span class="text-muted">País:</span> <span class="text-primary">{{ optimizeResult.targeting?.geo_locations?.countries?.join(', ') }}</span></div>
              </div>
            </div>

            <!-- Results per ad set -->
            <div v-for="r in (optimizeResult.results || [])" :key="r.ad_set_id" class="flex items-center gap-2 bg-bg-elevated/30 rounded-lg px-3 py-2 border border-bg-border/50">
              <component :is="r.status === 'applied' ? CheckCircle : XCircle" class="w-4 h-4 shrink-0" :class="r.status === 'applied' ? 'text-emerald-400' : 'text-red-400'" />
              <span class="text-xs text-primary flex-1 truncate">{{ r.name || r.ad_set_id }}</span>
              <span class="text-2xs shrink-0" :class="r.status === 'applied' ? 'text-emerald-400' : 'text-red-400'">
                {{ r.status === 'applied' ? 'Aplicado' : r.status === 'targeting_gerado' ? 'Sugestão' : r.error || 'Erro' }}
              </span>
            </div>

            <p class="text-2xs text-muted flex items-center gap-1">
              <Bot class="w-3 h-3" /> Modelo: {{ optimizeResult.model_used }}
            </p>
          </div>
        </div>
      </div>
t  <!-- Smart Rules -->
	  <div class="card border-bg-border/50 shadow-lg shadow-black/10">
		<div class="flex items-center gap-2 mb-4">
		  <Gauge class="w-5 h-5 text-emerald-400" />
		  <h2 class="text-primary text-sm font-semibold">Regras Inteligentes</h2>
		</div>
		<p class="text-muted text-xs mb-3">Defina regras para o Auto-Pilot seguir automaticamente</p>
		<div class="space-y-2">
		  <div v-for="(rule, i) in smartRules" :key="i" class="flex items-center gap-2 bg-bg-elevated/30 rounded-lg p-2.5 border border-bg-border/30">
			<select v-model="rule.metric" class="bg-bg-elevated/50 border border-bg-border rounded-lg px-2 py-1.5 text-xs text-primary">
			  <option value="ctr">CTR</option>
			  <option value="frequency">Frequência</option>
			  <option value="cpc">CPC</option>
			  <option value="roas">ROAS</option>
			  <option value="spend">Gasto</option>
			</select>
			<select v-model="rule.operator" class="bg-bg-elevated/50 border border-bg-border rounded-lg px-2 py-1.5 text-xs text-primary">
			  <option value="lt">&lt;</option>
			  <option value="gt">&gt;</option>
			</select>
			<input v-model.number="rule.value" type="number" step="0.1" class="w-16 bg-bg-elevated/50 border border-bg-border rounded-lg px-2 py-1.5 text-xs text-primary text-center" />
			<span class="text-muted text-2xs">por {{ rule.days }} dias</span>
			<select v-model="rule.action" class="bg-bg-elevated/50 border border-bg-border rounded-lg px-2 py-1.5 text-xs text-primary">
			  <option value="alert">Alertar</option>
			  <option value="pause">Pausar</option>
			</select>
			<button @click="rule.enabled = !rule.enabled" class="px-2 py-1 rounded text-xs font-medium" :class="rule.enabled ? 'text-emerald-400 bg-emerald-500/10' : 'text-muted bg-bg-elevated/50'">{{ rule.enabled ? 'Ativo' : 'Inativo' }}</button>
			<button @click="smartRules.splice(i,1)" class="text-muted hover:text-red-400 p-1"><X class="w-3 h-3" /></button>
		  </div>
		</div>
		<button @click="smartRules.push({metric:'ctr',operator:'lt',value:0.5,action:'alert',days:3,enabled:true})" class="mt-2 text-xs text-blue-bright hover:underline">+ Adicionar regra</button>
		<button @click="saveRules" :disabled="savingRules" class="mt-3 w-full flex items-center justify-center gap-2 bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-400 text-sm font-medium py-2.5 rounded-xl transition-all disabled:opacity-50 border border-emerald-500/20">
		  <Gauge class="w-4 h-4" />
		  {{ savingRules ? 'Salvando...' : 'Salvar Regras Inteligentes' }}
		</button>
	  </div>

	  <!-- A/B Testing -->
	  <div class="card border-bg-border/50 shadow-lg shadow-black/10">
		<div class="flex items-center gap-2 mb-4">
		  <FlaskConical class="w-5 h-5 text-purple-400" />
		  <h2 class="text-primary text-sm font-semibold">Teste A/B de Público</h2>
		</div>
		<p class="text-muted text-xs mb-3">Crie uma variação de público para testar</p>
		<div class="grid md:grid-cols-2 gap-3">
		  <div>
			<label class="text-secondary text-2xs block mb-1">Nome da variação</label>
			<input v-model="abName" placeholder="Ex: Público Jovem" class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-xs focus:outline-none focus:border-purple-500" />
		  </div>
		  <div class="flex gap-2">
			<div>
			  <label class="text-secondary text-2xs block mb-1">Idade min</label>
			  <input v-model.number="abMinAge" type="number" class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-xs focus:outline-none focus:border-purple-500" />
			</div>
			<div>
			  <label class="text-secondary text-2xs block mb-1">Idade max</label>
			  <input v-model.number="abMaxAge" type="number" class="w-full bg-bg-elevated/50 border border-bg-border rounded-lg px-3 py-2 text-primary text-xs focus:outline-none focus:border-purple-500" />
			</div>
		  </div>
		</div>
		<button @click="createABTest" :disabled="creatingAB" class="mt-3 w-full flex items-center justify-center gap-2 bg-gradient-to-r from-purple-600 to-purple-500 hover:from-purple-500 hover:to-purple-400 text-white text-sm font-medium py-2.5 rounded-xl transition-all disabled:opacity-50 shadow-lg shadow-purple-500/20">
		  <FlaskConical class="w-4 h-4" />
		  {{ creatingAB ? 'Criando...' : 'Criar Teste A/B' }}
		</button>
		<div v-if="abResult" class="mt-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg px-3 py-2 text-xs text-emerald-400">{{ abResult }}</div>
	  </div>

      <!-- Recommendations -->
      <div v-if="c.recommendations?.length" class="card border-bg-border/50 shadow-lg shadow-black/10">
        <h2 class="text-primary text-sm font-semibold mb-4 flex items-center gap-2">
          <Lightbulb class="w-4 h-4 text-amber-400" />
          AI Recommendations
        </h2>
        <div class="space-y-3">
          <RecommendationCard v-for="r in c.recommendations" :key="r.id" :recommendation="r" />
        </div>
      </div>
    </template>
  </div>
</template>
