<script lang="ts">
import { Component, Inject, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import { APP_STATE_KEY } from '@/store/state'

@Component({
    expose: ['toggleMobileSidebar', 'closeMobileSidebar', 'openMobileSidebar'],
    emits: ['update:collapsed']
})
class NavigationBar extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: any
    @Prop({ type: Boolean, default: false }) readonly collapsed!: boolean

    // ─── 数据属性 ───
    mobileSidebarVisible = false
    dockerExpanded = false
    swarmExpanded = false
    apisixExpanded = false

    // ─── 计算属性 ───
    get isDockerActive() {
        return this.isActive('/docker/')
    }

    get isSwarmSubActive() {
        return this.isActive('/swarm/')
    }

    get isApisixActive() {
        return this.isActive('/apisix/')
    }

    // ─── 监听器 ───
    @Watch('isDockerActive', { immediate: true })
    onDockerActiveChange(isActive: boolean) {
        if (isActive && !this.collapsed) {
            this.dockerExpanded = true
        }
    }

    @Watch('isSwarmSubActive', { immediate: true })
    onSwarmActiveChange(isActive: boolean) {
        if (isActive && !this.collapsed) {
            this.swarmExpanded = true
        }
    }

    @Watch('isApisixActive', { immediate: true })
    onApisixActiveChange(isActive: boolean) {
        if (isActive && !this.collapsed) {
            this.apisixExpanded = true
        }
    }

    // ─── 方法 ───
    isActive(prefix: string) {
        return this.$route.path.startsWith(prefix)
    }

    toggleDocker() {
        if (this.collapsed) {
            this.$emit('update:collapsed', false)
            this.dockerExpanded = true
        } else {
            this.dockerExpanded = !this.dockerExpanded
        }
    }

    toggleSwarm() {
        if (this.collapsed) {
            this.$emit('update:collapsed', false)
            this.swarmExpanded = true
        } else {
            this.swarmExpanded = !this.swarmExpanded
        }
    }

    toggleApisix() {
        if (this.collapsed) {
            this.$emit('update:collapsed', false)
            this.apisixExpanded = true
        } else {
            this.apisixExpanded = !this.apisixExpanded
        }
    }

    toggleMobileSidebar() {
        this.mobileSidebarVisible = !this.mobileSidebarVisible
    }

    closeMobileSidebar() {
        this.mobileSidebarVisible = false
    }

    openMobileSidebar() {
        this.mobileSidebarVisible = true
    }

    handleResize() {
        if (window.innerWidth >= 768) {
            this.mobileSidebarVisible = false
        }
    }

    // ─── 生命周期 ───
    mounted() {
        // 根据当前路由初始化子菜单展开状态
        this.dockerExpanded = this.$route.path.startsWith('/docker/')
        this.swarmExpanded = this.$route.path.startsWith('/swarm/')
        this.apisixExpanded = this.$route.path.startsWith('/apisix/')
        window.addEventListener('resize', this.handleResize)
    }

    unmounted() {
        window.removeEventListener('resize', this.handleResize)
    }
}

export default toNative(NavigationBar)
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
      <div v-if="state.serviceAvailability.docker">
        <!-- 折叠状态：只显示图标 -->
        <router-link
          v-if="collapsed"
          to="/overview"
          class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/') }"
          title="Docker"
        >
          <i class="fab fa-docker"></i>
        </router-link>
        <!-- 展开状态：显示完整子菜单 -->
        <template v-else>
          <button
            @click.stop="toggleDocker"
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
        </template>
      </div>

      <!-- Swarm 折叠子菜单 -->
      <div v-if="state.serviceAvailability.swarm">
        <!-- 折叠状态：只显示图标 -->
        <router-link
          v-if="collapsed"
          to="/overview"
          class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm') }"
          title="Swarm 集群"
        >
          <i class="fas fa-circle-nodes"></i>
        </router-link>
        <!-- 展开状态：显示完整子菜单 -->
        <template v-else>
          <button
            @click.stop="toggleSwarm"
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
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': $route.path === '/swarm/tasks' }"
            >
              <i class="fas fa-list-check"></i>
              <span>任务</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Apisix 折叠子菜单 -->
      <div v-if="state.serviceAvailability.apisix">
        <!-- 折叠状态：只显示图标 -->
        <router-link
          v-if="collapsed"
          to="/overview"
          class="flex items-center gap-3 px-3 py-3 text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
          :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/') }"
          title="Apisix"
        >
          <i class="fas fa-cloud"></i>
        </router-link>
        <!-- 展开状态：显示完整子菜单 -->
        <template v-else>
          <button
            @click.stop="toggleApisix"
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
        </template>
      </div>
    </nav>

    <!-- 底部折叠按钮 -->
    <div class="border-t border-slate-200 p-3">
      <button 
        @click="$emit('update:collapsed', !collapsed)"
        class="flex items-center justify-center gap-3 px-3 py-3 w-full text-sm font-medium text-slate-600 rounded-xl transition-all duration-200 hover:bg-slate-100 hover:text-slate-900"
        :title="collapsed ? '展开菜单' : '收起菜单'"
      >
        <i :class="collapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
        <span v-if="!collapsed">收起菜单</span>
      </button>
    </div>
  </aside>
</template>
