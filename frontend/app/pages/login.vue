<script setup lang="ts">
definePageMeta({ layout: 'auth' })

import { useAuthStore } from '~/stores/useAuthStore'
import { LogIn, UserPlus, Eye, EyeOff } from 'lucide-vue-next'

const auth = useAuthStore()
const router = useRouter()

const mode = ref<'login' | 'register'>('login')
const name = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    if (mode.value === 'login') {
      await auth.login(email.value, password.value)
    } else {
      await auth.register(name.value, email.value, password.value)
    }
    router.push('/dashboard')
  } catch (e: any) {
    error.value = e.message ?? 'An error occurred'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="w-full max-w-sm">
    <div class="text-center mb-8">
      <div class="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-gradient-to-br from-blue-default to-blue-glow mb-3 shadow-lg shadow-blue-default/30">
        <span class="text-white font-bold text-lg">M</span>
      </div>
      <h1 class="text-primary font-bold text-xl tracking-wide">Meta Ads AI</h1>
      <p class="text-muted text-sm mt-1">AI-powered campaign orchestration</p>
    </div>

    <div class="card border-bg-border/50 shadow-xl shadow-black/20">
      <div class="flex gap-1 bg-bg-elevated/50 rounded-xl p-1 mb-6">
        <button
          v-for="tab in ['login', 'register']"
          :key="tab"
          class="flex-1 flex items-center justify-center gap-1.5 py-2 text-sm font-medium rounded-lg transition-all capitalize"
          :class="mode === tab
            ? 'bg-bg-surface text-primary shadow-sm'
            : 'text-muted hover:text-secondary'"
          @click="mode = tab as 'login' | 'register'"
        >
          <component :is="tab === 'login' ? LogIn : UserPlus" class="w-3.5 h-3.5" />
          {{ tab }}
        </button>
      </div>

      <form @submit.prevent="submit" class="space-y-4">
        <div v-if="mode === 'register'">
          <label class="text-secondary text-xs font-medium block mb-1.5">Name</label>
          <input
            v-model="name"
            type="text"
            placeholder="Your name"
            required
            class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
          />
        </div>

        <div>
          <label class="text-secondary text-xs font-medium block mb-1.5">Email</label>
          <input
            v-model="email"
            type="email"
            placeholder="you@example.com"
            required
            class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
          />
        </div>

        <div>
          <label class="text-secondary text-xs font-medium block mb-1.5">Password</label>
          <div class="relative">
            <input
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="••••••••"
              required
              minlength="8"
              class="w-full bg-bg-elevated/50 border border-bg-border rounded-xl px-3 py-2.5 pr-10 text-primary text-sm placeholder-muted/50 focus:outline-none focus:border-blue-default focus:ring-1 focus:ring-blue-default/30 transition-all"
            />
            <button type="button" @click="showPassword = !showPassword" class="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-secondary">
              <component :is="showPassword ? EyeOff : Eye" class="w-4 h-4" />
            </button>
          </div>
        </div>

        <div v-if="error" class="text-red-400 text-xs bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
          {{ error }}
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-gradient-to-r from-blue-default to-blue-bright hover:from-blue-bright hover:to-blue-glow text-white font-semibold py-2.5 rounded-xl transition-all text-sm disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-blue-default/20 hover:shadow-blue-default/40"
        >
          <span v-if="loading" class="flex items-center justify-center gap-2">
            <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            Loading…
          </span>
          <span v-else>{{ mode === 'login' ? 'Sign In' : 'Create Account' }}</span>
        </button>
      </form>
    </div>
  </div>
</template>
