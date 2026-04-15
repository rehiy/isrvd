<script setup>
import { computed, inject, ref, watch, onMounted, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';

import { APP_STATE_KEY } from '@/store/state.js';

const state = inject(APP_STATE_KEY)
const collapsed = defineModel('collapsed', { type: Boolean, default: false })
const route = useRoute()

// 移动端侧边栏显示状态
const mobileSidebarVisible = ref(false)

// Docker 子菜单展开状态 - 初始化时根据当前路由判断
const dockerExpanded = ref(route.path.startsWith('/docker/'))

// Swarm 子菜单展开状态
const swarmExpanded = ref(route.path.startsWith('/swarm/'))

// Apisix 子菜单展开状态
const apisixExpanded = ref(route.path.startsWith('/apisix/'))

// 统一高亮判断：当前路由是否以指定前缀开头
const isActive = (prefix) => route.path.startsWith(prefix)

// 用于 watch 的 computed（仍需响应式）
const isDockerActive = computed(() => isActive('/docker/'))
const isSwarmSubActive = computed(() => isActive('/swarm/'))
const isApisixActive = computed(() => isActive('/apisix/'))

// 监听路由变化，自动展开子菜单
watch(isDockerActive, (isActive) => {
  if (isActive && !collapsed.value) {
    dockerExpanded.value = true
  }
}, { immediate: true })

watch(isSwarmSubActive, (isActive) => {
  if (isActive && !collapsed.value) {
    swarmExpanded.value = true
  }
}, { immediate: true })

watch(isApisixActive, (isActive) => {
  if (isActive && !collapsed.value) {
    apisixExpanded.value = true
  }
}, { immediate: true })

// 切换 Docker 子菜单
const toggleDocker = () => {
  if (collapsed.value) {
    collapsed.value = false
    dockerExpanded.value = true
  } else {
    dockerExpanded.value = !dockerExpanded.value
  }
}

// 切换 Swarm 子菜单
const toggleSwarm = () => {
  if (collapsed.value) {
    collapsed.value = false
    swarmExpanded.value = true
  } else {
    swarmExpanded.value = !swarmExpanded.value
  }
}

// 切换 Apisix 子菜单
const toggleApisix = () => {
  if (collapsed.value) {
    collapsed.value = false
    apisixExpanded.value = true
  } else {
    apisixExpanded.value = !apisixExpanded.value
  }
}

// 移动端侧边栏控制
const toggleMobileSidebar = () => {
  mobileSidebarVisible.value = !mobileSidebarVisible.value
}

const closeMobileSidebar = () => {
  mobileSidebarVisible.value = false
}

// 提供给父组件调用的方法
const openMobileSidebar = () => {
  mobileSidebarVisible.value = true
}

// 暴露方法给父组件
defineExpose({
  openMobileSidebar,
  closeMobileSidebar,
  toggleMobileSidebar
})

// 监听窗口大小变化，自动关闭移动端侧边栏
const handleResize = () => {
  if (window.innerWidth >= 768) {
    mobileSidebarVisible.value = false
  }
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<template>
  <!-- 移动端遮罩层 -->
  <div 
    v-if="mobileSidebarVisible"
    class="fixed inset-0 bg-black bg-opacity-50 z-40 md:hidden"
    @click="closeMobileSidebar"
  ></div>
  
  <aside 
    class="fixed left-0 top-0 h-screen bg-white border-r border-slate-200 z-50 flex flex-col transition-all duration-300"
    :class="[collapsed ? 'w-16' : 'w-64', mobileSidebarVisible ? 'translate-x-0' : '-translate-x-full md:translate-x-0']"
  >
    <!-- Logo 区域 -->
    <div class="h-16 flex items-center border-b border-slate-200" :class="collapsed ? 'justify-center' : 'px-4'" @click="closeMobileSidebar">
      <div class="flex items-center" :class="collapsed ? '' : 'space-x-3'">
        <div class="w-10 h-10 rounded-xl bg-primary-500 flex items-center justify-center shadow-glow">
          <i class="fas fa-server text-white text-lg"></i>
        </div>
        <span v-if="!collapsed" class="text-xl font-bold gradient-text">Isrvd</span>
      </div>
    </div>

    <!-- 移动端关闭按钮 -->
    <div class="md:hidden absolute top-4 right-4">
      <button 
        @click="closeMobileSidebar"
        class="w-8 h-8 rounded-lg bg-slate-100 hover:bg-slate-200 flex items-center justify-center text-slate-600 transition-colors"
      >
        <i class="fas fa-times text-sm"></i>
      </button>
    </div>

    <!-- 导航链接 -->
    <nav v-if="state.username" class="flex-1 py-4 px-3 space-y-1 overflow-y-auto" @click="closeMobileSidebar">
      <router-link 
        to="/overview" 
        class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
        active-class="bg-blue-50 text-blue-700 hover:bg-blue-100"
        :title="collapsed ? '概览' : ''"
      >
        <i class="fas fa-gauge-high"></i>
        <span v-if="!collapsed">概览</span>
      </router-link>
      <router-link 
        to="/explorer" 
        class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
        active-class="bg-blue-50 text-blue-700 hover:bg-blue-100"
        :title="collapsed ? '文件管理' : ''"
      >
        <i class="fas fa-folder-open"></i>
        <span v-if="!collapsed">文件管理</span>
      </router-link>
      <router-link 
        to="/shell" 
        class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
        active-class="bg-blue-50 text-blue-700 hover:bg-blue-100"
        :title="collapsed ? 'Shell 终端' : ''"
      >
        <i class="fas fa-terminal"></i>
        <span v-if="!collapsed">Shell 终端</span>
      </router-link>
      
      <!-- Docker 折叠子菜单 -->
      <div v-if="!collapsed">
        <button 
          @click="toggleDocker"
          class="flex items-center gap-3 px-3 py-3 w-full text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/') }"
        >
          <i class="fab fa-docker"></i>
          <span>Docker 管理</span>
          <i 
            class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200"
            :class="{ 'rotate-180': dockerExpanded }"
          ></i>
        </button>
        <div v-show="dockerExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
          <router-link
            to="/docker/containers" 
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/container') }"
          >
            <i class="fas fa-cube"></i>
            <span>容器</span>
          </router-link>
          <router-link 
            to="/docker/networks" 
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/network') }"
          >
            <i class="fas fa-network-wired"></i>
            <span>网络</span>
          </router-link>
          <router-link 
            to="/docker/images" 
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/image') }"
          >
            <i class="fas fa-layer-group"></i>
            <span>镜像</span>
          </router-link>
          <router-link 
            to="/docker/registries"
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/registr') }"
          >
            <i class="fas fa-warehouse"></i>
            <span>镜像源</span>
          </router-link>
          <router-link 
            to="/docker/volumes"
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/volume') }"
          >
            <i class="fas fa-database"></i>
            <span>存储卷</span>
          </router-link>
        </div>
      </div>
      
      <!-- 折叠状态下的 Docker 菜单 -->
      <router-link 
        v-if="collapsed"
        to="/overview"
        class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/') }"
        title="Docker"
      >
        <i class="fab fa-docker"></i>
      </router-link>

      <!-- Swarm 折叠子菜单 -->
      <div v-if="!collapsed">
        <button
          @click="toggleSwarm"
          class="flex items-center gap-3 px-3 py-3 w-full text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm') }"
        >
          <i class="fas fa-circle-nodes"></i>
          <span>Swarm 集群</span>
          <i
            class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200"
            :class="{ 'rotate-180': swarmExpanded }"
          ></i>
        </button>
        <div v-show="swarmExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
          <router-link
            to="/swarm/nodes"
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm/node') }"
          >
            <i class="fas fa-server"></i>
            <span>节点</span>
          </router-link>
          <router-link
            to="/swarm/services"
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm/service') }"
          >
            <i class="fas fa-cubes"></i>
            <span>服务</span>
          </router-link>
          <router-link
            to="/swarm/tasks"
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm/task') }"
          >
            <i class="fas fa-tasks"></i>
            <span>任务</span>
          </router-link>
        </div>
      </div>
      <!-- 折叠状态下的 Swarm 菜单 -->
      <router-link
        v-if="collapsed"
        to="/overview"
        class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm') }"
        title="Swarm 集群"
      >
        <i class="fas fa-circle-nodes"></i>
      </router-link>

      <!-- Apisix 折叠子菜单 -->
      <div v-if="!collapsed">
        <button 
          @click="toggleApisix"
          class="flex items-center gap-3 px-3 py-3 w-full text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/') }"
        >
          <i class="fas fa-cloud"></i>
          <span>Apisix 管理</span>
          <i 
            class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200"
            :class="{ 'rotate-180': apisixExpanded }"
          ></i>
        </button>
        <div v-show="apisixExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
          <router-link 
            to="/apisix/routes" 
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/route') }"
          >
            <i class="fas fa-route"></i>
            <span>路由</span>
          </router-link>
          <router-link 
            to="/apisix/consumers" 
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/consumer') }"
          >
            <i class="fas fa-users"></i>
            <span>用户</span>
          </router-link>
          <router-link 
            to="/apisix/whitelist" 
            class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
            :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/whitelist') }"
          >
            <i class="fas fa-shield-halved"></i>
            <span>白名单</span>
          </router-link>
        </div>
      </div>

      <!-- 折叠状态下的 Apisix 菜单 -->
      <router-link
        v-if="collapsed"
        to="/overview"
        class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/') }"
        title="Apisix"
      >
        <i class="fas fa-cloud"></i>
      </router-link>
    </nav>

    <!-- 底部折叠按钮 -->
    <div class="border-t border-slate-200 p-3">
      <button 
        @click="collapsed = !collapsed"
        class="flex items-center justify-center gap-3 px-3 py-3 w-full text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
        :title="collapsed ? '展开菜单' : '收起菜单'"
      >
        <i :class="collapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
        <span v-if="!collapsed">收起菜单</span>
      </button>
    </div>
  </aside>
</template>
