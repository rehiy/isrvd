<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'

@Component
class Breadcrumb extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: any
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 计算属性 ───
    get paths() {
        if (!this.state.currentPath || this.state.currentPath === '/') return []
        return this.state.currentPath.split('/').filter((part: string) => part)
    }

    // ─── 方法 ───
    navigateTo(path: string) {
        this.actions.loadFiles(path)
    }
}

export default toNative(Breadcrumb)
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
