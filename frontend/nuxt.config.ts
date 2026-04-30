export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  modules: [
    '@nuxtjs/tailwindcss',
    '@pinia/nuxt',
    '@vueuse/motion/nuxt',
  ],

  // Nuxt 4 app/ directory convention
  future: {
    compatibilityVersion: 4,
  },

  // Runtime config — only public (non-secret) values here
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080/api/v1',
    },
  },

  tailwindcss: {
    cssPath: '~/assets/css/main.css',
    configPath: 'tailwind.config.ts',
  },

  typescript: {
    strict: true,
    typeCheck: false,
  },

  // Disable SSR for dashboard (SPA mode — all auth-protected)
  ssr: false,
})
