<script setup>
import { computed, inject, ref } from 'vue';
import { useRoute } from 'vue-router';

import { APP_STATE_KEY } from '@/store/state.js';

const state = inject(APP_STATE_KEY)
const collapsed = defineModel('collapsed', { type: Boolean, default: false })
const route = useRoute()

// Docker 子菜单展开状态
const dockerExpanded = ref(false)

// 判断当前是否在 Docker 相关路由下
const isDockerActive = computed(() => {
  return route.path.startsWith('/docker/')
})

// 切换 Docker 子菜单
const toggleDocker = () => {
  if (collapsed.value) {
    // 侧边栏折叠时，展开侧边栏并展开子菜单
    collapsed.value = false
    dockerExpanded.value = true
  } else {
    dockerExpanded.value = !dockerExpanded.value
  }
}
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
      
      <!-- Docker 折叠子菜单 -->
      <div v-if="!collapsed">
        <button 
          @click="toggleDocker"
          class="nav-link w-full"
          :class="{ 'nav-link-active': isDockerActive }"
        >
          <i class="fab fa-docker"></i>
          <span>Docker</span>
          <i 
            class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200"
            :class="{ 'rotate-180': dockerExpanded }"
          ></i>
        </button>
        <div v-show="dockerExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
          <router-link 
            to="/docker/containers" 
            class="nav-link text-sm"
            active-class="nav-link-active"
          >
            <i class="fas fa-cube"></i>
            <span>容器</span>
          </router-link>
          <router-link 
            to="/docker/images" 
            class="nav-link text-sm"
            active-class="nav-link-active"
          >
            <i class="fas fa-layer-group"></i>
            <span>镜像</span>
          </router-link>
          <router-link 
            to="/docker/networks" 
            class="nav-link text-sm"
            active-class="nav-link-active"
          >
            <i class="fas fa-network-wired"></i>
            <span>网络</span>
          </router-link>
          <router-link 
            to="/docker/volumes" 
            class="nav-link text-sm"
            active-class="nav-link-active"
          >
            <i class="fas fa-database"></i>
            <span>卷</span>
          </router-link>
        </div>
      </div>
      
      <!-- 折叠状态下的 Docker 菜单 -->
      <router-link 
        v-if="collapsed"
        to="/docker/containers" 
        class="nav-link"
        :class="{ 'nav-link-active': isDockerActive }"
        title="Docker"
      >
        <i class="fab fa-docker"></i>
      </router-link>
    </nav>

    <!-- 底部折叠按钮 -->
    <div class="border-t border-slate-200 p-3">
      <button 
        @click="collapsed = !collapsed"
        class="w-full nav-link justify-center"
        :title="collapsed ? '展开菜单' : '收起菜单'"
      >
        <i :class="collapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
        <span v-if="!collapsed">收起菜单</span>
      </button>
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
