import { readFileSync } from 'node:fs'
import { resolve } from 'path'
import { load as yamlLoad } from 'js-yaml'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: './',
  plugins: [vue(), tailwindcss(), {
    name: 'icon-metadata',
    enforce: 'pre',
    load(id) {
      if (!id.endsWith('/@fortawesome/fontawesome-free/metadata/icons.yml?raw')) {
        return null
      }
      const file = id.slice(0, -'?raw'.length)
      this.addWatchFile(file)
      const icons = yamlLoad(readFileSync(file, 'utf8'))
      const metadata = Object.fromEntries(Object.entries(icons).map(([name, icon]) => [name, {
        label: icon.label,
        styles: icon.styles
      }]))
      return `export default ${JSON.stringify(JSON.stringify(metadata))}`
    }
  }],
  resolve: {
    alias: {
      '@': resolve(import.meta.dirname, 'src')
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
      '/openapi/': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/api/ssh/to': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
        ws: true
      },
      '/api/shell': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
        ws: true
      },
      '/n/': {
        target: 'http://localhost:8080',
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
    outDir: 'dist',
    emptyOutDir: true,
    rolldownOptions: {
      output: {
        strictExecutionOrder: true,
        codeSplitting: {
          includeDependenciesRecursively: false,
          groups: [{
            name: 'vendor',
            test: /[\\/]node_modules[\\/]/,
            entriesAware: true,
            maxSize: 400 * 1024
          }]
        }
      }
    }
  }
})
