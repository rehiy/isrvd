<script lang="ts">
import { Component, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import { configGroups, usePortal } from '@/stores'

import { canAccessRoute } from '@/router'

import { NODE_LIST_PERM } from '@/helper/node'

const MENU_PREFIXES = {
    local: '/local/',
    apisix: '/apisix/',
    caddy: '/caddy/',
    docker: '/docker/',
    swarm: '/swarm/',
    compose: '/compose/',
    ssh: '/ssh/',
    node: '/node',
    config: '/system/config'
} as const

type Menu = keyof typeof MENU_PREFIXES
const menus = Object.keys(MENU_PREFIXES) as Menu[]

@Component({
    expose: ['toggleMobileSidebar', 'closeMobileSidebar', 'openMobileSidebar'],
    emits: ['update:collapsed']
})
class NavigationBar extends Vue {
    portal = usePortal()
    @Prop({ type: Boolean, default: false }) readonly collapsed!: boolean

    get configGroups() {
        const canViewConfig = this.portal.hasPerm('GET /api/system/config') || this.portal.hasPerm('PUT /api/system/config')
        return configGroups.filter(item => canViewConfig || item.id === 'metrics' && this.portal.hasPerm('metrics'))
    }

    get apisixLinks() {
        const links = [
            { path: '/apisix/routes', label: '路由', icon: 'fa-route' },
            { path: '/apisix/upstreams', label: '上游', icon: 'fa-server' },
            { path: '/apisix/consumers', label: '消费者', icon: 'fa-users' },
            { path: '/apisix/whitelist', label: '访问授权', icon: 'fa-shield-halved' },
            { path: '/apisix/ssls', label: 'SSL 证书', icon: 'fa-certificate' },
            { path: '/apisix/plugin-configs', label: '插件配置', icon: 'fa-puzzle-piece' },
        ]
        return links.filter(link => canAccessRoute(this.$router.resolve(link.path), this.portal.hasPerm))
    }

    mobileSidebarVisible = false
    expanded = new Set<Menu>()

    get canListNodes() {
        return this.portal.hasPerm(NODE_LIST_PERM)
    }

    get nodeVisible() {
        return !this.portal.currentNodeId && this.canListNodes
    }

    get composeDeployVisible() {
        return this.portal.hasPerm('POST /api/compose/docker')
    }

    get composeSwarmVisible() {
        return this.portal.hasPerm('POST /api/compose/swarm')
    }

    get composeVisible() {
        return this.composeDeployVisible || this.composeSwarmVisible
    }

    @Watch('$route.path', { immediate: true })
    onRoutePathChange(path: string, previousPath = '') {
        if (this.collapsed) {
            return
        }
        for (const menu of menus) {
            const prefix = MENU_PREFIXES[menu]
            if (path.startsWith(prefix) && !previousPath.startsWith(prefix)) {
                this.expanded.add(menu)
            }
        }
    }

    isActive(prefix: string) {
        return this.$route.path.startsWith(prefix)
    }

    isMenuActive(menu: Menu) {
        return this.isActive(MENU_PREFIXES[menu])
    }

    toggleMenu(menu: Menu) {
        if (this.collapsed) {
            this.$emit('update:collapsed', false)
            this.expanded.add(menu)
        } else if (this.expanded.has(menu)) {
            this.expanded.delete(menu)
        } else {
            this.expanded.add(menu)
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
        this.expanded.clear()
        for (const menu of menus) {
            if (this.isMenuActive(menu)) {
                this.expanded.add(menu)
            }
        }
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
    class="fixed left-0 inset-y-0 z-50 flex flex-col transition-all duration-300"
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
      <button v-if="!collapsed" type="button" class="btn-icon btn-icon-slate lg:hidden ml-2" @click="closeMobileSidebar">
        <i class="fas fa-times text-sm"></i>
      </button>
    </div>

    <!-- 导航链接 -->
    <nav v-if="portal.username" class="flex-1 py-4 px-3 space-y-1 overflow-y-auto" @click="closeMobileSidebar">
      <router-link to="/overview" class="nav-link" active-class="nav-link-active" :title="collapsed ? $t('资源概览') : ''">
        <i class="fas fa-gauge-high"></i>
        <span v-if="!collapsed">{{ $t('资源概览') }}</span>
      </router-link>
      <!-- 本机管理折叠子菜单 -->
      <div v-if="portal.hasPerm('GET /api/overview/monitor') || portal.hasPerm('GET /api/filer/files') || portal.hasPerm('GET /api/local/processes') || portal.hasPerm('GET /api/shell')">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('local') }" :title="$t('本机管理')" @click.stop="toggleMenu('local')">
          <i class="fas fa-computer"></i>
        </button>
        <!-- 展开状态：显示完整子菜单 -->
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('local') }" @click.stop="toggleMenu('local')">
            <i class="fas fa-computer"></i>
            <span>{{ $t('本机管理') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('local') }"></i>
          </button>
          <div v-show="expanded.has('local')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link v-if="portal.hasPerm('GET /api/overview/monitor')" to="/local/monitor" class="nav-link" :class="{ 'nav-link-active': isActive('/local/monitor') }">
              <i class="fas fa-desktop"></i>
              <span>{{ $t('系统监控') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/local/processes')" to="/local/process" class="nav-link" :class="{ 'nav-link-active': isActive('/local/process') }">
              <i class="fas fa-microchip"></i>
              <span>{{ $t('进程管理') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/filer/files')" to="/local/explorer" class="nav-link" :class="{ 'nav-link-active': isActive('/local/explorer') }">
              <i class="fas fa-folder-open"></i>
              <span>{{ $t('文件管理') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/shell')" to="/local/shell" class="nav-link" :class="{ 'nav-link-active': isActive('/local/shell') }">
              <i class="fas fa-terminal"></i>
              <span>{{ $t('Shell 终端') }}</span>
            </router-link>
          </div>
        </template>
      </div>
      <!-- SSH/SFTP 折叠子菜单 -->
      <div v-if="portal.hasPerm('GET /api/ssh/hosts') || portal.hasPerm('GET /api/ssh/credentials')">
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('ssh') }" title="SSH / SFTP" @click.stop="toggleMenu('ssh')">
          <i class="fas fa-server"></i>
        </button>
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('ssh') }" @click.stop="toggleMenu('ssh')">
            <i class="fas fa-server"></i>
            <span>SSH / SFTP</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('ssh') }"></i>
          </button>
          <div v-show="expanded.has('ssh')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link v-if="portal.hasPerm('GET /api/ssh/hosts')" to="/ssh/hosts" class="nav-link" :class="{ 'nav-link-active': isActive('/ssh/hosts') || isActive('/ssh/host/') }">
              <i class="fas fa-server"></i>
              <span>{{ $t('主机连接') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/ssh/credentials')" to="/ssh/credentials" class="nav-link" :class="{ 'nav-link-active': isActive('/ssh/credentials') }">
              <i class="fas fa-id-card"></i>
              <span>{{ $t('认证凭据') }}</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- APISIX 折叠子菜单 -->
      <div v-if="apisixLinks.length">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('apisix') }" :title="$t('APISIX 网关')" @click.stop="toggleMenu('apisix')">
          <i class="fas fa-cloud"></i>
        </button>
        <!-- 有权限：展开状态显示完整子菜单 -->
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('apisix') }" @click.stop="toggleMenu('apisix')">
            <i class="fas fa-cloud"></i>
            <span>{{ $t('APISIX 网关') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('apisix') }"></i>
          </button>
          <div v-show="expanded.has('apisix')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link v-for="link in apisixLinks" :key="link.path" :to="link.path" class="nav-link" :class="{ 'nav-link-active': isActive(link.path) }">
              <i :class="['fas', link.icon]"></i>
              <span>{{ $t(link.label) }}</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Caddy 折叠子菜单 -->
      <div v-if="portal.hasPerm('caddy')">
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('caddy') }" :title="$t('Caddy 网关')" @click.stop="toggleMenu('caddy')">
          <i class="fas fa-shield"></i>
        </button>
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('caddy') }" @click.stop="toggleMenu('caddy')">
            <i class="fas fa-shield"></i>
            <span>{{ $t('Caddy 网关') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('caddy') }"></i>
          </button>
          <div v-show="expanded.has('caddy')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link
              v-if="portal.hasPerm('GET /api/caddy/routes')"
              to="/caddy/routes"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/caddy/route') }"
            >
              <i class="fas fa-route"></i>
              <span>{{ $t('路由') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/caddy/servers')" to="/caddy/servers" class="nav-link" :class="{ 'nav-link-active': isActive('/caddy/servers') }">
              <i class="fas fa-server"></i><span>{{ $t('服务') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/caddy/basic-auth')" to="/caddy/basic-auth" class="nav-link" :class="{ 'nav-link-active': isActive('/caddy/basic-auth') }">
              <i class="fas fa-lock"></i>
              <span>{{ $t('基础认证') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/caddy/certs')" to="/caddy/certs" class="nav-link" :class="{ 'nav-link-active': isActive('/caddy/cert') }">
              <i class="fas fa-certificate"></i>
              <span>{{ $t('SSL 证书') }}</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/caddy/global')"
              to="/caddy/global"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/caddy/global') }"
            >
              <i class="fas fa-sliders"></i>
              <span>{{ $t('全局选项') }}</span>
            </router-link>
            <router-link v-if="portal.hasPerm('GET /api/caddy/config')" to="/caddy/raw" class="nav-link" :class="{ 'nav-link-active': isActive('/caddy/raw') }">
              <i class="fas fa-code"></i>
              <span>{{ $t('原始配置') }}</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Docker 折叠子菜单 -->
      <div v-if="portal.hasPerm('docker')">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('docker') }" :title="$t('Docker 服务')" @click.stop="toggleMenu('docker')">
          <i class="fab fa-docker"></i>
        </button>
        <!-- 展开状态：显示完整子菜单 -->
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('docker') }" @click.stop="toggleMenu('docker')">
            <i class="fab fa-docker"></i>
            <span>{{ $t('Docker 服务') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('docker') }"></i>
          </button>
          <div v-show="expanded.has('docker')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link
              v-if="portal.hasPerm('GET /api/docker/containers')"
              to="/docker/containers"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/docker/container') }"
            >
              <i class="fas fa-cube"></i>
              <span>{{ $t('容器') }}</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/networks')"
              to="/docker/networks"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/docker/network') }"
            >
              <i class="fas fa-network-wired"></i>
              <span>{{ $t('网络') }}</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/volumes')"
              to="/docker/volumes"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/docker/volume') }"
            >
              <i class="fas fa-database"></i>
              <span>{{ $t('数据卷') }}</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/images')"
              to="/docker/images"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/docker/image') }"
            >
              <i class="fas fa-layer-group"></i>
              <span>{{ $t('镜像') }}</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/docker/registries')"
              to="/docker/registries"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/docker/registr') }"
            >
              <i class="fas fa-warehouse"></i>
              <span>{{ $t('镜像仓库') }}</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- Swarm 折叠子菜单 -->
      <div v-if="portal.hasPerm('swarm')">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('swarm') }" :title="$t('Swarm 集群')" @click.stop="toggleMenu('swarm')">
          <i class="fas fa-circle-nodes"></i>
        </button>
        <!-- 有权限：展开状态显示完整子菜单 -->
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('swarm') }" @click.stop="toggleMenu('swarm')">
            <i class="fas fa-circle-nodes"></i>
            <span>{{ $t('Swarm 集群') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('swarm') }"></i>
          </button>
          <div v-show="expanded.has('swarm')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link v-if="portal.hasPerm('GET /api/swarm/nodes')" to="/swarm/nodes" class="nav-link" :class="{ 'nav-link-active': isActive('/swarm/node') }">
              <i class="fas fa-server"></i>
              <span>{{ $t('节点') }}</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/swarm/services')"
              to="/swarm/services"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/swarm/service') }"
            >
              <i class="fas fa-cubes"></i>
              <span>{{ $t('服务') }}</span>
            </router-link>
            <router-link
              v-if="portal.hasPerm('GET /api/swarm/tasks')"
              to="/swarm/tasks"
              class="nav-link"
              :class="{ 'nav-link-active': $route.path === '/swarm/tasks' }"
            >
              <i class="fas fa-list-check"></i>
              <span>{{ $t('任务') }}</span>
            </router-link>
          </div>
        </template>
      </div>

      <div v-if="composeVisible">
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('compose') }" :title="$t('Compose 部署')" @click.stop="toggleMenu('compose')">
          <i class="fas fa-file-code"></i>
        </button>
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('compose') }" :aria-expanded="expanded.has('compose')" @click.stop="toggleMenu('compose')">
            <i class="fas fa-file-code"></i>
            <span>{{ $t('Compose 部署') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('compose') }"></i>
          </button>
          <div v-show="expanded.has('compose')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link
              v-if="composeDeployVisible"
              to="/compose/docker"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/compose/docker') }"
            >
              <i class="fab fa-docker"></i>
              <span>{{ $t('单机容器') }}</span>
            </router-link>
            <router-link
              v-if="composeSwarmVisible"
              to="/compose/swarm"
              class="nav-link"
              :class="{ 'nav-link-active': isActive('/compose/swarm') }"
            >
              <i class="fas fa-cubes"></i>
              <span>{{ $t('Swarm 服务') }}</span>
            </router-link>
            <router-link to="/compose/marketplace" class="nav-link" :class="{ 'nav-link-active': isActive('/compose/marketplace') }">
              <i class="fas fa-store"></i>
              <span>{{ $t('应用市场') }}</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- 计划任务 -->
      <router-link v-if="portal.hasPerm('GET /api/cron/jobs')" to="/cron/jobs" class="nav-link" active-class="nav-link-active" :title="collapsed ? $t('计划任务') : ''">
        <i class="fas fa-clock"></i>
        <span v-if="!collapsed">{{ $t('计划任务') }}</span>
      </router-link>

      <!-- 操作审计 -->
      <router-link v-if="portal.hasPerm('GET /api/system/audit/logs')" to="/system/audit/logs" class="nav-link" active-class="nav-link-active" :title="collapsed ? $t('操作审计') : ''">
        <i class="fas fa-clipboard-list"></i>
        <span v-if="!collapsed">{{ $t('操作审计') }}</span>
      </router-link>

      <!-- 用户管理 -->
      <router-link v-if="!portal.currentNodeId && portal.hasPerm('GET /api/account/members')" to="/account/members" class="nav-link" active-class="nav-link-active" :title="collapsed ? $t('用户管理') : ''">
        <i class="fas fa-users"></i>
        <span v-if="!collapsed">{{ $t('用户管理') }}</span>
      </router-link>

      <!-- 受管机折叠子菜单 -->
      <div v-if="nodeVisible">
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('node') }" :title="$t('集中管控')" @click.stop="toggleMenu('node')">
          <i class="fas fa-sitemap"></i>
        </button>
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('node') }" @click.stop="toggleMenu('node')">
            <i class="fas fa-sitemap"></i>
            <span>{{ $t('集中管控') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('node') }"></i>
          </button>
          <div v-show="expanded.has('node')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link v-if="canListNodes" to="/node/access" class="nav-link" :class="{ 'nav-link-active': isActive('/node/access') }">
              <i class="fas fa-server"></i>
              <span>{{ $t('节点管理') }}</span>
            </router-link>
            <router-link v-if="canListNodes" to="/node/portmaps" class="nav-link" :class="{ 'nav-link-active': isActive('/node/portmaps') }">
              <i class="fas fa-right-left"></i>
              <span>{{ $t('端口转发') }}</span>
            </router-link>
          </div>
        </template>
      </div>

      <!-- 系统配置折叠子菜单 -->
      <div v-if="configGroups.length">
        <!-- 折叠状态只显示图标，点击展开侧边栏 -->
        <button v-if="collapsed" type="button" class="nav-link justify-center" :class="{ 'nav-link-active': isMenuActive('config') }" :title="$t('系统配置')" @click.stop="toggleMenu('config')">
          <i class="fas fa-gear"></i>
        </button>
        <template v-else>
          <button type="button" class="nav-link w-full" :class="{ 'nav-link-active': isMenuActive('config') }" @click.stop="toggleMenu('config')">
            <i class="fas fa-gear"></i>
            <span>{{ $t('系统配置') }}</span>
            <i class="fas fa-chevron-down ml-auto text-xs transition-transform duration-200" :class="{ 'rotate-180': expanded.has('config') }"></i>
          </button>
          <div v-show="expanded.has('config')" class="mt-1 ml-4 pl-3 border-l-2 border-slate-200 space-y-1">
            <router-link
              v-for="item in configGroups"
              :key="item.id"
              :to="`/system/config/${item.id}`"
              class="nav-link"
              :class="{ 'nav-link-active': isActive(`/system/config/${item.id}`) }"
            >
              <i class="fas" :class="item.icon"></i>
              <span>{{ $t(item.label) }}</span>
            </router-link>
          </div>
        </template>
      </div>
    </nav>

    <!-- 底部工具条：GitHub 链接 + API 文档 + 折叠按钮 -->
    <div class="border-t border-slate-200/50 p-3" :class="collapsed ? 'space-y-1' : 'flex items-center gap-2'">
      <a
        href="https://github.com/rehiy/isrvd"
        target="_blank"
        rel="noopener noreferrer"
        class="btn-icon btn-icon-slate"
        :class="collapsed ? 'w-full h-10' : 'w-10 h-10 flex-shrink-0'"
        :title="$t('GitHub 仓库')"
      >
        <i class="fab fa-github text-base"></i>
      </a>
      <a
        v-if="portal.openapiEnabled"
        href="openapi/"
        target="_blank"
        rel="noopener noreferrer"
        class="btn-icon btn-icon-slate"
        :class="collapsed ? 'w-full h-10' : 'w-10 h-10 flex-shrink-0'"
        :title="$t('API 文档')"
      >
        <i class="fas fa-code text-base"></i>
      </a>
      <button
        type="button"
        class="btn-icon btn-icon-slate max-lg:hidden"
        :class="collapsed ? 'w-full h-10' : 'flex-1 h-10 text-xs font-medium'"
        :title="collapsed ? $t('展开菜单') : $t('收起菜单')"
        @click="$emit('update:collapsed', !collapsed)"
      >
        <i :class="collapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
        <span v-if="!collapsed">{{ $t('收起菜单') }}</span>
      </button>
    </div>
  </aside>

  <!-- 移动端遮罩层 -->
  <div v-if="mobileSidebarVisible" class="fixed inset-0 bg-black/30 backdrop-blur-sm z-40 lg:hidden" @click="closeMobileSidebar"></div>
</template>
