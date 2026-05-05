<script setup lang="ts">
import { Send, Loader2, Trash2, Sparkles, User as UserIcon, Bot } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiButton from '~/components/ui/UiButton.vue'

interface Msg {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  created_at?: string
}

const api = useApi()
const messages = ref<Msg[]>([])
const draft = ref('')
const sending = ref(false)
const loadingHistory = ref(true)
const errorMsg = ref<string | null>(null)
const suggestions = ref<string[]>([])
const messagesEnd = ref<HTMLElement | null>(null)

async function loadHistory() {
  loadingHistory.value = true
  try {
    const [hist, sug] = await Promise.all([
      api.get<{ data: Msg[] }>('/ai/chat'),
      api.get<{ data: string[] }>('/ai/chat/suggestions'),
    ])
    messages.value = hist.data || []
    suggestions.value = sug.data || []
  } catch (e: any) {
    errorMsg.value = e?.data?.error?.message || e?.message || 'Não foi possível carregar.'
  } finally {
    loadingHistory.value = false
  }
}

async function send(text?: string) {
  const msg = (text ?? draft.value).trim()
  if (!msg || sending.value) return
  errorMsg.value = null
  sending.value = true

  const optimistic: Msg = {
    id: `tmp-${Date.now()}`,
    role: 'user',
    content: msg,
    created_at: new Date().toISOString(),
  }
  messages.value.push(optimistic)
  draft.value = ''
  await scrollToBottom()

  try {
    const res = await api.post<{ data: { reply: Msg } }>('/ai/chat', { message: msg })
    if (res?.data?.reply) {
      messages.value.push(res.data.reply)
      await scrollToBottom()
    }
  } catch (e: any) {
    const m = e?.data?.error?.message || e?.message || 'Falha ao enviar mensagem.'
    errorMsg.value = m.includes('chave') || m.includes('api key')
      ? `${m} — abra Ajustes → Chaves de IA.`
      : m
  } finally {
    sending.value = false
  }
}

async function clearChat() {
  if (!confirm('Apagar todo o histórico do chat?')) return
  try {
    await api.del('/ai/chat')
    messages.value = []
  } catch (e: any) {
    errorMsg.value = e?.message || 'Falha ao limpar.'
  }
}

async function scrollToBottom() {
  await nextTick()
  messagesEnd.value?.scrollIntoView({ behavior: 'smooth' })
}

function onSubmit(e: Event) {
  e.preventDefault()
  send()
}

function autoResize(e: Event) {
  const el = e.target as HTMLTextAreaElement
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 200)}px`
}

onMounted(loadHistory)

function formatTime(s?: string) {
  if (!s) return ''
  const d = new Date(s)
  return d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}

// Markdown leve sem dependência: negrito, itálico, listas, tabelas, parágrafos.
function renderMd(text: string): string {
  let out = escapeHtml(text)
  out = out.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  out = out.replace(/(^|[^*])\*([^*]+)\*(?!\*)/g, '$1<em>$2</em>')
  const blocks = out.split(/\n\n+/)
  return blocks.map((b) => {
    if (/^\s*[-*]\s+/m.test(b)) {
      const items = b.split(/\n/).filter((l) => l.trim()).map((l) =>
        `<li>${l.replace(/^\s*[-*]\s+/, '')}</li>`,
      ).join('')
      return `<ul class="list-disc pl-5 space-y-1">${items}</ul>`
    }
    if (b.includes('|') && b.split('\n')[1]?.includes('---')) {
      return `<pre class="text-xs bg-bg-muted rounded p-2 overflow-x-auto whitespace-pre">${b}</pre>`
    }
    return `<p>${b.replace(/\n/g, '<br>')}</p>`
  }).join('')
}
</script>

<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-ink">Conversar com a IA</h1>
        <p class="text-sm text-ink-muted">Pergunte sobre suas contas, campanhas, investimento ou estratégia.</p>
      </div>
      <UiButton v-if="messages.length" variant="ghost" size="sm" @click="clearChat">
        <Trash2 class="h-4 w-4" /> Limpar
      </UiButton>
    </div>

    <div class="flex-1 space-y-4 overflow-y-auto pr-2 pb-4">
      <div v-if="loadingHistory" class="flex items-center justify-center py-12">
        <Loader2 class="h-6 w-6 animate-spin text-accent" />
      </div>

      <UiCard v-else-if="!messages.length" class="!p-8">
        <div class="text-center">
          <div class="mx-auto mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full bg-accent-soft text-accent">
            <Sparkles class="h-6 w-6" />
          </div>
          <h2 class="text-lg font-semibold text-ink">Olá. Como posso ajudar?</h2>
          <p class="mt-1 text-sm text-ink-muted max-w-md mx-auto">
            A IA olha suas contas em tempo real. Pergunte qualquer coisa sobre desempenho, custos ou estratégia.
          </p>
          <div v-if="suggestions.length" class="mt-6 flex flex-wrap justify-center gap-2">
            <button
              v-for="s in suggestions"
              :key="s"
              type="button"
              class="rounded-full border border-border bg-bg px-4 py-2 text-sm text-ink-muted transition hover:border-accent hover:text-ink"
              @click="send(s)"
            >
              {{ s }}
            </button>
          </div>
        </div>
      </UiCard>

      <template v-else>
        <div
          v-for="m in messages"
          :key="m.id"
          class="flex gap-3"
          :class="m.role === 'user' ? 'justify-end' : ''"
        >
          <div
            v-if="m.role !== 'user'"
            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent"
          >
            <Bot class="h-4 w-4" />
          </div>

          <div
            :class="[
              'max-w-[75%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed',
              m.role === 'user'
                ? 'bg-accent text-white'
                : 'bg-bg border border-border text-ink',
            ]"
          >
            <div v-if="m.role === 'user'" class="whitespace-pre-wrap">{{ m.content }}</div>
            <div v-else class="space-y-2" v-html="renderMd(m.content)" />
            <div
              :class="[
                'mt-1 text-[10px]',
                m.role === 'user' ? 'text-white/70 text-right' : 'text-ink-faint',
              ]"
            >
              {{ formatTime(m.created_at) }}
            </div>
          </div>

          <div
            v-if="m.role === 'user'"
            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-bg-muted text-ink-muted"
          >
            <UserIcon class="h-4 w-4" />
          </div>
        </div>

        <div v-if="sending" class="flex gap-3">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent">
            <Bot class="h-4 w-4" />
          </div>
          <div class="rounded-2xl border border-border bg-bg px-4 py-3">
            <div class="flex items-center gap-2 text-sm text-ink-muted">
              <Loader2 class="h-3.5 w-3.5 animate-spin" />
              Pensando...
            </div>
          </div>
        </div>

        <div ref="messagesEnd" />
      </template>
    </div>

    <div v-if="errorMsg" class="mb-2 rounded-lg bg-danger-soft px-3 py-2 text-sm text-danger">
      {{ errorMsg }}
    </div>

    <form class="flex items-end gap-2 rounded-xl border border-border bg-bg p-2" @submit="onSubmit">
      <textarea
        v-model="draft"
        rows="1"
        placeholder="Pergunte qualquer coisa..."
        class="flex-1 resize-none bg-transparent px-2 py-1.5 text-sm text-ink placeholder:text-ink-faint focus:outline-none"
        :disabled="sending"
        @input="autoResize"
        @keydown.enter.exact.prevent="send()"
      />
      <UiButton type="submit" variant="primary" :loading="sending" :disabled="!draft.trim()">
        <Send class="h-4 w-4" />
        <span class="hidden sm:inline">Enviar</span>
      </UiButton>
    </form>
  </div>
</template>
