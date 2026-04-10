<script setup>
import { inject } from 'vue'

import { APP_STATE_KEY } from '@/store/state.js'

import AuthLogout from '@/views/logout.vue'

const state = inject(APP_STATE_KEY)
</script>

<template>
  <nav class="sticky top-0 z-50 navbar">
    <div class="w-full px-6 py-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-8">
          <!-- Logo -->
          <a class="flex items-center space-x-2 group">
            <div class="w-10 h-10 rounded-xl bg-primary-500 flex items-center justify-center shadow-glow group-hover:shadow-glow-lg transition-all duration-300">
              <i class="fas fa-server text-white text-lg"></i>
            </div>
            <span class="text-xl font-bold gradient-text">Isrvd</span>
          </a>
          
          <!-- Navigation Links -->
          <div v-if="state.username" class="flex items-center space-x-1">
            <router-link 
              to="/explorer" 
              class="nav-link"
              active-class="nav-link-active"
            >
              <i class="fas fa-folder-open"></i>
              <span>文件管理</span>
            </router-link>
            <router-link 
              to="/markdown" 
              class="nav-link"
              active-class="nav-link-active"
            >
              <i class="fas fa-edit"></i>
              <span>Markdown</span>
            </router-link>
            <router-link 
              to="/shell" 
              class="nav-link"
              active-class="nav-link-active"
            >
              <i class="fas fa-terminal"></i>
              <span>Shell 终端</span>
            </router-link>
          </div>
        </div>

        <!-- User Info -->
        <div v-if="state.username" class="flex items-center space-x-4">
          <div class="flex items-center space-x-3 px-4 py-2 rounded-xl bg-slate-100/50">
            <div class="w-8 h-8 rounded-full bg-primary-400 flex items-center justify-center">
              <i class="fas fa-user text-white text-sm"></i>
            </div>
            <span class="text-sm font-medium text-slate-700">{{ state.username }}</span>
          </div>
          <AuthLogout />
        </div>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.nav-link {
  @apply flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-600 rounded-lg;
  @apply hover:bg-slate-100 hover:text-slate-900 transition-all duration-200;
}

.nav-link-active {
  @apply bg-primary-50 text-primary-700;
  @apply hover:bg-primary-100;
}
</style>
