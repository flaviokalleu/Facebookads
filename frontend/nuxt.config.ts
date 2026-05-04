export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  srcDir: 'app/',
  dir: { pages: 'pages', layouts: 'layouts' },
  devtools: { enabled: true },
  modules: [
    '@nuxtjs/tailwindcss',
    '@pinia/nuxt',
    '@nuxtjs/google-fonts',
    '@vueuse/nuxt',
  ],
  googleFonts: {
    families: { Inter: [400, 500, 600, 700] },
    display: 'swap',
    subsets: ['latin', 'latin-ext'],
  },
  css: ['~/assets/css/main.css'],
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080/api/v1',
    },
  },
  app: {
    head: {
      title: 'Gestor de Tráfego Imobiliário',
      htmlAttrs: { lang: 'pt-BR' },
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Gestão de tráfego imobiliário com IA. Em português.' },
      ],
    },
  },
})
