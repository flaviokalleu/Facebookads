<script setup lang="ts">
import UiButton from '~/components/ui/UiButton.vue'

definePageMeta({ layout: 'blank' })
</script>

<template>
  <section class="flex min-h-screen items-center justify-center bg-bg px-6">
    <div class="mx-auto max-w-3xl text-center">
      <h1 class="text-4xl font-semibold tracking-tight text-ink sm:text-5xl">
        Gestão de tráfego imobiliário com IA. Em português.
      </h1>
      <p class="mt-6 text-lg text-ink-muted">
        Cole seu acesso uma vez. A IA cuida do resto.
      </p>
      <div class="mt-10 flex justify-center">
        <NuxtLink to="/onboarding">
          <UiButton variant="primary" size="lg">Conectar minha conta Meta</UiButton>
        </NuxtLink>
      </div>
    </div>
  </section>
</template>
