<script setup lang="ts">
import { Key, CheckCircle2, Eye, EyeOff } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiInput from '~/components/ui/UiInput.vue'
import UiButton from '~/components/ui/UiButton.vue'
import UiBadge from '~/components/ui/UiBadge.vue'

interface ProviderKey {
  key: string
  label: string
  description: string
  baseUrl: string
}

const providers: ProviderKey[] = [
  { key: 'ai.deepseek.api_key',  label: 'DeepSeek',  description: 'Modelo padrão pra análise híbrida — barato e rápido. Recomendado.', baseUrl: 'https://platform.deepseek.com' },
  { key: 'ai.anthropic.api_key', label: 'Anthropic Claude', description: 'Modelos premium. Use só se quiser substituir o DeepSeek em casos críticos.', baseUrl: 'https://console.anthropic.com' },
  { key: 'ai.openai.api_key',    label: 'OpenAI',    description: 'Opcional. Backup quando outros estão fora do ar.', baseUrl: 'https://platform.openai.com' },
]

const api = useApi()
const values = reactive<Record<string, string>>({})
const masked = reactive<Record<string, boolean>>({})
const saving = reactive<Record<string, boolean>>({})
const status = reactive<Record<string, 'configured' | 'empty'>>({})
const message = ref<{ key: string; text: string; ok: boolean } | null>(null)

async function loadStatus() {
  try {
    const res = await api.get<{ data: Array<{ key: string; value: string; is_secret: boolean }> }>('/admin/config')
    const list = res?.data || []
    for (const p of providers) {
      const found = list.find((x) => x.key === p.key)
      status[p.key] = found && found.value && found.value !== '' ? 'configured' : 'empty'
      masked[p.key] = true
      values[p.key] = ''
    }
  } catch {
    for (const p of providers) status[p.key] = 'empty'
  }
}

onMounted(loadStatus)

async function save(p: ProviderKey) {
  if (!values[p.key]) return
  saving[p.key] = true
  message.value = null
  try {
    await api.put(`/admin/config/${p.key}`, { value: values[p.key], is_secret: true })
    message.value = { key: p.key, text: 'Chave salva. Pode levar até 5 minutos para a IA usar a nova chave.', ok: true }
    values[p.key] = ''
    await loadStatus()
  } catch (e: any) {
    const msg = e?.data?.error?.message || e?.message || 'Não foi possível salvar.'
    message.value = { key: p.key, text: msg, ok: false }
  } finally {
    saving[p.key] = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-ink">Chaves de IA</h1>
      <p class="mt-1 text-sm text-ink-muted max-w-2xl">
        A IA precisa de uma chave de API pra funcionar. O DeepSeek é o motor padrão — mais barato e rápido pra análises de tráfego.
        As chaves ficam criptografadas no banco e nunca aparecem no frontend.
      </p>
    </div>

    <div class="space-y-4">
      <UiCard v-for="p in providers" :key="p.key">
        <div class="flex items-start gap-4">
          <div class="rounded-full bg-accent-soft p-2 text-accent">
            <Key class="h-5 w-5" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex flex-wrap items-center gap-3">
              <h3 class="font-semibold text-ink">{{ p.label }}</h3>
              <UiBadge v-if="status[p.key] === 'configured'" variant="success">
                <CheckCircle2 class="h-3 w-3" /> Configurado
              </UiBadge>
              <UiBadge v-else variant="neutral">Sem chave</UiBadge>
              <a
                :href="p.baseUrl"
                target="_blank"
                rel="noopener"
                class="ml-auto text-xs text-accent hover:underline"
              >
                Onde encontro?
              </a>
            </div>
            <p class="mt-1 text-sm text-ink-muted">{{ p.description }}</p>

            <div class="mt-4 flex items-end gap-2">
              <div class="flex-1 relative">
                <UiInput
                  v-model="values[p.key]"
                  :type="masked[p.key] ? 'password' : 'text'"
                  :placeholder="status[p.key] === 'configured' ? 'Atualizar (deixe em branco para manter)' : 'Cole a chave aqui'"
                  autocomplete="off"
                />
                <button
                  type="button"
                  class="absolute right-2 top-9 text-ink-faint hover:text-ink"
                  @click="masked[p.key] = !masked[p.key]"
                >
                  <Eye v-if="masked[p.key]" class="h-4 w-4" />
                  <EyeOff v-else class="h-4 w-4" />
                </button>
              </div>
              <UiButton
                variant="primary"
                :loading="saving[p.key]"
                :disabled="!values[p.key]"
                @click="save(p)"
              >
                Salvar
              </UiButton>
            </div>

            <p
              v-if="message?.key === p.key"
              :class="['mt-2 text-sm', message.ok ? 'text-success' : 'text-danger']"
            >
              {{ message.text }}
            </p>
          </div>
        </div>
      </UiCard>
    </div>
  </div>
</template>
