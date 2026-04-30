<script setup lang="ts">
import { useProvidersStore } from '~/stores/useProvidersStore'
import { Cpu, CheckCircle, XCircle, Key, Plus, X, Save, ExternalLink, DollarSign, Hash, Activity } from 'lucide-vue-next'

const store = useProvidersStore()
onMounted(() => Promise.all([store.fetchProviders(), store.fetchConfigs()]))

const editingKey = ref<string | null>(null)
const editValue = ref('')
const saving = ref(false)

function startEdit(key: string) {
  editingKey.value = key
  editValue.value = ''
}

function cancelEdit() {
  editingKey.value = null
  editValue.value = ''
}

async function saveEdit(key: string, isSecret: boolean) {
  if (!editValue.value.trim()) return
  saving.value = true
  try {
    await store.saveConfig(key, editValue.value, isSecret)
    editingKey.value = null
    editValue.value = ''
  } finally {
    saving.value = false
  }
}

interface ProviderBrand {
  key: string
  name: string
  model: string
  color: string
  secret: boolean
}

const AI_PROVIDERS: ProviderBrand[] = [
  { key: 'ai.anthropic.api_key', name: 'Anthropic',  model: 'Claude',   color: '#D97706', secret: true },
  { key: 'ai.openai.api_key',    name: 'OpenAI',     model: 'GPT',      color: '#10A37F', secret: true },
  { key: 'ai.google.api_key',    name: 'Google AI',  model: 'Gemini',   color: '#4285F4', secret: true },
  { key: 'ai.deepseek.api_key',  name: 'DeepSeek',   model: 'V3',       color: '#4F46E5', secret: true },
  { key: 'ai.zhipu.api_key',     name: 'Zhipu AI',   model: 'GLM',      color: '#8B5CF6', secret: true },
  { key: 'ai.moonshot.api_key',  name: 'Moonshot',   model: 'Kimi',     color: '#06B6D4', secret: true },
  { key: 'ai.alibaba.api_key',   name: 'Alibaba',    model: 'Qwen',     color: '#FF6A00', secret: true },
  { key: 'ai.xai.api_key',       name: 'xAI',        model: 'Grok',     color: '#E5E7EB', secret: true },
]

const OTHER_KEYS = [
  { key: 'meta.app_id',     name: 'Meta App ID',     color: '#1877F2', secret: false },
  { key: 'meta.app_secret', name: 'Meta App Secret', color: '#1877F2', secret: true },
  { key: 'jwt.secret',      name: 'JWT Secret',      color: '#6B7280', secret: true },
]

function getConfig(key: string) {
  return store.configs.find((c: any) => c.key === key)
}

function isSet(key: string) {
  return !!getConfig(key)
}

// Custom key form
const showCustomForm = ref(false)
const customKey = ref('')
const customValue = ref('')
const customSecret = ref(false)

async function saveCustom() {
  if (!customKey.value || !customValue.value) return
  saving.value = true
  try {
    await store.saveConfig(customKey.value, customValue.value, customSecret.value)
    customKey.value = ''
    customValue.value = ''
    customSecret.value = false
    showCustomForm.value = false
  } finally {
    saving.value = false
  }
}

const onlineCount = computed(() => store.providers.filter((p: any) => p.available).length)
const totalCost = computed(() => store.providers.reduce((sum: number, p: any) => sum + (p.total_cost ?? 0), 0))
const totalRequests = computed(() => store.providers.reduce((sum: number, p: any) => sum + (p.total_requests ?? 0), 0))
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center gap-2">
      <Cpu class="w-5 h-5 text-blue-glow" />
      <div>
        <h1 class="text-primary text-xl font-bold">AI Providers</h1>
        <p class="text-muted text-sm">Configure API keys and monitor provider health</p>
      </div>
    </div>

    <!-- Provider Status -->
    <div class="card border-bg-border/50 shadow-lg shadow-black/10">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-primary text-sm font-semibold flex items-center gap-2">
          <Activity class="w-4 h-4 text-blue-glow" />
          Provider Status
        </h2>
        <div class="flex items-center gap-4 text-xs text-muted">
          <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-emerald-400" /> {{ onlineCount }}/{{ store.providers.length }} online</span>
          <span class="flex items-center gap-1"><Hash class="w-3 h-3" /> {{ totalRequests.toLocaleString() }} requests</span>
          <span class="flex items-center gap-1"><DollarSign class="w-3 h-3" /> ${{ totalCost.toFixed(4) }}</span>
        </div>
      </div>

      <template v-if="store.loading">
        <div class="grid md:grid-cols-2 xl:grid-cols-3 gap-3">
          <SkeletonCard v-for="i in 6" :key="i" />
        </div>
      </template>
      <div v-else-if="store.providers.length === 0" class="flex flex-col items-center justify-center py-12 text-center">
        <div class="w-14 h-14 rounded-2xl bg-bg-elevated/50 flex items-center justify-center mb-4">
          <Cpu class="w-7 h-7 text-muted" />
        </div>
        <p class="text-primary font-medium mb-1">No providers configured</p>
        <p class="text-muted text-sm max-w-xs">Add API keys below to enable AI providers.</p>
      </div>
      <div v-else class="grid md:grid-cols-2 xl:grid-cols-3 gap-3">
        <div
          v-for="p in store.providers"
          :key="p.name"
          class="bg-bg-elevated/50 border border-bg-border/50 rounded-xl p-4 hover:border-blue-default/30 transition-all"
        >
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2.5">
              <div class="w-9 h-9 rounded-lg bg-white/5 flex items-center justify-center">
                <Cpu class="w-5 h-5 text-blue-glow/70" />
              </div>
              <div>
                <p class="text-primary text-sm font-medium capitalize">{{ p.name }}</p>
                <p class="text-muted text-xs font-mono">{{ p.model_id }}</p>
              </div>
            </div>
            <component :is="p.available ? CheckCircle : XCircle" class="w-4 h-4" :class="p.available ? 'text-emerald-400' : 'text-red-400'" />
          </div>

          <div class="grid grid-cols-2 gap-2 text-xs">
            <div class="bg-bg-base/50 rounded-lg p-2">
              <p class="text-muted mb-0.5">Requests</p>
              <p class="text-primary font-mono font-medium">{{ (p.total_requests ?? 0).toLocaleString() }}</p>
            </div>
            <div class="bg-bg-base/50 rounded-lg p-2">
              <p class="text-muted mb-0.5">Total Cost</p>
              <p class="text-primary font-mono font-medium">${{ (p.total_cost ?? 0).toFixed(4) }}</p>
            </div>
          </div>

          <div v-if="p.cost_per_1m_input" class="mt-2 flex gap-3 text-xs text-muted">
            <span>In: ${{ p.cost_per_1m_input }}/1M</span>
            <span>Out: ${{ p.cost_per_1m_output }}/1M</span>
          </div>
        </div>
      </div>
    </div>

    <!-- API Keys -->
    <div class="card border-bg-border/50 shadow-lg shadow-black/10">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-primary text-sm font-semibold flex items-center gap-2">
          <Key class="w-4 h-4 text-blue-glow" />
          API Keys
        </h2>
        <button
          class="flex items-center gap-1 text-xs text-blue-bright hover:text-blue-glow transition-colors font-medium"
          @click="showCustomForm = !showCustomForm"
        >
          <Plus class="w-3 h-3" />
          {{ showCustomForm ? 'Cancel' : 'Custom key' }}
        </button>
      </div>

      <!-- Custom key form -->
      <div v-if="showCustomForm" class="bg-bg-base/50 border border-bg-border/50 rounded-xl p-4 mb-5 space-y-3">
        <div class="grid sm:grid-cols-2 gap-3">
          <input
            v-model="customKey"
            type="text"
            placeholder="Config key (e.g. ai.openai.api_key)"
            class="bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
          />
          <input
            v-model="customValue"
            :type="customSecret ? 'password' : 'text'"
            placeholder="Value"
            class="bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
          />
        </div>
        <div class="flex items-center justify-between">
          <label class="flex items-center gap-2 text-secondary text-sm cursor-pointer select-none">
            <input v-model="customSecret" type="checkbox" class="accent-blue-default rounded" />
            Encrypt as secret
          </label>
          <button
            class="flex items-center gap-1 bg-gradient-to-r from-blue-default to-blue-bright hover:from-blue-bright hover:to-blue-glow text-white text-sm font-medium px-5 py-2 rounded-xl transition-all disabled:opacity-50"
            :disabled="saving"
            @click="saveCustom"
          >
            <Save class="w-3.5 h-3.5" />
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </div>

      <!-- AI Provider keys -->
      <div class="space-y-5">
        <div>
          <h3 class="text-xs font-semibold text-secondary uppercase tracking-widest mb-2">AI Providers</h3>
          <div class="space-y-1">
            <div
              v-for="p in AI_PROVIDERS"
              :key="p.key"
              class="flex items-center gap-3 bg-bg-base/50 border border-bg-border/50 rounded-xl px-4 py-3 group hover:border-blue-default/20 transition-all"
            >
              <div class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0" :style="{ backgroundColor: p.color + '18' }">
                <Cpu class="w-5 h-5" :style="{ color: p.color }" />
              </div>

              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <p class="text-primary text-sm font-medium">{{ p.name }}</p>
                  <span class="text-muted text-xs font-mono">{{ p.model }}</span>
                </div>
                <p class="text-muted text-xs font-mono truncate">{{ p.key }}</p>
              </div>

              <span
                class="text-xs px-1.5 py-px rounded-full border shrink-0"
                :class="isSet(p.key) ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20' : 'text-amber-400 bg-amber-500/10 border-amber-500/20'"
              >
                {{ isSet(p.key) ? 'Active' : 'Missing' }}
              </span>

              <div v-if="editingKey === p.key" class="flex items-center gap-2 shrink-0">
                <input
                  v-model="editValue"
                  :type="p.secret ? 'password' : 'text'"
                  placeholder="Enter key…"
                  class="w-40 bg-bg-elevated/50 border border-bg-border rounded-lg px-2.5 py-1.5 text-primary text-xs placeholder-muted/50 focus:outline-none focus:border-blue-default transition-all"
                  @keyup.enter="saveEdit(p.key, p.secret)"
                  @keyup.escape="cancelEdit"
                />
                <button class="text-xs bg-blue-default hover:bg-blue-bright text-white px-3 py-1.5 rounded-lg transition-all font-medium disabled:opacity-50" :disabled="saving" @click="saveEdit(p.key, p.secret)">
                  {{ saving ? '…' : 'Save' }}
                </button>
                <button class="text-xs text-muted hover:text-secondary px-2 py-1.5 rounded-lg hover:bg-bg-elevated/50 transition-colors" @click="cancelEdit">
                  <X class="w-3.5 h-3.5" />
                </button>
              </div>

              <div v-else class="flex items-center gap-2 shrink-0">
                <template v-if="isSet(p.key)">
                  <span class="text-muted text-xs font-mono bg-bg-elevated/50 px-2 py-0.5 rounded border border-bg-border/50">••••••••••••</span>
                </template>
                <button class="text-xs text-blue-bright hover:text-blue-glow font-medium opacity-0 group-hover:opacity-100 transition-all" @click="startEdit(p.key)">
                  {{ isSet(p.key) ? 'Update' : 'Configure' }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div>
          <h3 class="text-xs font-semibold text-secondary uppercase tracking-widest mb-2">Platform & System</h3>
          <div class="space-y-1">
            <div
              v-for="item in OTHER_KEYS"
              :key="item.key"
              class="flex items-center gap-3 bg-bg-base/50 border border-bg-border/50 rounded-xl px-4 py-3 group hover:border-blue-default/20 transition-all"
            >
              <div class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0" :style="{ backgroundColor: item.color + '18' }">
                <Key class="w-5 h-5" :style="{ color: item.color }" />
              </div>

              <div class="flex-1 min-w-0">
                <p class="text-primary text-sm font-medium">{{ item.name }}</p>
                <p class="text-muted text-xs font-mono truncate">{{ item.key }}</p>
              </div>

              <span
                class="text-xs px-1.5 py-px rounded-full border shrink-0"
                :class="isSet(item.key) ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20' : 'text-amber-400 bg-amber-500/10 border-amber-500/20'"
              >
                {{ isSet(item.key) ? 'Active' : 'Missing' }}
              </span>

              <div v-if="editingKey === item.key" class="flex items-center gap-2 shrink-0">
                <input
                  v-model="editValue"
                  :type="item.secret ? 'password' : 'text'"
                  placeholder="Enter value…"
                  class="w-40 bg-bg-elevated/50 border border-bg-border rounded-lg px-2.5 py-1.5 text-primary text-xs placeholder-muted/50 focus:outline-none focus:border-blue-default transition-all"
                  @keyup.enter="saveEdit(item.key, item.secret)"
                  @keyup.escape="cancelEdit"
                />
                <button class="text-xs bg-blue-default hover:bg-blue-bright text-white px-3 py-1.5 rounded-lg transition-all font-medium disabled:opacity-50" :disabled="saving" @click="saveEdit(item.key, item.secret)">
                  {{ saving ? '…' : 'Save' }}
                </button>
                <button class="text-xs text-muted hover:text-secondary px-2 py-1.5 rounded-lg hover:bg-bg-elevated/50 transition-colors" @click="cancelEdit">
                  <X class="w-3.5 h-3.5" />
                </button>
              </div>

              <div v-else class="flex items-center gap-2 shrink-0">
                <template v-if="isSet(item.key) && !item.secret">
                  <span class="text-secondary text-xs font-mono truncate max-w-32 bg-bg-elevated/50 px-2 py-0.5 rounded border border-bg-border/50">
                    {{ getConfig(item.key)?.value }}
                  </span>
                </template>
                <template v-else-if="isSet(item.key)">
                  <span class="text-muted text-xs font-mono bg-bg-elevated/50 px-2 py-0.5 rounded border border-bg-border/50">••••••••••••</span>
                </template>
                <button class="text-xs text-blue-bright hover:text-blue-glow font-medium opacity-0 group-hover:opacity-100 transition-all" @click="startEdit(item.key)">
                  {{ isSet(item.key) ? 'Update' : 'Configure' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
