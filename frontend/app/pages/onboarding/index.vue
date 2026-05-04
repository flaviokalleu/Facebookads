<script setup lang="ts">
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import UiButton from '~/components/ui/UiButton.vue'
import UiInput from '~/components/ui/UiInput.vue'
import OnboardingProgress from '~/components/onboarding/OnboardingProgress.vue'
import { useOnboardingStore } from '~/stores/onboarding'

definePageMeta({ layout: 'blank' })

const onboarding = useOnboardingStore()

const schema = toTypedSchema(
  z.object({
    appId: z.string().min(6, 'Informe seu App ID'),
    appSecret: z.string().min(10, 'Informe seu App Secret'),
  }),
)

const { defineField, handleSubmit, errors } = useForm({
  validationSchema: schema,
  initialValues: { appId: onboarding.appId, appSecret: onboarding.appSecret },
})

const [appId, appIdAttrs] = defineField('appId')
const [appSecret, appSecretAttrs] = defineField('appSecret')

const onSubmit = handleSubmit((values) => {
  onboarding.appId = values.appId
  onboarding.appSecret = values.appSecret
  navigateTo('/onboarding/token')
})
</script>

<template>
  <div class="min-h-screen bg-bg">
    <OnboardingProgress :step="1" :total="3" />
    <section class="mx-auto max-w-xl px-6 py-12">
      <h1 class="text-3xl font-semibold tracking-tight text-ink">
        Vamos conectar sua conta Meta.
      </h1>
      <p class="mt-3 text-ink-muted">
        Primeiro, informe os dados do seu aplicativo no Meta para Desenvolvedores.
      </p>

      <form class="mt-8 space-y-5" @submit.prevent="onSubmit">
        <UiInput
          v-model="appId"
          v-bind="appIdAttrs"
          label="App ID"
          placeholder="000000000000000"
          :error="errors.appId"
        />
        <UiInput
          v-model="appSecret"
          v-bind="appSecretAttrs"
          label="App Secret"
          type="password"
          placeholder="seu app secret"
          hint="Este dado fica criptografado. Nunca será exibido novamente."
          :error="errors.appSecret"
        />
        <div class="flex justify-end pt-2">
          <UiButton type="submit" variant="primary">Continuar</UiButton>
        </div>
      </form>
    </section>
  </div>
</template>
