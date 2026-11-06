import { fileURLToPath } from 'node:url'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: false },
  devServer: {
    https: process.env.NUXT_DEV_TLS === 'true' ? {
      cert: fileURLToPath(new URL('../.certs/localhost.crt', import.meta.url)),
      key: fileURLToPath(new URL('../.certs/localhost.key', import.meta.url)),
    } : false,
  },
  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss'],
  tailwindcss: {
    cssPath: '~/assets/css/main.css',
    configPath: '~/tailwind.config.ts',
  },
  components: [{ path: '~/components', pathPrefix: false }],
  routeRules: {
    '/': { headers: { 'cache-control': 'private, no-store' } },
    '/login': { headers: { 'cache-control': 'private, no-store' } },
    '/chat': { headers: { 'cache-control': 'private, no-store' } },
    '/chat/**': { headers: { 'cache-control': 'private, no-store' } },
  },
  runtimeConfig: {
    apiProxyTarget: '',
    public: {
      apiBase: '',
      wsBase: 'ws://localhost:8080',
    },
  },
  app: {
    head: {
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700&display=swap' },
      ],
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
