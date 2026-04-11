<script setup>
import { inject, computed } from 'vue'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const paths = computed(() => {
  if (!state.currentPath || state.currentPath === '/') return []
  return state.currentPath.split('/').filter(part => part)
})

const navigateTo = (path) => {
  actions.loadFiles(path)
}
</script>

<template>
  <nav class="mb-4" aria-label="breadcrumb">
    <ol class="flex items-center space-x-1 text-sm">
      <li>
        <a 
          class="flex items-center px-3 py-1.5 rounded-lg text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition-all duration-200"
          href="#" 
          @click="navigateTo('/')"
        >
          <i class="fas fa-home mr-1.5"></i>
          <span>首页</span>
        </a>
      </li>
      
      <template v-for="(part, index) in paths" :key="index">
        <li class="flex items-center">
          <i class="fas fa-chevron-right text-slate-300 text-xs mx-1"></i>
        </li>
        <li v-if="index < paths.length - 1">
          <a 
            class="px-3 py-1.5 rounded-lg text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition-all duration-200"
            href="#" 
            @click="navigateTo('/' + paths.slice(0, index + 1).join('/'))"
          >
            {{ part }}
          </a>
        </li>
        <li v-else class="px-3 py-1.5 rounded-lg bg-primary-50 text-primary-700 font-medium">
          {{ part }}
        </li>
      </template>
    </ol>
  </nav>
</template>
