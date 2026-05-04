<script setup lang="ts">
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import UiButton from '~/components/ui/UiButton.vue'
import UiInput from '~/components/ui/UiInput.vue'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'

definePageMeta({ layout: 'blank' })

const { login, register } = useAuth()
const { fetchTree } = useMetaTree()
const apiError = ref<string | null>(null)
const submitting = ref(false)
const mode = ref<'login' | 'register'>('login')

const schema = computed(() =>
  toTypedSchema(
    mode.value === 'login'
      ? z.object({
          email: z.string().email('E-mail inválido'),
          password: z.string().min(6, 'Mínimo 6 caracteres'),
          name: z.string().optional(),
        })
      : z.object({
          email: z.string().email('E-mail inválido'),
          password: z.string().min(8, 'Mínimo 8 caracteres'),
          name: z.string().min(2, 'Informe seu nome'),
        }),
  ),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: schema,
  initialValues: { email: '', password: '', name: '' },
})

const [email, emailAttrs] = defineField('email')
const [password, passwordAttrs] = defineField('password')
const [name, nameAttrs] = defineField('name')

function switchMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  apiError.value = null
  resetForm()
}

function extractError(e: any): string {
  const errObj = e?.data?.error
  return (
    (typeof errObj === 'string' ? errObj : null) ||
    errObj?.message ||
    errObj?.code ||
    e?.data?.message ||
    e?.message ||
    'Não foi possível autenticar.'
  )
}

const onSubmit = handleSubmit(async (values) => {
  apiError.value = null
  submitting.value = true
  try {
    if (mode.value === 'register') {
      await register({ email: values.email, password: values.password, name: values.name || values.email })
    } else {
      await login({ email: values.email, password: values.password })
    }
    const tree = await fetchTree()
    const hasMeta = (tree.businesses?.length || 0) + (tree.personal_accounts?.length || 0) > 0
    navigateTo(hasMeta ? '/dashboard' : '/onboarding')
  } catch (e: any) {
    apiError.value = extractError(e)
  } finally {
    submitting.value = false
  }
})
</script>

<template>
  <div class="min-h-screen bg-bg-subtle flex items-center justify-center px-6 py-12">
    <div class="w-full max-w-md">
      <h1 class="text-3xl font-semibold tracking-tight text-ink text-center">
        {{ mode === 'login' ? 'Entrar no sistema' : 'Criar sua conta' }}
      </h1>
      <p class="mt-2 text-center text-ink-muted">
        {{ mode === 'login' ? 'Acesse seu painel de tráfego.' : 'É rápido — só e-mail, senha e nome.' }}
      </p>

      <UiCard class="mt-8">
        <form class="space-y-5" @submit.prevent="onSubmit">
          <UiInput
            v-if="mode === 'register'"
            v-model="name"
            v-bind="nameAttrs"
            label="Nome"
            placeholder="Seu nome"
            autocomplete="name"
            :error="errors.name"
          />
          <UiInput
            v-model="email"
            v-bind="emailAttrs"
            label="E-mail"
            type="email"
            placeholder="voce@exemplo.com"
            autocomplete="email"
            :error="errors.email"
          />
          <UiInput
            v-model="password"
            v-bind="passwordAttrs"
            label="Senha"
            type="password"
            placeholder="••••••••"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            :error="errors.password"
          />

          <UiBadge v-if="apiError" variant="danger">{{ apiError }}</UiBadge>

          <UiButton type="submit" variant="primary" class="w-full" :loading="submitting">
            {{ mode === 'login' ? 'Entrar' : 'Criar conta' }}
          </UiButton>
        </form>
      </UiCard>

      <button
        type="button"
        class="mt-6 block w-full text-center text-sm text-ink-muted hover:text-ink"
        @click="switchMode"
      >
        {{ mode === 'login' ? 'Ainda não tem conta? Cadastre-se.' : 'Já tem conta? Entrar.' }}
      </button>
    </div>
  </div>
</template>
