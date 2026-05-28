import { resolve } from 'path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: './',
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api/': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/api/shell': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
        ws: true
      },
      '/api/ssh/to': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
        ws: true
      },
      '/api/docker/container': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
        ws: true
      }
    }
  },
  build: {
    outDir: '../public',
    emptyOutDir: false
  }
})
