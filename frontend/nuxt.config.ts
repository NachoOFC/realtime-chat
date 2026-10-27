export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: false },
  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss'],
  css: ['~/assets/css/main.css'],
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      wsBase: 'ws://localhost:8080',
      demoMode: false,
    },
  },
  app: {
    head: {
      title: 'Realtime Chat',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        {
          name: 'description',
          content: 'Chat 1:1 en tiempo real con Nuxt 3, Go y WebSockets.',
        },
      ],
    },
  },
})
