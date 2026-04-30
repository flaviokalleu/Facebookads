<script setup lang="ts">
import { api } from '~/lib/api'
import {
  Palette, Sparkles, Send, AlertTriangle, CheckCircle,
  AlertCircle, RefreshCw, Lightbulb, Target, FileText,
  MessageSquareText, Cpu, WandSparkles, TrendingUp, Eye
} from 'lucide-vue-next'

const creatives = ref<any[]>([])
const loading = ref(true)

// AI Analysis
const analyzing = ref(false)
const aiInsights = ref<any>(null)
const aiAnalyzed = ref(false)
const aiModelUsed = ref('')
const aiError = ref('')

// AI Improve
const improveMode = ref(false)
const selectedCreative = ref<any>(null)
const instructions = ref('')
const improving = ref(false)
const variations = ref<any[]>([])
const showImproveForm = ref(false)

onMounted(async () => {
  try {
    const data = await api.creatives.list()
    creatives.value = data ?? []
  } catch {
    // no-op
  } finally {
    loading.value = false
  }
})

async function analyzeWithAI() {
  analyzing.value = true
  aiError.value = ''
  aiInsights.value = null
  try {
    const result = await api.creatives.analyze()
    aiAnalyzed.value = result.ai_analyzed
    aiInsights.value = result.ai_insights
    aiModelUsed.value = result.model_used || ''
    if (result.creatives) {
      // Merge any enriched creative data
      result.creatives.forEach((rc: any) => {
        const idx = creatives.value.findIndex((c: any) => c.id === rc.id)
        if (idx >= 0) {
          creatives.value[idx] = { ...creatives.value[idx], ...rc }
        }
      })
    }
  } catch (e: any) {
    aiError.value = e.message || 'AI analysis failed'
  } finally {
    analyzing.value = false
  }
}

function openImprove(creative: any) {
  selectedCreative.value = creative
  instructions.value = ''
  variations.value = []
  showImproveForm.value = true
}

async function improveWithAI() {
  if (!instructions.value.trim()) return
  improving.value = true
  try {
    const result = await api.creatives.improve({
      instructions: instructions.value,
      creative_id: selectedCreative.value?.id,
      campaign_name: selectedCreative.value?.campaign_name,
      headline: selectedCreative.value?.headline,
      body: selectedCreative.value?.body,
      cta: selectedCreative.value?.cta_type,
    })
    variations.value = result.variations || []
  } catch (e: any) {
    console.error('Improve failed', e)
  } finally {
    improving.value = false
  }
}

const sortedByFatigue = computed(() =>
  [...creatives.value].sort((a, b) => (b.fatigue_score ?? 0) - (a.fatigue_score ?? 0))
)

const highFatigue = computed(() => creatives.value.filter((c: any) => (c.fatigue_score ?? 0) >= 75).length)
const moderateFatigue = computed(() => creatives.value.filter((c: any) => (c.fatigue_score ?? 0) >= 50 && (c.fatigue_score ?? 0) < 75).length)
const healthyFatigue = computed(() => creatives.value.filter((c: any) => (c.fatigue_score ?? 0) < 50).length)

function fatigueColor(score: number) {
  if (score >= 75) return 'text-red-400'
  if (score >= 50) return 'text-amber-400'
  return 'text-emerald-400'
}

function fatigueBg(score: number) {
  if (score >= 75) return 'bg-red-500'
  if (score >= 50) return 'bg-amber-500'
  return 'bg-emerald-500'
}
</script>

<template>
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div class="flex items-center gap-2">
        <Palette class="w-5 h-5 text-purple-400" />
        <div>
          <h1 class="text-primary text-xl font-bold">Creative Studio</h1>
          <p class="text-muted text-sm">Fatigue analysis &amp; AI-powered optimization</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <button
          :disabled="analyzing || creatives.length === 0"
          class="flex items-center gap-2 bg-purple-500/20 hover:bg-purple-500/30 text-purple-300 text-sm font-medium px-4 py-2 rounded-xl transition-all disabled:opacity-50 border border-purple-500/20"
          @click="analyzeWithAI"
        >
          <Sparkles class="w-4 h-4" :class="{ 'animate-pulse': analyzing }" />
          {{ analyzing ? 'Analyzing...' : 'AI Analyze' }}
        </button>
      </div>
    </div>

    <!-- Stats bar -->
    <div v-if="!loading && creatives.length" class="flex gap-3 text-sm flex-wrap">
      <div class="card border-bg-border/50 flex items-center gap-2 py-2 px-4 shadow-sm">
        <AlertTriangle class="w-4 h-4 text-red-400" />
        <span class="text-red-400 font-bold">{{ highFatigue }}</span>
        <span class="text-muted">High fatigue</span>
      </div>
      <div class="card border-bg-border/50 flex items-center gap-2 py-2 px-4 shadow-sm">
        <AlertCircle class="w-4 h-4 text-amber-400" />
        <span class="text-amber-400 font-bold">{{ moderateFatigue }}</span>
        <span class="text-muted">Moderate</span>
      </div>
      <div class="card border-bg-border/50 flex items-center gap-2 py-2 px-4 shadow-sm">
        <CheckCircle class="w-4 h-4 text-emerald-400" />
        <span class="text-emerald-400 font-bold">{{ healthyFatigue }}</span>
        <span class="text-muted">Healthy</span>
      </div>
    </div>

    <!-- AI Insights Panel -->
    <div v-if="aiInsights" class="card border-bg-border/50 shadow-lg shadow-black/10 bg-gradient-to-r from-purple-500/5 to-transparent">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-primary text-sm font-semibold flex items-center gap-2">
          <Lightbulb class="w-4 h-4 text-amber-400" />
          AI Creative Insights
        </h2>
        <span class="text-muted text-2xs flex items-center gap-1">
          <Cpu class="w-3 h-3" /> {{ aiModelUsed }}
        </span>
      </div>

      <div class="grid md:grid-cols-2 gap-4">
        <!-- Winning Patterns -->
        <div class="bg-emerald-500/5 border border-emerald-500/10 rounded-xl p-3">
          <h3 class="text-xs font-semibold text-emerald-400 mb-2 flex items-center gap-1.5">
            <TrendingUp class="w-3.5 h-3.5" /> Winning Patterns
          </h3>
          <ul class="space-y-1">
            <li v-for="(p, i) in (aiInsights.winning_patterns || [])" :key="i" class="text-xs text-secondary flex items-start gap-1.5">
              <span class="text-emerald-400 mt-0.5">•</span> {{ p }}
            </li>
            <li v-if="!(aiInsights.winning_patterns || []).length" class="text-xs text-muted">No patterns identified</li>
          </ul>
        </div>

        <!-- Losing Patterns -->
        <div class="bg-red-500/5 border border-red-500/10 rounded-xl p-3">
          <h3 class="text-xs font-semibold text-red-400 mb-2 flex items-center gap-1.5">
            <AlertTriangle class="w-3.5 h-3.5" /> Losing Patterns
          </h3>
          <ul class="space-y-1">
            <li v-for="(p, i) in (aiInsights.losing_patterns || [])" :key="i" class="text-xs text-secondary flex items-start gap-1.5">
              <span class="text-red-400 mt-0.5">•</span> {{ p }}
            </li>
            <li v-if="!(aiInsights.losing_patterns || []).length" class="text-xs text-muted">No patterns identified</li>
          </ul>
        </div>

        <!-- Headline Insights -->
        <div class="bg-blue-500/5 border border-blue-500/10 rounded-xl p-3">
          <h3 class="text-xs font-semibold text-blue-400 mb-2 flex items-center gap-1.5">
            <FileText class="w-3.5 h-3.5" /> Headline Insights
          </h3>
          <p class="text-xs text-secondary">{{ aiInsights.headline_insights || 'No analysis available' }}</p>
        </div>

        <!-- CTA Insights -->
        <div class="bg-amber-500/5 border border-amber-500/10 rounded-xl p-3">
          <h3 class="text-xs font-semibold text-amber-400 mb-2 flex items-center gap-1.5">
            <MessageSquareText class="w-3.5 h-3.5" /> CTA Insights
          </h3>
          <p class="text-xs text-secondary">{{ aiInsights.cta_insights || 'No analysis available' }}</p>
        </div>
      </div>

      <!-- AI Recommendations -->
      <div v-if="(aiInsights.recommendations || []).length" class="mt-4 bg-bg-elevated/50 rounded-xl p-3">
        <h3 class="text-xs font-semibold text-purple-400 mb-2 flex items-center gap-1.5">
          <WandSparkles class="w-3.5 h-3.5" /> Recommendations
        </h3>
        <ul class="space-y-1">
          <li v-for="(r, i) in aiInsights.recommendations" :key="i" class="text-xs text-secondary flex items-start gap-1.5">
            <span class="text-purple-400 mt-0.5">→</span> {{ r }}
          </li>
        </ul>
      </div>
    </div>

    <!-- AI Error -->
    <div v-if="aiError" class="text-red-400 text-sm bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3">
      {{ aiError }}
    </div>

    <template v-if="loading">
      <div class="grid md:grid-cols-2 xl:grid-cols-3 gap-3">
        <SkeletonCard v-for="i in 6" :key="i" />
      </div>
    </template>

    <!-- Empty -->
    <div v-else-if="creatives.length === 0" class="flex flex-col items-center justify-center py-16 text-center">
      <div class="w-16 h-16 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-4">
        <Palette class="w-8 h-8 text-muted" />
      </div>
      <p class="text-primary font-medium mb-1">No creatives yet</p>
      <p class="text-muted text-sm max-w-xs">Sync your campaigns to pull ad creatives from Meta Ads.</p>
    </div>

    <!-- Creative Grid -->
    <div v-else class="grid md:grid-cols-2 xl:grid-cols-3 gap-3">
      <div
        v-for="c in sortedByFatigue"
        :key="c.id"
        class="card border-bg-border/50 shadow-lg shadow-black/10 hover:border-blue-muted/50 transition-all duration-200 group relative"
      >
        <!-- Fatigue badge -->
        <div class="flex items-start justify-between gap-2 mb-3">
          <div class="min-w-0 flex-1">
            <p class="text-primary text-xs font-semibold truncate">{{ c.headline || 'Untitled' }}</p>
            <p class="text-muted text-2xs truncate">{{ c.campaign_name }}</p>
          </div>
          <span class="text-2xs font-bold shrink-0 px-2 py-0.5 rounded-full border" :class="`${fatigueColor(c.fatigue_score)} bg-${c.fatigue_score >= 75 ? 'red' : c.fatigue_score >= 50 ? 'amber' : 'emerald'}-500/10`">
            {{ c.fatigue_score >= 75 ? 'High Fatigue' : c.fatigue_score >= 50 ? 'Moderate' : 'Healthy' }}
          </span>
        </div>

        <!-- Body text preview -->
        <p class="text-muted text-xs line-clamp-2 mb-3 leading-relaxed">{{ c.body || 'No body text' }}</p>

        <!-- Fatigue bar -->
        <div class="mb-3">
          <div class="flex justify-between text-2xs text-muted mb-0.5">
            <span>Fatigue</span>
            <span :class="fatigueColor(c.fatigue_score)">{{ (c.fatigue_score ?? 0).toFixed(0) }}%</span>
          </div>
          <div class="h-1.5 bg-bg-elevated rounded-full overflow-hidden">
            <div class="h-full rounded-full transition-all duration-500" :class="fatigueBg(c.fatigue_score)" :style="{ width: `${Math.min(c.fatigue_score ?? 0, 100)}%` }" />
          </div>
        </div>

        <!-- Metrics -->
        <div class="flex gap-3 text-2xs mb-2">
          <div class="flex items-center gap-1">
            <TrendingUp class="w-3 h-3 text-muted" />
            <span class="text-muted">CTR </span>
            <span class="text-primary font-mono font-medium">{{ ((c.ctr ?? 0) * 100).toFixed(2) }}%</span>
          </div>
          <div class="flex items-center gap-1">
            <Eye class="w-3 h-3 text-muted" />
            <span class="text-muted">Impr. </span>
            <span class="text-primary font-mono font-medium">{{ (c.impressions ?? 0) >= 1000 ? `${((c.impressions ?? 0) / 1000).toFixed(1)}K` : (c.impressions ?? 0) }}</span>
          </div>
          <div class="text-muted">Freq: <span class="text-primary font-mono">{{ (c.frequency ?? 0).toFixed(1) }}x</span></div>
        </div>

        <!-- CTA -->
        <div v-if="c.cta_type" class="mb-2">
          <span class="text-2xs bg-blue-500/10 text-blue-300 px-1.5 py-0.5 rounded border border-blue-500/20">{{ c.cta_type }}</span>
        </div>

        <!-- Improve button -->
        <button
          class="mt-2 w-full flex items-center justify-center gap-1.5 text-2xs font-medium text-purple-400 hover:text-purple-300 bg-purple-500/10 hover:bg-purple-500/20 px-3 py-1.5 rounded-lg border border-purple-500/20 transition-all opacity-0 group-hover:opacity-100"
          @click="openImprove(c)"
        >
          <WandSparkles class="w-3 h-3" />
          Improve with AI
        </button>
      </div>
    </div>

    <!-- AI Improve Modal -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showImproveForm" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="showImproveForm = false">
          <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" />
          <div class="relative bg-bg-surface border border-bg-border/50 rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto p-6">
            <button @click="showImproveForm = false" class="absolute top-4 right-4 text-muted hover:text-primary">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
            </button>

            <div class="flex items-center gap-3 mb-6">
              <div class="w-10 h-10 rounded-xl bg-purple-500/10 flex items-center justify-center">
                <WandSparkles class="w-5 h-5 text-purple-400" />
              </div>
              <div>
                <h2 class="text-primary font-bold text-lg">AI Creative Improvement</h2>
                <p class="text-muted text-sm">Tell the AI how you want to improve this creative</p>
              </div>
            </div>

            <!-- Current Creative -->
            <div v-if="selectedCreative" class="bg-bg-elevated/50 rounded-xl p-4 mb-4 border border-bg-border/50">
              <p class="text-primary text-xs font-semibold mb-2">Current Creative</p>
              <p class="text-xs text-secondary"><span class="text-muted">Headline:</span> {{ selectedCreative.headline || '—' }}</p>
              <p class="text-xs text-secondary mt-1"><span class="text-muted">Body:</span> {{ selectedCreative.body?.slice(0, 100) || '—' }}{{ selectedCreative.body?.length > 100 ? '...' : '' }}</p>
              <p class="text-xs text-secondary mt-1"><span class="text-muted">CTA:</span> {{ selectedCreative.cta_type || '—' }}</p>
              <p class="text-xs text-secondary mt-1"><span class="text-muted">Campaign:</span> {{ selectedCreative.campaign_name }}</p>
            </div>

            <textarea
              v-model="instructions"
              rows="4"
              class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-4 py-3 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-purple-500 focus:ring-1 focus:ring-purple-500/30 transition-all resize-none"
              placeholder="Describe what you want to improve...&#10;&#10;Example: Make this more urgent for a limited-time offer, target young homeowners, add social proof..."
            />

            <button
              :disabled="!instructions.trim() || improving"
              class="mt-4 w-full flex items-center justify-center gap-2 bg-gradient-to-r from-purple-600 to-purple-500 hover:from-purple-500 hover:to-purple-400 text-white font-semibold py-3 rounded-xl transition-all text-sm disabled:opacity-50 shadow-lg shadow-purple-500/20"
              @click="improveWithAI"
            >
              <Send class="w-4 h-4" />
              {{ improving ? 'Generating...' : 'Generate Improvements' }}
            </button>

            <!-- Variations -->
            <div v-if="variations.length" class="mt-6 space-y-3">
              <h3 class="text-primary text-sm font-semibold flex items-center gap-2">
                <WandSparkles class="w-4 h-4 text-purple-400" />
                AI-Generated Variations
              </h3>
              <div
                v-for="v in variations"
                :key="v.variant"
                class="bg-gradient-to-br from-purple-500/5 to-transparent border border-purple-500/20 rounded-xl p-4"
              >
                <span class="text-2xs text-purple-400 font-semibold uppercase tracking-wider">Variant {{ v.variant }}</span>
                <div class="mt-2 space-y-1.5">
                  <p class="text-xs"><span class="text-muted">Headline:</span> <span class="text-primary">{{ v.headline }}</span></p>
                  <p class="text-xs"><span class="text-muted">Body:</span> <span class="text-primary">{{ v.primary_text }}</span></p>
                  <p class="text-xs"><span class="text-muted">CTA:</span> <span class="text-primary">{{ v.cta }}</span></p>
                  <p class="text-xs text-secondary mt-2 bg-bg-base/50 rounded-lg p-2">{{ v.reasoning }}</p>
                  <p v-if="v.expected_impact" class="text-2xs text-emerald-400 mt-1 flex items-center gap-1">
                    <TrendingUp class="w-3 h-3" /> {{ v.expected_impact }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
