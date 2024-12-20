import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  base: '/pyla/',
  resolve: {
    alias: {
      vue: 'vue/dist/vue.esm-bundler.js',
    }
  }
})
