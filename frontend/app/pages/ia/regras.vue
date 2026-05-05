<script setup lang="ts">
import { Shield, Bot, AlertTriangle } from 'lucide-vue-next'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import UiInput from '~/components/ui/UiInput.vue'
import UiButton from '~/components/ui/UiButton.vue'
import { useAiActions, type SafetyRule } from '~/composables/useAiActions'

const ai = useAiActions()
const toast = useToast()
const rules = ref<SafetyRule[]>([])
const loading = ref(true)
const editValues = reactive<Record<string, string>>({})
const saving = reactive<Record<string, boolean>>({})
const message = ref<string | null>(null)

const labels: Record<string, { title: string, hint: string, format: (v: number) => string }> = {
  // Maturidade — bloqueio antes de qualquer ação
  min_age_hours:             { title: 'Idade mínima da campanha pra IA agir',           hint: 'Padrão 7 dias. Antes disso a IA não mexe — campanha ainda em aprendizagem do Meta.',  format: (v) => `${(v / 24).toFixed(1)} dias` },
  min_conversions_to_decide: { title: 'Conversões mínimas em 7 dias pra IA agir',       hint: 'Meta exige ~50 eventos pra sair da fase de aprendizagem. Abaixo disso, leitura é ruim.', format: (v) => `${v.toFixed(0)} conversões` },
  respect_learning_phase:    { title: 'Respeitar fase de aprendizagem do Meta',         hint: '1 = sim (recomendado). 0 = ignorar e agir mesmo com poucos dados.',                    format: (v) => v >= 0.5 ? 'Ativado' : 'Desativado' },

  // Pausa
  pause_cpl_ratio:      { title: 'Pausar quando custo for X vezes a média da conta', hint: 'Ex: 3 = pausa quando o custo por contato for 3× a média',                                  format: (v) => `${v}×` },
  alert_cpl_ratio:      { title: 'Alertar quando custo for X vezes a média',         hint: 'Não pausa, só sinaliza pra revisão',                                                       format: (v) => `${v}×` },
  min_spend_to_pause:   { title: 'Gasto mínimo antes de pausar (R$)',                 hint: 'Protege contra leitura ruim de amostras pequenas',                                          format: (v) => `R$ ${v.toFixed(2)}` },
  max_pause_pct_per_day:{ title: 'Teto de pausas por dia (% dos anúncios ativos)',   hint: 'Proteção contra pausa em cascata',                                                          format: (v) => `${(v * 100).toFixed(0)}%` },

  // Sinais
  alert_ctr_min:        { title: 'CTR mínimo aceitável',                              hint: 'Abaixo disso, sinaliza atenção',                                                            format: (v) => `${(v * 100).toFixed(1)}%` },
  alert_freq_max:       { title: 'Frequência máxima antes de avisar fadiga',         hint: 'Acima disso, IA sugere rotacionar criativo',                                                format: (v) => v.toFixed(1) },

  // Escalar (vencedores)
  scale_cpl_ratio:      { title: 'Escalar quando custo estiver até X vezes a média',  hint: 'Ex: 0.6 = escala quando custo está 40% abaixo da média (vencedor).',                       format: (v) => `${v}×` },
  scale_min_ctr:        { title: 'CTR mínimo pra escalar',                             hint: 'Vencedor precisa ter CTR alto pra justificar mais verba',                                  format: (v) => `${(v * 100).toFixed(1)}%` },
  scale_max_freq:       { title: 'Frequência máxima pra escalar',                     hint: 'Acima disso, escalar verticalmente vai saturar — duplicar é melhor',                       format: (v) => v.toFixed(1) },
  scale_factor:         { title: 'Quanto aumentar (multiplicador)',                   hint: 'Ex: 1.2 = +20% de verba. Limite Meta-friendly pra não resetar aprendizagem.',              format: (v) => `${((v - 1) * 100).toFixed(0)}% a mais` },
}

async function load() {
  loading.value = true
  rules.value = await ai.listRules()
  for (const r of rules.value) {
    editValues[r.rule_key] = String(r.rule_value)
  }
  loading.value = false
}

onMounted(load)

async function save(rule: SafetyRule) {
  const v = parseFloat(editValues[rule.rule_key])
  if (Number.isNaN(v)) return
  saving[rule.rule_key] = true
  message.value = null
  try {
    await ai.setRule(rule.rule_key, v, rule.account_meta_id ?? undefined)
    message.value = 'Regra atualizada.'
    toast.success('Regra atualizada', `${labels[rule.rule_key]?.title || rule.rule_key} → ${labels[rule.rule_key]?.format(v) || v}`)
    await load()
  } catch (e: any) {
    const m = e?.data?.error?.message || 'Não foi possível salvar.'
    message.value = m
    toast.error('Não foi possível salvar', m)
  } finally {
    saving[rule.rule_key] = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-ink">Como a IA decide</h1>
      <p class="mt-1 text-sm text-ink-muted max-w-2xl">
        A IA roda em modo híbrido: pausas seguras são aplicadas automaticamente, decisões mais ousadas
        ficam pra você aprovar em <NuxtLink to="/ia/historico" class="text-accent hover:underline">Histórico</NuxtLink>.
        Aqui você ajusta os limites de segurança.
      </p>
    </div>

    <div class="grid gap-4 sm:grid-cols-2">
      <UiCard>
        <div class="flex items-start gap-3">
          <div class="rounded-full bg-success-soft p-2 text-success"><Shield class="h-5 w-5" /></div>
          <div>
            <p class="font-semibold text-ink">Aplicado automaticamente</p>
            <p class="mt-1 text-sm text-ink-muted">
              Pausa anúncio com custo por contato muito acima da média, gasto suficiente e idade ≥ 24h.
              Reversível com 1 clique.
            </p>
          </div>
        </div>
      </UiCard>
      <UiCard>
        <div class="flex items-start gap-3">
          <div class="rounded-full bg-accent-soft p-2 text-accent"><Bot class="h-5 w-5" /></div>
          <div>
            <p class="font-semibold text-ink">Proposto pra aprovação</p>
            <p class="mt-1 text-sm text-ink-muted">
              Mexer em verba, criar campanha, duplicar conjunto vencedor. Você decide.
            </p>
          </div>
        </div>
      </UiCard>
    </div>

    <UiCard>
      <h2 class="text-lg font-semibold text-ink">Limites de segurança</h2>
      <p class="mt-1 text-sm text-ink-muted">Valores em vigor. Ajuste se quiser que a IA seja mais ou menos agressiva.</p>

      <div class="mt-6 space-y-5">
        <div v-if="loading" class="text-sm text-ink-muted">Carregando…</div>

        <div v-for="rule in rules" :key="rule.rule_key" class="border-t border-border pt-5 first:border-t-0 first:pt-0">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-ink">
                {{ labels[rule.rule_key]?.title || rule.rule_key }}
              </p>
              <p class="mt-0.5 text-xs text-ink-muted">{{ labels[rule.rule_key]?.hint || rule.description }}</p>
              <p class="mt-1 text-xs text-ink-faint">
                Valor atual: <span class="font-medium text-ink">{{ labels[rule.rule_key]?.format(rule.rule_value) || rule.rule_value }}</span>
                <UiBadge v-if="rule.is_default" variant="neutral" class="ml-2">padrão</UiBadge>
                <UiBadge v-else variant="success" class="ml-2">personalizado</UiBadge>
              </p>
            </div>
            <div class="flex items-center gap-2">
              <UiInput v-model="editValues[rule.rule_key]" class="w-28" />
              <UiButton
                variant="primary"
                size="sm"
                :loading="saving[rule.rule_key]"
                @click="save(rule)"
              >
                Salvar
              </UiButton>
            </div>
          </div>
        </div>
      </div>

      <p v-if="message" class="mt-4 text-sm text-success">{{ message }}</p>
    </UiCard>

    <UiCard>
      <div class="flex items-start gap-3 text-sm text-ink-muted">
        <AlertTriangle class="mt-0.5 h-5 w-5 text-warning" />
        <p>
          A IA <strong class="text-ink">nunca</strong> mexe em campanhas com menos de 24h de vida (fase de aprendizagem)
          e respeita um teto diário de pausas para evitar cascata.
        </p>
      </div>
    </UiCard>
  </div>
</template>
