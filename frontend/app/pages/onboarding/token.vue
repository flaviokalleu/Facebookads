<script setup lang="ts">
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import UiButton from '~/components/ui/UiButton.vue'
import UiTextarea from '~/components/ui/UiTextarea.vue'
import UiCard from '~/components/ui/UiCard.vue'
import UiBadge from '~/components/ui/UiBadge.vue'
import OnboardingProgress from '~/components/onboarding/OnboardingProgress.vue'
import { useOnboardingStore } from '~/stores/onboarding'

definePageMeta({ layout: 'blank' })

const onboarding = useOnboardingStore()
const api = useApi()
const apiError = ref<string | null>(null)
const submitting = ref(false)

if (!onboarding.appId || !onboarding.appSecret) {
  navigateTo('/onboarding')
}

const schema = toTypedSchema(
  z.object({
    accessToken: z.string().min(20, 'Cole o token de acesso completo'),
  }),
)

const { defineField, handleSubmit, errors } = useForm({
  validationSchema: schema,
  initialValues: { accessToken: onboarding.accessToken },
})

const [accessToken, accessTokenAttrs] = defineField('accessToken')

const onSubmit = handleSubmit(async (values) => {
  apiError.value = null
  submitting.value = true
  onboarding.accessToken = values.accessToken
  try {
    await api.post('/auth/meta/connect-v2', {
      app_id: onboarding.appId,
      app_secret: onboarding.appSecret,
      access_token: values.accessToken,
    })
    navigateTo('/onboarding/ready')
  } catch (e: any) {
    const errObj = e?.data?.error
    const msg =
      (typeof errObj === 'string' ? errObj : null) ||
      errObj?.message ||
      errObj?.code ||
      e?.data?.message ||
      e?.message ||
      'Não foi possível conectar agora.'
    apiError.value = String(msg)
  } finally {
    submitting.value = false
  }
})
</script>

<template>
  <div class="min-h-screen bg-bg">
    <OnboardingProgress :step="2" :total="3" />
    <section class="mx-auto max-w-xl px-6 py-12">
      <h1 class="text-3xl font-semibold tracking-tight text-ink">
        Cole seu token de acesso.
      </h1>
      <p class="mt-3 text-ink-muted">
        Este token autoriza o sistema a ler suas empresas, contas e anúncios.
      </p>

      <UiCard class="mt-6">
        <p class="text-sm font-medium text-ink">Onde encontro o token?</p>
        <ol class="mt-2 space-y-1 text-sm text-ink-muted list-decimal pl-5">
          <li>Acesse o
            <a
              class="text-accent hover:underline"
              href="https://developers.facebook.com/tools/explorer/"
              target="_blank"
              rel="noopener"
            >Graph API Explorer</a>.
          </li>
          <li>Selecione seu aplicativo e gere um User Access Token com as permissões de Marketing.</li>
          <li>Copie e cole abaixo. Vamos transformar em token de longa duração automaticamente.</li>
        </ol>
      </UiCard>

      <form class="mt-6 space-y-5" @submit.prevent="onSubmit">
        <UiTextarea
          v-model="accessToken"
          v-bind="accessTokenAttrs"
          label="Token de acesso"
          placeholder="EAAB..."
          :rows="6"
          :error="errors.accessToken"
        />

        <UiBadge v-if="apiError" variant="danger">{{ apiError }}</UiBadge>

        <div class="flex items-center justify-between pt-2">
          <NuxtLink to="/onboarding" class="text-sm text-ink-muted hover:text-ink">Voltar</NuxtLink>
          <UiButton type="submit" variant="primary" :loading="submitting">Conectar</UiButton>
        </div>
      </form>
    </section>
  </div>
</template>
