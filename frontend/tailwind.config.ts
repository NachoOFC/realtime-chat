import { fileURLToPath } from 'node:url'

export default {
  darkMode: 'class',
  content: [
    fileURLToPath(new URL('./app.vue', import.meta.url)),
    fileURLToPath(new URL('./components/**/*.vue', import.meta.url)),
    fileURLToPath(new URL('./layouts/**/*.vue', import.meta.url)),
    fileURLToPath(new URL('./pages/**/*.vue', import.meta.url)),
  ],
}
