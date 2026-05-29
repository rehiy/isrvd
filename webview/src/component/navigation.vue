<script lang="ts">
import { Component, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

@Component({
    expose: ['toggleMobileSidebar', 'closeMobileSidebar', 'openMobileSidebar'],
    emits: ['update:collapsed']
})
class NavigationBar extends Vue {
    portal = usePortal()
    @Prop({ type: Boolean, default: false }) readonly collapsed!: boolean

    // ─── 数据属性 ───
    mobileSidebarVisible = false
    localExpanded = false
    apisixExpanded = false
    caddyExpanded = false
    dockerExpanded = false
    swarmExpanded = false

    // ─── 计算属性 ───
    get isLocalActive() {
        return this.isActive('/local/explorer') || this.isActive('/local/shell')
    }

    get isApisixActive() {
        return this.isActive('/apisix/')
    }

    get isCaddyActive() {
        return this.isActive('/caddy/')
    }

    get isDockerActive() {
        return this.isActive('/docker/')
    }

    get isSwarmSubActive() {
        return this.isActive('/swarm/')
    }

    // Compose 部署菜单可见性
    get composeDeployVisible() {
        return this.portal.hasPerm('POST /api/compose/docker/deploy')
    }

    // ─── 监听器 ───
    @Watch('isLocalActive', { immediate: true })
    onLocalActiveChange(isActive: boolean) {
        if (isActive && !this.collapsed) {
            this.localExpanded = true
        }
    }

    @Watch('isApisixActive', { immediate: true })
    onApisixActiveChange(isActive: boolean) {
        if (isActive && !this.collapsed) {
            this.apisixExpanded = true
        }
    }

    @Watch('isCaddyActive', { immediate: true })
    onCaddyActiveChange(isActive: boolean) {
        if (isActive && !this.collapsed) {
            this.caddyExpanded = true
        }
    }

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

    // ─── 方法 ───
    isActive(prefix: string) {
        return this.$route.path.startsWith(prefix)
    }

    toggleLocal() {
        if (this.collapsed) {
            this.$emit('update:collapsed', false)
            this.localExpanded = true
        } else {
            this.localExpanded = !this.localExpanded
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

    toggleCaddy() {
        if (this.collapsed) {
            this.$emit('update:collapsed', false)
            this.caddyExpanded = true
        } else {
            this.caddyExpanded = !this.caddyExpanded
        }
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
        if (window.innerWidth >= 1024) {
            this.mobileSidebarVisible = false
        }
    }

    // ─── 生命周期 ───
    mounted() {
        // 根据当前路由初始化子菜单展开状态
        this.localExpanded = this.$route.path.startsWith('/local/explorer') || this.$route.path.startsWith('/local/shell')
        this.apisixExpanded = this.$route.path.startsWith('/apisix/')
        this.caddyExpanded = this.$route.path.startsWith('/caddy/')
        this.dockerExpanded = this.$route.path.startsWith('/docker/')
        this.swarmExpanded = this.$route.path.startsWith('/swarm/')
        window.addEventListener('resize', this.handleResize)
    }

    unmounted() {
        window.removeEventListener('resize', this.handleResize)
    }
}

export default toNative(NavigationBar)
</script>

<template>
  <aside 
    class="fixed left-0 top-0 h-screen border-r border-slate-200/50 z-50 flex flex-col transition-all duration-300"
    :class="[
      collapsed ? 'w-16' : 'w-64',
      mobileSidebarVisible ? 'translate-x-0 bg-white/95 backdrop-blur-xl shadow-2xl' : '-translate-x-full lg:translate-x-0 lg:bg-white/80 lg:backdrop-blur-xl'
    ]"
  >
    <!-- Logo 区域 -->
    <div class="h-16 flex items-center border-b border-slate-200/50" :class="collapsed ? 'justify-center' : 'px-4'">
      <div class="flex items-center space-x-3 flex-1 min-w-0" :class="collapsed ? 'justify-center flex-none' : ''">
        <img
          src="@/assets/logo.svg"
          alt="iSrvd"
          class="flex-shrink-0 transition-all duration-300"
          :class="collapsed ? 'w-9 h-9 object-cover object-left' : 'w-auto h-9 max-w-[11rem] object-contain'"
        >
      </div>
      <!-- 移动端关闭按钮，仅展开状态下显示 -->
      <button v-if="!collapsed" class="btn-icon btn-icon-slate lg:hidden ml-2" @click="closeMobileSidebar">
        <i class="fas fa-times text-sm"></i>
      </button>
    </div>

    <!-- 导航链接 -->
    <nav v-if="portal.username" class="flex-1 py-4 px-3 space-y-1 overflow-y-auto" @click="closeMobileSidebar">
      <router-link to="/overview" class="nav-link" active-class="bg-blue-50 text-blue-700" :title="collapsed ? '概览' : ''">
        <i class="fas fa-gauge-high"></i>
        <span v-if="!collapsed">概览</span>
      </router-link>
      <!-- 本机管理折叠子菜单 -->
      <div v-if="portal.hasPerm('GET /api/filer/list') || portal.hasPerm('GET /api/shell')">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" class="nav-link justify-center" :class="{ 'bg-blue-50 text-blue-700': isLocalActive }" title="本机管理" @click.stop="toggleLocal">
          <i class="fas fa-computer"></i>
        </button>
        <!-- 展开状态：显示完整子菜单 -->
        <template v-else>
          <button class="nav-link w-full" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isLocalActive }" @click.stop="toggleLocal">
            <i class="fas fa-computer"></i>
            <span>本机管理</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': localExpanded }"></i>
          </button>
          <div v-show="localExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link v-if="portal.hasPerm('GET /api/filer/list')" to="/local/explorer" class="nav-link" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/local/explorer') }">
              <i class="fas fa-folder-open"></i>
              <span>文件管理</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/shell')" to="/local/shell" class="nav-link" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/local/shell') }">
              <i class="fas fa-terminal"></i>
              <span>Shell 终端</span>
            </router-link>
          </div>
        </template>
      </div>
      <!-- APISIX 折叠子菜单 -->
      <div v-if="portal.hasPerm('apisix')">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" class="nav-link justify-center" :class="{ 'bg-blue-50 text-blue-700': isActive('/apisix/') }" title="APISIX 网关" @click.stop="toggleApisix">
          <i class="fas fa-cloud"></i>
        </button>
        <!-- 有权限：展开状态显示完整子菜单 -->
        <template v-else>
          <button class="nav-link w-full" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/') }" @click.stop="toggleApisix">
            <i class="fas fa-cloud"></i>
            <span>APISIX 网关</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': apisixExpanded }"></i>
          </button>
          <div v-show="apisixExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link
              v-if="portal.hasPerm('GET /api/apisix/routes')"
              to="/apisix/routes"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/route') }"
            >
              <i class="fas fa-route"></i>
              <span>路由</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/apisix/upstreams')"
              to="/apisix/upstreams"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/upstream') }"
            >
              <i class="fas fa-diagram-project"></i>
              <span>上游</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/apisix/consumers')"
              to="/apisix/consumers"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/consumer') }"
            >
              <i class="fas fa-users"></i>
              <span>消费者</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/apisix/whitelist')"
              to="/apisix/whitelist"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/whitelist') }"
            >
              <i class="fas fa-shield-halved"></i>
              <span>白名单</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/apisix/ssls')" to="/apisix/ssls" class="nav-link" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/ssl') }">
              <i class="fas fa-certificate"></i>
              <span>SSL 证书</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/apisix/plugin-configs')"
              to="/apisix/plugin-configs"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/apisix/plugin-config') }"
            >
              <i class="fas fa-puzzle-piece"></i>
              <span>插件配置</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Caddy 折叠子菜单 -->
      <div v-if="portal.hasPerm('caddy')">
        <button v-if="collapsed" class="nav-link justify-center" :class="{ 'bg-blue-50 text-blue-700': isActive('/caddy/') }" title="Caddy 网关" @click.stop="toggleCaddy">
          <i class="fas fa-shield"></i>
        </button>
        <template v-else>
          <button class="nav-link w-full" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/caddy/') }" @click.stop="toggleCaddy">
            <i class="fas fa-shield"></i>
            <span>Caddy 网关</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': caddyExpanded }"></i>
          </button>
          <div v-show="caddyExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link
              v-if="portal.hasPerm('GET /api/caddy/routes')"
              to="/caddy/routes"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/caddy/route') }"
            >
              <i class="fas fa-route"></i>
              <span>路由</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/caddy/certs')" to="/caddy/certs" class="nav-link" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/caddy/cert') }">
              <i class="fas fa-certificate"></i>
              <span>TLS 证书</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/caddy/global')"
              to="/caddy/global"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/caddy/global') }"
            >
              <i class="fas fa-sliders"></i>
              <span>全局选项</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/caddy/config')" to="/caddy/raw" class="nav-link" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/caddy/raw') }">
              <i class="fas fa-code"></i>
              <span>原始配置</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Docker 折叠子菜单 -->
      <div v-if="portal.hasPerm('docker')">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" class="nav-link justify-center" :class="{ 'bg-blue-50 text-blue-700': isActive('/docker/') }" title="Docker 服务" @click.stop="toggleDocker">
          <i class="fab fa-docker"></i>
        </button>
        <!-- 展开状态：显示完整子菜单 -->
        <template v-else>
          <button class="nav-link w-full" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/') }" @click.stop="toggleDocker">
            <i class="fab fa-docker"></i>
            <span>Docker 服务</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': dockerExpanded }"></i>
          </button>
          <div v-show="dockerExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link
              v-if="portal.hasPerm('GET /api/docker/containers')"
              to="/docker/containers"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/container') }"
            >
              <i class="fas fa-cube"></i>
              <span>容器</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/networks')"
              to="/docker/networks"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/network') }"
            >
              <i class="fas fa-network-wired"></i>
              <span>网络</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/volumes')"
              to="/docker/volumes"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/volume') }"
            >
              <i class="fas fa-database"></i>
              <span>存储</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/images')"
              to="/docker/images"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/image') }"
            >
              <i class="fas fa-layer-group"></i>
              <span>镜像</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/registries')"
              to="/docker/registries"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/docker/registr') }"
            >
              <i class="fas fa-warehouse"></i>
              <span>镜像源</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Swarm 折叠子菜单 -->
      <div v-if="portal.hasPerm('swarm')">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" class="nav-link justify-center" :class="{ 'bg-blue-50 text-blue-700': isActive('/swarm') }" title="Swarm 集群" @click.stop="toggleSwarm">
          <i class="fas fa-circle-nodes"></i>
        </button>
        <!-- 有权限：展开状态显示完整子菜单 -->
        <template v-else>
          <button class="nav-link w-full" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm') }" @click.stop="toggleSwarm">
            <i class="fas fa-circle-nodes"></i>
            <span>Swarm 集群</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': swarmExpanded }"></i>
          </button>
          <div v-show="swarmExpanded" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link v-if="portal.hasPerm('GET /api/swarm/nodes')" to="/swarm/nodes" class="nav-link" :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm/node') }">
              <i class="fas fa-server"></i>
              <span>节点</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/swarm/services')"
              to="/swarm/services"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': isActive('/swarm/service') }"
            >
              <i class="fas fa-cubes"></i>
              <span>服务</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/swarm/tasks')"
              to="/swarm/tasks"
              class="nav-link"
              :class="{ 'bg-blue-50 text-blue-700 hover:bg-blue-100': $route.path === '/swarm/tasks' }"
            >
              <i class="fas fa-list-check"></i>
              <span>任务</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Compose 部署 -->
      <router-link v-if="composeDeployVisible" to="/compose/deploy" class="nav-link" active-class="bg-blue-50 text-blue-700" :title="collapsed ? 'Compose 部署' : ''">
        <i class="fas fa-file-code"></i>
        <span v-if="!collapsed">Compose 部署</span>
      </router-link>
      <!-- SSH/SFTP 连接 -->
      <router-link v-if="portal.hasPerm('GET /api/ssh/hosts')" to="/ssh/hosts" class="nav-link" :class="{ 'bg-blue-50 text-blue-700': isActive('/ssh') }" :title="collapsed ? 'SSH/SFTP 连接' : ''">
        <i class="fas fa-server"></i>
        <span v-if="!collapsed">SSH/SFTP 连接</span>
      </router-link>

      <!-- 计划任务 -->
      <router-link v-if="portal.hasPerm('GET /api/cron/jobs')" to="/cron/jobs" class="nav-link" active-class="bg-blue-50 text-blue-700" :title="collapsed ? '计划任务' : ''">
        <i class="fas fa-clock"></i>
        <span v-if="!collapsed">计划任务</span>
      </router-link>

      <!-- 操作审计 -->
      <router-link v-if="portal.hasPerm('GET /api/system/audit/logs')" to="/system/audit/logs" class="nav-link" active-class="bg-blue-50 text-blue-700" :title="collapsed ? '操作审计' : ''">
        <i class="fas fa-clipboard-list"></i>
        <span v-if="!collapsed">操作审计</span>
      </router-link>

      <!-- 用户管理 -->
      <router-link v-if="portal.hasPerm('GET /api/account/members')" to="/account/members" class="nav-link" active-class="bg-blue-50 text-blue-700" :title="collapsed ? '用户管理' : ''">
        <i class="fas fa-users"></i>
        <span v-if="!collapsed">用户管理</span>
      </router-link>

      <!-- 系统配置 -->
      <router-link v-if="portal.hasPerm('PUT /api/system/config')" to="/system/config" class="nav-link" active-class="bg-blue-50 text-blue-700" :title="collapsed ? '系统配置' : ''">
        <i class="fas fa-gear"></i>
        <span v-if="!collapsed">系统配置</span>
      </router-link>
    </nav>

    <!-- 底部工具条：GitHub 链接 + 折叠按钮 -->
    <div class="border-t border-slate-200/50 p-3" :class="collapsed ? 'space-y-1' : 'flex items-center gap-2'">
      <a
        href="https://github.com/rehiy/isrvd"
        target="_blank"
        rel="noopener noreferrer"
        class="btn-icon btn-icon-slate"
        :class="collapsed ? 'w-full h-10' : 'w-10 h-10 flex-shrink-0'"
        title="GitHub 仓库"
      >
        <i class="fab fa-github text-base"></i>
      </a>
      <button
        class="btn-icon btn-icon-slate"
        :class="collapsed ? 'w-full h-10' : 'flex-1 h-10 text-xs font-medium'"
        :title="collapsed ? '展开菜单' : '收起菜单'"
        @click="$emit('update:collapsed', !collapsed)"
      >
        <i :class="collapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
        <span v-if="!collapsed">收起菜单</span>
      </button>
    </div>
  </aside>

  <!-- 移动端遮罩层 -->
  <div v-if="mobileSidebarVisible" class="fixed inset-0 bg-black/30 backdrop-blur-sm z-30 lg:hidden" @click="closeMobileSidebar"></div>
</template>
