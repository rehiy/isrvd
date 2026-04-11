<script setup>
import { inject } from 'vue'

import { APP_STATE_KEY } from '@/store/state.js'

import AuthLogout from '@/views/logout.vue'

const state = inject(APP_STATE_KEY)
const collapsed = defineModel('collapsed', { type: Boolean, default: false })
</script>

<template>
  <aside 
    class="fixed left-0 top-0 h-screen bg-white border-r border-slate-200 z-50 flex flex-col transition-all duration-300"
    :class="collapsed ? 'w-16' : 'w-64'"
  >
    <!-- Logo 区域 -->
    <div class="h-16 flex items-center border-b border-slate-200" :class="collapsed ? 'justify-center' : 'px-4'">
      <div class="flex items-center" :class="collapsed ? '' : 'space-x-3'">
        <div class="w-10 h-10 rounded-xl bg-primary-500 flex items-center justify-center shadow-glow">
          <i class="fas fa-server text-white text-lg"></i>
        </div>
        <span v-if="!collapsed" class="text-xl font-bold gradient-text">Isrvd</span>
      </div>
    </div>

    <!-- 导航链接 -->
    <nav v-if="state.username" class="flex-1 py-4 px-3 space-y-1 overflow-y-auto">
      <router-link 
        to="/explorer" 
        class="nav-link"
        active-class="nav-link-active"
        :title="collapsed ? '文件管理' : ''"
      >
        <i class="fas fa-folder-open"></i>
        <span v-if="!collapsed">文件管理</span>
      </router-link>
      <router-link 
        to="/markdown" 
        class="nav-link"
        active-class="nav-link-active"
        :title="collapsed ? 'Markdown' : ''"
      >
        <i class="fas fa-edit"></i>
        <span v-if="!collapsed">Markdown</span>
      </router-link>
      <router-link 
        to="/shell" 
        class="nav-link"
        active-class="nav-link-active"
        :title="collapsed ? 'Shell 终端' : ''"
      >
        <i class="fas fa-terminal"></i>
        <span v-if="!collapsed">Shell 终端</span>
      </router-link>
      <router-link 
        to="/docker" 
        class="nav-link"
        active-class="nav-link-active"
        :title="collapsed ? 'Docker' : ''"
      >
        <i class="fab fa-docker"></i>
        <span v-if="!collapsed">Docker</span>
      </router-link>
    </nav>

    <!-- 底部区域 -->
    <div class="border-t border-slate-200">
      <!-- 折叠按钮 -->
      <div class="p-3">
        <button 
          @click="collapsed = !collapsed"
          class="w-full nav-link justify-center"
          :title="collapsed ? '展开菜单' : '收起菜单'"
        >
          <i :class="collapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
          <span v-if="!collapsed">收起菜单</span>
        </button>
      </div>

      <!-- 用户信息 -->
      <div v-if="state.username" class="p-3 border-t border-slate-100">
        <div 
          class="flex items-center p-2 rounded-xl bg-slate-50"
          :class="collapsed ? 'justify-center' : 'space-x-3'"
        >
          <div class="w-8 h-8 rounded-full bg-primary-400 flex items-center justify-center flex-shrink-0">
            <i class="fas fa-user text-white text-sm"></i>
          </div>
          <div v-if="!collapsed" class="flex-1 min-w-0">
            <div class="text-sm font-medium text-slate-700 truncate">{{ state.username }}</div>
          </div>
          <AuthLogout v-if="!collapsed" />
        </div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.nav-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #475569;
  border-radius: 0.75rem;
  transition: all 0.2s;
}
.nav-link:hover {
  background-color: #f1f5f9;
  color: #0f172a;
}
.nav-link-active {
  background-color: #eff6ff;
  color: #1d4ed8;
}
.nav-link-active:hover {
  background-color: #dbeafe;
}
</style>
